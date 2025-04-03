package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"
)

const (
	PORT    = "20000" // TCP Port
	API_URL = "http://192.168.1.125:3000/api/speakers"
)

// โครงสร้างข้อมูลจาก API
type Speaker struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	SeatName      string `json:"seatName"`
	Prio          bool   `json:"prio"`
	PrioOn        bool   `json:"prioOn"`
	ParticipantID int    `json:"participantId"`
	MicOn         bool   `json:"micOn"`
}

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

	fmt.Printf("👥 Client %d เชื่อมต่อ: %s\n", client.id, conn.RemoteAddr())
	return client
}

// ลบ client
func (s *Server) RemoveClient(id int) {
	s.clientLock.Lock()
	defer s.clientLock.Unlock()

	if client, exists := s.clients[id]; exists {
		fmt.Printf("👋 Client %d ยกเลิกการเชื่อมต่อ: %s\n", id, client.conn.RemoteAddr())
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
			fmt.Printf("⚠️ ไม่สามารถส่งข้อมูลไปยัง Client %d: %v\n", id, err)
			disconnectedClients = append(disconnectedClients, id)
		}
	}

	// ลบ clients ที่ยกเลิกการเชื่อมต่อ
	for _, id := range disconnectedClients {
		go s.RemoveClient(id)
	}
}

// ฟังก์ชันดึงข้อมูลจาก API
func getSpeakers() ([]Speaker, error) {
	resp, err := http.Get(API_URL)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถเชื่อมต่อกับ API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถอ่านข้อมูลจาก API: %v", err)
	}

	var speakers []Speaker
	if err := json.Unmarshal(body, &speakers); err != nil {
		return nil, fmt.Errorf("ไม่สามารถแปลงข้อมูล JSON: %v", err)
	}

	return speakers, nil
}

// ฟังก์ชันสร้าง XML สำหรับ DiscussionActivity
func generateDiscussionXML(speakers []Speaker) string {
	participantsXML := ""
	for _, speaker := range speakers {
		if speaker.MicOn {
			participantXML := fmt.Sprintf(`<ParticipantContainer Id="%d"><Seat Id="%d"><SeatData Name="%s" MicrophoneActive="true" SeatType="Delegate" IsSpecialStation="false" /><IsReposnding>false</IsReposnding></Seat></ParticipantContainer>`,
				speaker.ParticipantID,
				speaker.ID,
				speaker.SeatName,
			)
			participantsXML += participantXML
		}
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="80"><ActiveList><Participants>%s</Participants></ActiveList></Discussion></DiscussionActivity>`,
		time.Now().Format("2006-01-02T15:04:05"),
		participantsXML,
	)
}

// ฟังก์ชันสร้าง XML สำหรับ SeatActivity
func generateSeatXML(speaker Speaker, micState bool) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><SeatActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Seat" Type="SeatUpdated"><Seat Id="%d"><SeatData Name="%s" MicrophoneActive="%v" SeatType="Delegate" IsSpecialStation="false" /><Participant Id="%d"><ParticipantData Present="false" VotingWeight="1" VotingAuthorisation="true" MicrophoneAuthorisation="true" FirstName="" MiddleName="" LastName="%s" Title="" Country="" RemainingSpeechTime="-1" SpeechTimerOnHold="false" /></Participant><IsReposnding>false</IsReposnding></Seat></SeatActivity>`,
		time.Now().Format("2006-01-02T15:04:05"),
		speaker.ID,
		speaker.SeatName,
		micState,
		speaker.ParticipantID,
		speaker.SeatName,
	)
}

// ฟังก์ชันดึงข้อมูลจาก API และส่งไปยัง clients
func (s *Server) ProcessAndBroadcast() {
	var lastSpeakers []Speaker
	speakerStates := make(map[int]bool)

	for {
		speakers, err := getSpeakers()
		if err != nil {
			fmt.Println("⚠️ ไม่สามารถดึงข้อมูล speakers:", err)
			// ส่ง XML ว่างเมื่อไม่มีข้อมูลจาก API
			emptyXML := `<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="` + time.Now().Format("2006-01-02T15:04:05") + `" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="80"><ActiveList><Participants></Participants></ActiveList></Discussion></DiscussionActivity>`
			header := make([]byte, 8)
			binary.LittleEndian.PutUint32(header[0:4], 3)
			binary.LittleEndian.PutUint32(header[4:8], uint32(len(emptyXML)))
			s.Broadcast(append(header, []byte(emptyXML)...))
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
				seatXML := generateSeatXML(speaker, speaker.MicOn)
				header := make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 5)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
				s.Broadcast(append(header, []byte(seatXML)...))
				speakerStates[speaker.ID] = speaker.MicOn
			}
		}

		// ตรวจสอบที่นั่งที่หายไป
		for id, state := range speakerStates {
			if !currentSpeakerIDs[id] && state {
				// สร้าง speaker ข้อมูลเดิมแต่ปิดไมค์
				for _, oldSpeaker := range lastSpeakers {
					if oldSpeaker.ID == id {
						seatXML := generateSeatXML(oldSpeaker, false)
						header := make([]byte, 8)
						binary.LittleEndian.PutUint32(header[0:4], 5)
						binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
						s.Broadcast(append(header, []byte(seatXML)...))
						speakerStates[id] = false
						break
					}
				}
			}
		}

		// ส่ง DiscussionActivity เมื่อรายการที่นั่งเปลี่ยน
		if !reflect.DeepEqual(speakers, lastSpeakers) {
			discussionXML := generateDiscussionXML(speakers)
			header := make([]byte, 8)
			binary.LittleEndian.PutUint32(header[0:4], 3)
			binary.LittleEndian.PutUint32(header[4:8], uint32(len(discussionXML)))
			s.Broadcast(append(header, []byte(discussionXML)...))
			lastSpeakers = speakers
		}

		time.Sleep(time.Second)
	}
}

// จัดการการเชื่อมต่อจาก client
func handleClientConnection(server *Server, conn net.Conn) {
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

func main() {
	// สร้าง server
	server := NewServer()

	// เริ่ม server
	listener, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		fmt.Printf("❌ ไม่สามารถเริ่ม server ได้: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("🚀 Server กำลังทำงานที่พอร์ต %s\n", PORT)

	// เริ่มการประมวลผลและส่งข้อมูล
	go server.ProcessAndBroadcast()

	// รับการเชื่อมต่อจาก clients
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("⚠️ ไม่สามารถรับการเชื่อมต่อจาก client ได้: %v\n", err)
			continue
		}
		go handleClientConnection(server, conn)
	}
}
