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
	"time"
)

const (
	PORT    = "20000" // TCP Port
	API_URL = "http://10.106.0.30:3000/api/speakers"
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

// ฟังก์ชันส่ง XML ไปยัง client
func sendXML(conn net.Conn, topic uint32, xmlData string) error {
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], topic)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(xmlData)))

	if _, err := conn.Write(header); err != nil {
		return fmt.Errorf("ไม่สามารถส่ง header: %v", err)
	}
	if _, err := conn.Write([]byte(xmlData)); err != nil {
		return fmt.Errorf("ไม่สามารถส่ง XML: %v", err)
	}
	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("📡 Client Connected:", conn.RemoteAddr())

	var lastSpeakers []Speaker
	speakerStates := make(map[int]bool)

	for {
		speakers, err := getSpeakers()
		if err != nil {
			fmt.Println("⚠️ ไม่สามารถดึงข้อมูล speakers:", err)
			// ส่ง XML ว่างเมื่อไม่มีข้อมูลจาก API
			emptyXML := `<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="` + time.Now().Format("2006-01-02T15:04:05") + `" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="80"><ActiveList><Participants></Participants></ActiveList></Discussion></DiscussionActivity>`
			if err := sendXML(conn, 3, emptyXML); err != nil {
				fmt.Printf("❌ ไม่สามารถส่ง Discussion Activity: %v\n", err)
				return
			}
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
				if err := sendXML(conn, 5, seatXML); err != nil {
					fmt.Printf("❌ ไม่สามารถส่ง Seat Activity: %v\n", err)
					return
				}
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
						if err := sendXML(conn, 5, seatXML); err != nil {
							fmt.Printf("❌ ไม่สามารถส่ง Seat Activity: %v\n", err)
							return
						}
						speakerStates[id] = false
						break
					}
				}
			}
		}

		// ส่ง DiscussionActivity เมื่อรายการที่นั่งเปลี่ยน
		if !reflect.DeepEqual(speakers, lastSpeakers) {
			discussionXML := generateDiscussionXML(speakers)
			if err := sendXML(conn, 3, discussionXML); err != nil {
				fmt.Printf("❌ ไม่สามารถส่ง Discussion Activity: %v\n", err)
				return
			}
			lastSpeakers = speakers
		}

		time.Sleep(time.Second)
	}
}

func main() {
	listener, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		fmt.Println("⚠️ Failed to start server:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("🚀 XML TCP Server running on port", PORT)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("⚠️ Connection error:", err)
			continue
		}

		go handleConnection(conn)
	}
}
