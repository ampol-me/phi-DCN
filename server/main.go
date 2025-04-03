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
	"unicode/utf16"
)

const (
	PORT     = "20000" // TCP Port
	API_URL  = "http://192.168.1.125:3000/api/speakers"
	USE_MOCK = true // true = ใช้ข้อมูล mock, false = ใช้ API จริง
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

var (
	mockMicState = true
	lastToggle   = time.Now()
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

// ฟังก์ชันจำลองข้อมูล API
func getMockSpeakers() ([]Speaker, error) {
	// สลับสถานะไมค์ทุก 5 วินาที
	if time.Since(lastToggle) >= 5*time.Second {
		mockMicState = !mockMicState
		lastToggle = time.Now()
		fmt.Printf("🔄 สลับสถานะไมค์เป็น: %v\n", mockMicState)
	}

	return []Speaker{
		{
			ID:            3539,
			Name:          "A05",
			SeatName:      "A05",
			Prio:          false,
			PrioOn:        false,
			ParticipantID: 0,
			MicOn:         mockMicState,
		},
	}, nil
}

// ฟังก์ชันดึงข้อมูล (เลือกระหว่าง API จริงหรือ mock)
func getSpeakers() ([]Speaker, error) {
	if USE_MOCK {
		return getMockSpeakers()
	}

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

// เพิ่มฟังก์ชันสำหรับแปลง string เป็น UTF-16LE with BOM
func toUTF16LEString(s string) []byte {
	// แปลงเป็น UTF-16LE
	u16 := utf16.Encode([]rune(s))

	// สร้าง buffer สำหรับเก็บ BOM และข้อมูล
	bytes := make([]byte, 2+len(u16)*2)

	// ใส่ BOM (0xFF 0xFE สำหรับ UTF-16LE)
	bytes[0] = 0xFF
	bytes[1] = 0xFE

	// แปลง []uint16 เป็น []byte
	for i, v := range u16 {
		bytes[2+i*2] = byte(v)
		bytes[2+i*2+1] = byte(v >> 8)
	}

	return bytes
}

// แก้ไขฟังก์ชัน generateDiscussionXML
func generateDiscussionXML(speakers []Speaker) []byte {
	// ตรวจสอบว่ามีไมค์ที่เปิดอยู่หรือไม่
	hasMicOn := false
	participantsXML := ""

	for _, speaker := range speakers {
		if speaker.MicOn {
			hasMicOn = true
			participantXML := fmt.Sprintf(`<ParticipantContainer Id="%d"><Seat Id="%d"><SeatData Name="%s" MicrophoneActive="true" SeatType="Delegate" IsSpecialStation="false" /><IsReposnding>false</IsReposnding></Seat></ParticipantContainer>`,
				speaker.ParticipantID,
				speaker.ID,
				speaker.SeatName,
			)
			participantsXML += participantXML
		}
	}

	var xmlStr string
	if !hasMicOn {
		xmlStr = fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="71"><ActiveList></ActiveList></Discussion></DiscussionActivity>`,
			time.Now().Format("2006-01-02T15:04:05.0000000-07:00"),
		)
	} else {
		xmlStr = fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="71"><ActiveList><Participants>%s</Participants></ActiveList></Discussion></DiscussionActivity>`,
			time.Now().Format("2006-01-02T15:04:05.0000000-07:00"),
			participantsXML,
		)
	}

	return toUTF16LEString(xmlStr)
}

// แก้ไขฟังก์ชัน generateSeatXML
func generateSeatXML(speaker Speaker, micState bool) []byte {
	xmlStr := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><SeatActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Seat" Type="SeatUpdated"><Seat Id="%d"><SeatData Name="%s" MicrophoneActive="%v" SeatType="Delegate" IsSpecialStation="false" /><Participant Id="%d"><ParticipantData Present="false" VotingWeight="1" VotingAuthorisation="true" MicrophoneAuthorisation="true" FirstName="" MiddleName="" LastName="%s" Title="" Country="" RemainingSpeechTime="-1" SpeechTimerOnHold="false" /></Participant><IsReposnding>false</IsReposnding></Seat></SeatActivity>`,
		time.Now().Format("2006-01-02T15:04:05.0000000-07:00"),
		speaker.ID,
		speaker.SeatName,
		micState,
		speaker.ParticipantID,
		speaker.SeatName,
	)

	return toUTF16LEString(xmlStr)
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
			emptyXML := toUTF16LEString(fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="80"><ActiveList><Participants></Participants></ActiveList></Discussion></DiscussionActivity>`,
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
				seatXML := generateSeatXML(speaker, speaker.MicOn)
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
						seatXML := generateSeatXML(oldSpeaker, false)
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
			discussionXML := generateDiscussionXML(speakers)
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
