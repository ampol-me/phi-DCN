package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"unicode/utf16"
)

const (
	SERVER_HOST     = "172.20.1.229" // IP address ของ Bosch DCN server
	SERVER_PORT     = "20000"        // พอร์ตของ server
	CONNECT_TIMEOUT = 5              // timeout การเชื่อมต่อ (วินาที)
	READ_TIMEOUT    = 10             // timeout การรับข้อมูล (วินาที)
)

// ฟังก์ชันสำหรับถอดรหัส header ที่เข้ารหัสมาแล้ว
func decodeHeader(encodedHeader string) (topic uint32, messageLength uint32, err error) {
	// แปลงสตริงเป็น []byte
	headerBytes := []byte(encodedHeader)

	// ตรวจสอบความยาวของ header
	if len(headerBytes) < 8 {
		return 0, 0, fmt.Errorf("header สั้นเกินไป: ต้องการ 8 bytes แต่ได้ %d bytes", len(headerBytes))
	}

	// ถอดรหัส topic และ messageLength
	topic = binary.LittleEndian.Uint32(headerBytes[0:4])
	messageLength = binary.LittleEndian.Uint32(headerBytes[4:8])

	return topic, messageLength, nil
}

// ฟังก์ชันจัดรูปแบบ XML ให้สวยงาม
func prettyXML(xmlStr string) []string {
	// แยก XML strings ด้วย <?xml
	xmlParts := bytes.Split([]byte(xmlStr), []byte("<?xml"))

	var results []string

	for _, part := range xmlParts {
		// ข้าม empty parts
		if len(bytes.TrimSpace(part)) == 0 {
			continue
		}

		// เติม <?xml กลับไป
		if !bytes.HasPrefix(part, []byte("<?xml")) {
			part = append([]byte("<?xml"), part...)
		}

		// สร้าง decoder สำหรับอ่าน XML
		decoder := xml.NewDecoder(bytes.NewReader(part))

		// สร้าง buffer สำหรับเก็บผลลัพธ์
		var prettyXML bytes.Buffer

		// สร้าง encoder ที่จะเขียนลงใน buffer พร้อมกำหนด indent
		encoder := xml.NewEncoder(&prettyXML)
		encoder.Indent("", "  ")

		// อ่านและเขียน XML token แต่ละตัว
		for {
			token, err := decoder.Token()
			if err != nil {
				break
			}
			if token == nil {
				break
			}

			err = encoder.EncodeToken(token)
			if err != nil {
				break
			}
		}

		// Flush encoder
		encoder.Flush()

		// เพิ่มผลลัพธ์ที่จัดรูปแบบแล้วเข้าไปใน array
		if prettyXML.Len() > 0 {
			results = append(results, prettyXML.String())
		}
	}

	return results
}

// ฟังก์ชันอ่านข้อมูลจนครบตามจำนวนที่ต้องการ
func readFull(conn net.Conn, buffer []byte) error {
	bytesRead := 0
	for bytesRead < len(buffer) {
		n, err := conn.Read(buffer[bytesRead:])
		if err != nil {
			return err
		}
		bytesRead += n
	}
	return nil
}

// โครงสร้างข้อมูล XML
type SeatActivity struct {
	XMLName xml.Name `xml:"SeatActivity"`
	Seat    struct {
		ID       string `xml:"Id,attr"`
		SeatData struct {
			Name             string `xml:"Name,attr"`
			MicrophoneActive bool   `xml:"MicrophoneActive,attr"`
		} `xml:"SeatData"`
	} `xml:"Seat"`
}

type DiscussionActivity struct {
	XMLName    xml.Name `xml:"DiscussionActivity"`
	Discussion struct {
		ActiveList struct {
			Participants struct {
				ParticipantContainers []struct {
					Seat struct {
						SeatData struct {
							Name             string `xml:"Name,attr"`
							MicrophoneActive bool   `xml:"MicrophoneActive,attr"`
						} `xml:"SeatData"`
					} `xml:"Seat"`
				} `xml:"ParticipantContainer"`
			} `xml:"Participants"`
		} `xml:"ActiveList"`
	} `xml:"Discussion"`
}

// แปลง XML เป็นข้อความสถานะที่เข้าใจง่าย
func parseXMLStatus(xmlStr string, topic uint32) string {
	switch topic {
	case 3: // Discussion Activity
		var discussion DiscussionActivity
		if err := xml.Unmarshal([]byte(xmlStr), &discussion); err == nil {
			var status strings.Builder
			status.WriteString("\n🎙️ สถานะไมค์ทั้งหมด:")
			for _, participant := range discussion.Discussion.ActiveList.Participants.ParticipantContainers {
				micStatus := "🔴 ปิด"
				if participant.Seat.SeatData.MicrophoneActive {
					micStatus = "🟢 เปิด"
				}
				status.WriteString(fmt.Sprintf("\n   %s: %s", participant.Seat.SeatData.Name, micStatus))
			}
			return status.String()
		}
	case 5: // Seat Activity
		var seat SeatActivity
		if err := xml.Unmarshal([]byte(xmlStr), &seat); err == nil {
			micStatus := "🔴 ปิด"
			if seat.Seat.SeatData.MicrophoneActive {
				micStatus = "🟢 เปิด"
			}
			return fmt.Sprintf("\n🎙️ การเปลี่ยนแปลง: %s %s", seat.Seat.SeatData.Name, micStatus)
		}
	}
	return ""
}

// ฟังก์ชันสร้าง header
func createHeader(topic uint32, length uint32) []byte {
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], topic)
	binary.LittleEndian.PutUint32(header[4:8], length)
	return header
}

// โครงสร้างสำหรับเก็บข้อมูล header และ message
type MessageData struct {
	topic   uint32
	length  uint32
	message []byte
}

func utf16LEToString(b []byte) string {
	// แปลงจาก bytes เป็น uint16 (UTF-16LE)
	utf16Words := make([]uint16, len(b)/2)
	for i := 0; i < len(b)/2; i++ {
		utf16Words[i] = uint16(b[i*2]) | uint16(b[i*2+1])<<8
	}
	// แปลงจาก UTF-16 เป็น string
	return string(utf16.Decode(utf16Words))
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Printf("🔗 เชื่อมต่อกับ %s:%s สำเร็จ\n", SERVER_HOST, SERVER_PORT)
	fmt.Println("📡 กำลังรอรับข้อมูล...")

	// สร้าง buffer สำหรับเก็บข้อมูลที่เหลือ
	remainingData := make([]byte, 0)

	for {
		// อ่านข้อมูลใหม่เข้ามาในบัฟเฟอร์
		buffer := make([]byte, 4096)
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("⚠️ การเชื่อมต่อถูกปิด: %v\n", err)
			return
		}

		// รวมข้อมูลที่เหลือจากรอบที่แล้วกับข้อมูลใหม่
		data := append(remainingData, buffer[:n]...)

		// วนลูปอ่านข้อมูลจนกว่าจะไม่พอ
		for len(data) >= 8 { // ต้องมีอย่างน้อย 8 bytes สำหรับ header
			// แสดงข้อมูลดิบ 16 bytes แรกเพื่อดีบัก
			debugLen := 16
			if len(data) < debugLen {
				debugLen = len(data)
			}
			fmt.Printf("📝 Raw data: [% x]\n", data[:debugLen])

			// ตรวจสอบว่าเป็น header หรือไม่
			if data[0] == 0x05 || data[0] == 0x03 { // ถ้าเป็น topic 5 หรือ 3
				// อ่าน header
				topic := uint32(data[0])
				length := binary.LittleEndian.Uint32(data[4:8])

				fmt.Printf("📨 พบ Header - Topic: %d, Length: %d bytes\n", topic, length)

				// ตรวจสอบว่ามีข้อมูล XML ครบหรือไม่
				if len(data) < 8+int(length) {
					// ถ้าข้อมูลไม่พอ เก็บไว้รอรอบหน้า
					remainingData = data
					break
				}

				// อ่าน XML message
				xmlMessage := data[8 : 8+length]

				// แปลง UTF-16LE เป็น UTF-8 ถ้าจำเป็น
				xmlStr := string(xmlMessage)
				if length > 2 && xmlMessage[0] == 0xFF && xmlMessage[1] == 0xFE {
					// ข้าม BOM (2 bytes) และแปลงเป็น UTF-8
					xmlStr = utf16LEToString(xmlMessage[2:])
				}

				// แสดงผล XML
				topicName := "Unknown"
				switch topic {
				case 3:
					topicName = "Discussion Activity"
				case 5:
					topicName = "Seat Activity"
				}

				// แยกและจัดรูปแบบ XML
				formattedXMLs := prettyXML(xmlStr)
				for _, xml := range formattedXMLs {
					status := parseXMLStatus(xml, topic)
					if status != "" {
						fmt.Println(status)
					}
					fmt.Printf("\n📜 Topic: %d (%s)\n%s\n", topic, topicName, xml)
				}

				fmt.Println(strings.Repeat("-", 80))

				// เลื่อนตำแหน่งข้อมูลไปข้างหน้า
				data = data[8+length:]
			} else {
				// ถ้าไม่ใช่ header ที่ถูกต้อง เลื่อนไป 1 byte
				data = data[1:]
			}
		}

		// เก็บข้อมูลที่เหลือไว้
		remainingData = data
	}
}

func main() {
	// เชื่อมต่อไปยัง server
	serverAddr := fmt.Sprintf("%s:%s", SERVER_HOST, SERVER_PORT)
	fmt.Printf("🔄 กำลังเชื่อมต่อไปยัง %s...\n", serverAddr)

	// สร้าง dialer พร้อม timeout
	dialer := net.Dialer{
		Timeout: time.Duration(CONNECT_TIMEOUT) * time.Second,
	}

	conn, err := dialer.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("❌ ไม่สามารถเชื่อมต่อกับ server ได้: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	handleConnection(conn)
}
