package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"net"
	"os"
)

const (
	SERVER_PORT = "20000" // พอร์ตของ server ที่จะเชื่อมต่อไป
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
func prettyXML(xmlStr string) string {
	// สร้าง decoder สำหรับอ่าน XML
	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlStr)))

	// สร้าง buffer สำหรับเก็บผลลัพธ์
	var prettyXML bytes.Buffer

	// สร้าง encoder ที่จะเขียนลงใน buffer พร้อมกำหนด indent
	encoder := xml.NewEncoder(&prettyXML)
	encoder.Indent("", "  ")

	// อ่านและเขียน XML token แต่ละตัว
	for {
		token, err := decoder.Token()
		if err != nil {
			// ถ้าอ่านไม่ได้ให้คืนค่า XML เดิม
			return xmlStr
		}
		if token == nil {
			break
		}

		err = encoder.EncodeToken(token)
		if err != nil {
			return xmlStr
		}
	}

	// Flush encoder เพื่อให้แน่ใจว่าข้อมูลทั้งหมดถูกเขียนลงใน buffer
	err := encoder.Flush()
	if err != nil {
		return xmlStr
	}

	return prettyXML.String()
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		// อ่าน header (8 bytes)
		header := make([]byte, 8)
		_, err := conn.Read(header)
		if err != nil {
			fmt.Println("⚠️ ไม่สามารถอ่าน header:", err)
			return
		}

		// แยก topic และ length จาก header
		topic := binary.LittleEndian.Uint32(header[0:4])
		length := binary.LittleEndian.Uint32(header[4:8])

		// อ่านข้อความ XML
		message := make([]byte, length)
		_, err = conn.Read(message)
		if err != nil {
			fmt.Println("⚠️ ไม่สามารถอ่านข้อความ:", err)
			return
		}

		// แสดงผล XML ในรูปแบบที่สวยงาม
		xmlMessage := string(message)

		// แสดงชื่อ Topic ตามความหมาย
		topicName := "Unknown"
		switch topic {
		case 3:
			topicName = "Discussion Activity"
		case 5:
			topicName = "Seat Activity"
		}

		fmt.Printf("\n📜 Topic: %d (%s)\n%s\n", topic, topicName, prettyXML(xmlMessage))
	}
}

func main() {
	// เชื่อมต่อไปยัง server
	conn, err := net.Dial("tcp", "localhost:"+SERVER_PORT)
	if err != nil {
		fmt.Println("⚠️ ไม่สามารถเชื่อมต่อกับ server ได้:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("🔗 เชื่อมต่อกับ server สำเร็จ")

	handleConnection(conn)
}
