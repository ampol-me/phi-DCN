package server

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"reflect"
	"sync"
	"time"

	"phi-DCN/client/api"
	"phi-DCN/client/xml"
)

// Client เก็บข้อมูลของ client ที่เชื่อมต่อ
type Client struct {
	conn net.Conn
	id   int
}

// Server จัดการการเชื่อมต่อของ clients
type Server struct {
	clients    map[int]*Client
	nextID     int
	clientLock sync.Mutex
}

// สร้าง Server ใหม่
func NewServer() *Server {
	return &Server{
		clients: make(map[int]*Client),
		nextID:  1,
	}
}

// เพิ่ม client ใหม่
func (s *Server) AddClient(conn net.Conn) *Client {
	s.clientLock.Lock()
	defer s.clientLock.Unlock()

	client := &Client{
		conn: conn,
		id:   s.nextID,
	}
	s.clients[s.nextID] = client
	s.nextID++

	fmt.Printf("👥 Client %d connected: %s\n", client.id, conn.RemoteAddr())
	return client
}

// ลบ client
func (s *Server) RemoveClient(id int) {
	s.clientLock.Lock()
	defer s.clientLock.Unlock()

	if client, exists := s.clients[id]; exists {
		fmt.Printf("👋 Client %d disconnected: %s\n", id, client.conn.RemoteAddr())
		client.conn.Close()
		delete(s.clients, id)
	}
}

// ส่งข้อมูลไปยังทุก clients
func (s *Server) Broadcast(data []byte) {
	s.clientLock.Lock()
	defer s.clientLock.Unlock()

	disconnectedClients := []int{}

	for id, client := range s.clients {
		_, err := client.conn.Write(data)
		if err != nil {
			fmt.Printf("⚠️ Cannot send data to Client %d: %v\n", id, err)
			disconnectedClients = append(disconnectedClients, id)
		}
	}

	// ลบ clients ที่ยกเลิกการเชื่อมต่อ
	for _, id := range disconnectedClients {
		go s.RemoveClient(id)
	}
}

// ฟังก์ชันดึงข้อมูลจาก API และส่งไปยัง clients
func (s *Server) ProcessAndBroadcast() {
	var lastSpeakers []api.Speaker
	speakerStates := make(map[int]bool)

	for {
		speakers, err := api.GetSpeakers()
		if err != nil {
			fmt.Println("⚠️ Cannot fetch speakers data:", err)
			// ส่ง XML ว่างเมื่อไม่มีข้อมูลจาก API
			emptyXML := xml.ToUTF16LEString(fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="80"><ActiveList><Participants></Participants></ActiveList></Discussion></DiscussionActivity>`,
				time.Now().Format("2006-01-02T15:04:05.0000000-07:00")))

			header := make([]byte, 8)
			binary.LittleEndian.PutUint32(header[0:4], 3)
			binary.LittleEndian.PutUint32(header[4:8], uint32(len(emptyXML)))
			s.Broadcast(append(header, emptyXML...))
			time.Sleep(time.Second)
			continue
		}

		// ตรวจสอบการเปลี่ยนแปลงของแต่ละที่นั่ง
		currentSpeakerIDs := make(map[int]bool)
		for _, speaker := range speakers {
			currentSpeakerIDs[speaker.ID] = true

			// ตรวจสอบการเปลี่ยนแปลงสถานะไมค์
			lastState, exists := speakerStates[speaker.ID]
			if !exists || lastState != speaker.MicOn {
				// ส่ง SeatActivity เมื่อสถานะเปลี่ยน
				seatXML := xml.GenerateSeatXML(speaker, speaker.MicOn)
				header := make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 5)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
				s.Broadcast(append(header, seatXML...))
				speakerStates[speaker.ID] = speaker.MicOn
			}
		}

		// ตรวจสอบที่นั่งที่หายไป
		for id, state := range speakerStates {
			if !currentSpeakerIDs[id] && state {
				// สร้าง speaker ข้อมูลเดิมแต่ปิดไมค์
				for _, oldSpeaker := range lastSpeakers {
					if oldSpeaker.ID == id {
						seatXML := xml.GenerateSeatXML(oldSpeaker, false)
						header := make([]byte, 8)
						binary.LittleEndian.PutUint32(header[0:4], 5)
						binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
						s.Broadcast(append(header, seatXML...))
						speakerStates[id] = false
						break
					}
				}
			}
		}

		// ส่ง DiscussionActivity เมื่อรายการที่นั่งเปลี่ยน
		if !reflect.DeepEqual(speakers, lastSpeakers) {
			discussionXML := xml.GenerateDiscussionXML(speakers)
			header := make([]byte, 8)
			binary.LittleEndian.PutUint32(header[0:4], 3)
			binary.LittleEndian.PutUint32(header[4:8], uint32(len(discussionXML)))
			s.Broadcast(append(header, discussionXML...))
			lastSpeakers = speakers
		}

		time.Sleep(time.Second)
	}
}

// จัดการการเชื่อมต่อจาก client
func HandleClientConnection(server *Server, conn net.Conn) {
	client := server.AddClient(conn)
	defer server.RemoveClient(client.id)

	// รอจนกว่า client จะยกเลิกการเชื่อมต่อ
	buffer := make([]byte, 1024)
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			return
		}
	}
}

// เริ่มทำงาน Server
func StartServer(port string) {
	// สร้าง server
	server := NewServer()

	// เริ่ม server
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("❌ Cannot start server: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("🚀 Server running on port %s\n", port)

	// เริ่มการประมวลผลและส่งข้อมูล
	go server.ProcessAndBroadcast()

	// รับการเชื่อมต่อจาก clients
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("⚠️ Cannot accept client connection: %v\n", err)
			continue
		}
		go HandleClientConnection(server, conn)
	}
}
