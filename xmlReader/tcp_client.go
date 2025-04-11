package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"os"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

// TopicNames แปลงรหัส topic เป็นชื่อ
var TopicNames = map[uint32]string{
	3: "DiscussionActivity",
	5: "SeatActivity",
}

// Header โครงสร้างข้อมูล header
type Header struct {
	Topic  uint32
	Length uint32
}

// DiscussionActivity โครงสร้างข้อมูล XML
type DiscussionActivity struct {
	XMLName    xml.Name `xml:"DiscussionActivity"`
	Version    string   `xml:"Version,attr"`
	TimeStamp  string   `xml:"TimeStamp,attr"`
	Topic      string   `xml:"Topic,attr"`
	Type       string   `xml:"Type,attr"`
	Discussion struct {
		ID         string `xml:"Id,attr"`
		ActiveList struct {
			Participants struct {
				ParticipantContainer []struct {
					ID   string `xml:"Id,attr"`
					Seat struct {
						ID       string `xml:"Id,attr"`
						SeatData struct {
							Name             string `xml:"Name,attr"`
							MicrophoneActive string `xml:"MicrophoneActive,attr"`
							SeatType         string `xml:"SeatType,attr"`
							IsSpecialStation string `xml:"IsSpecialStation,attr"`
						} `xml:"SeatData"`
						IsResponding string `xml:"IsReposnding"`
					} `xml:"Seat"`
				} `xml:"ParticipantContainer"`
			} `xml:"Participants"`
		} `xml:"ActiveList"`
	} `xml:"Discussion"`
}

// SeatActivity โครงสร้างข้อมูล XML
type SeatActivity struct {
	XMLName   xml.Name `xml:"SeatActivity"`
	Version   string   `xml:"Version,attr"`
	TimeStamp string   `xml:"TimeStamp,attr"`
	Topic     string   `xml:"Topic,attr"`
	Type      string   `xml:"Type,attr"`
	Seat      struct {
		ID       string `xml:"Id,attr"`
		SeatData struct {
			Name             string `xml:"Name,attr"`
			MicrophoneActive string `xml:"MicrophoneActive,attr"`
			SeatType         string `xml:"SeatType,attr"`
			IsSpecialStation string `xml:"IsSpecialStation,attr"`
		} `xml:"SeatData"`
		Participant struct {
			ID              string `xml:"Id,attr"`
			ParticipantData struct {
				Present                 string `xml:"Present,attr"`
				VotingWeight            string `xml:"VotingWeight,attr"`
				VotingAuthorisation     string `xml:"VotingAuthorisation,attr"`
				MicrophoneAuthorisation string `xml:"MicrophoneAuthorisation,attr"`
				FirstName               string `xml:"FirstName,attr"`
				MiddleName              string `xml:"MiddleName,attr"`
				LastName                string `xml:"LastName,attr"`
				Title                   string `xml:"Title,attr"`
				Country                 string `xml:"Country,attr"`
				RemainingSpeechTime     string `xml:"RemainingSpeechTime,attr"`
				SpeechTimerOnHold       string `xml:"SpeechTimerOnHold,attr"`
			} `xml:"ParticipantData"`
		} `xml:"Participant"`
		IsResponding string `xml:"IsReposnding"`
	} `xml:"Seat"`
}

func utf16ToUtf8(data []byte) (string, error) {
	if len(data) < 2 {
		return "", fmt.Errorf("ข้อมูลสั้นเกินไป")
	}

	// ตรวจสอบ BOM
	if data[0] == 0xFF && data[1] == 0xFE {
		data = data[2:] // ตัด BOM ออก
	}

	// แปลง bytes เป็น uint16
	u16 := make([]uint16, len(data)/2)
	for i := 0; i < len(u16); i++ {
		u16[i] = binary.LittleEndian.Uint16(data[i*2:])
	}

	// แปลง UTF-16 เป็น UTF-8
	runes := utf16.Decode(u16)
	buf := make([]byte, len(runes)*utf8.UTFMax)
	n := 0
	for _, r := range runes {
		n += utf8.EncodeRune(buf[n:], r)
	}
	return string(buf[:n]), nil
}

func main() {
	// รับค่า config จากผู้ใช้
	fmt.Println("=== การตั้งค่า TCP Client ===")
	var host, port string
	fmt.Print("กรุณาใส่ IP ของเซิร์ฟเวอร์ (default: localhost): ")
	fmt.Scanln(&host)
	if host == "" {
		host = "localhost"
	}

	fmt.Print("กรุณาใส่พอร์ตของเซิร์ฟเวอร์ (default: 20000): ")
	fmt.Scanln(&port)
	if port == "" {
		port = "20000"
	}

	// เชื่อมต่อกับเซิร์ฟเวอร์
	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		fmt.Printf("ไม่สามารถเชื่อมต่อกับเซิร์ฟเวอร์ได้: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("เชื่อมต่อกับเซิร์ฟเวอร์ที่ %s:%s สำเร็จ!\n", host, port)
	fmt.Println("รอรับข้อมูล...\n")

	buffer := make([]byte, 0)
	for {
		// รับข้อมูลจากเซิร์ฟเวอร์
		data := make([]byte, 4096)
		n, err := conn.Read(data)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("เกิดข้อผิดพลาดในการรับข้อมูล: %v\n", err)
			}
			break
		}

		buffer = append(buffer, data[:n]...)

		// ตรวจสอบว่ามีข้อมูลเพียงพอที่จะแยก header
		if len(buffer) >= 8 {
			// อ่าน header
			var header Header
			reader := bytes.NewReader(buffer[:8])
			binary.Read(reader, binary.LittleEndian, &header.Topic)
			binary.Read(reader, binary.LittleEndian, &header.Length)

			// ตรวจสอบว่ามีข้อมูล XML ทั้งหมดแล้วหรือยัง
			if len(buffer) >= int(8+header.Length) {
				xmlData := buffer[8 : 8+header.Length]
				buffer = buffer[8+header.Length:]

				// แปลง UTF-16 เป็น UTF-8
				xmlString, err := utf16ToUtf8(xmlData)
				if err != nil {
					fmt.Printf("ไม่สามารถแปลงข้อมูลเป็น UTF-8 ได้: %v\n", err)
					continue
				}

				// แสดงผล
				topicName := TopicNames[header.Topic]
				if topicName == "" {
					topicName = fmt.Sprintf("Unknown Topic (%d)", header.Topic)
				}

				fmt.Printf("\n📨 ข้อมูลที่ได้รับ (Topic: %d - %s):\n", header.Topic, topicName)
				fmt.Println("----------------------------------------")

				// แยกการประมวลผลตาม topic
				switch header.Topic {
				case 3: // DiscussionActivity
					var activity DiscussionActivity
					err = xml.Unmarshal([]byte(xmlString), &activity)
					if err != nil {
						fmt.Printf("ไม่สามารถแปลงข้อมูลเป็น XML ได้: %v\n", err)
						fmt.Printf("ข้อมูลดิบ: %x\n", xmlData)
						continue
					}
					output, _ := xml.MarshalIndent(activity, "  ", "    ")
					fmt.Println(string(output))

				case 5: // SeatActivity
					var activity SeatActivity
					err = xml.Unmarshal([]byte(xmlString), &activity)
					if err != nil {
						fmt.Printf("ไม่สามารถแปลงข้อมูลเป็น XML ได้: %v\n", err)
						fmt.Printf("ข้อมูลดิบ: %x\n", xmlData)
						continue
					}
					output, _ := xml.MarshalIndent(activity, "  ", "    ")
					fmt.Println(string(output))

				default:
					fmt.Println(xmlString)
				}

				fmt.Println("----------------------------------------")
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}
