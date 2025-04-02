package main

import (
	"encoding/binary"
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

func main() {
	// เชื่อมต่อไปยัง server
	conn, err := net.Dial("tcp", "localhost:"+SERVER_PORT)
	if err != nil {
		fmt.Println("⚠️ ไม่สามารถเชื่อมต่อกับ server ได้:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("🔗 เชื่อมต่อกับ server สำเร็จ")

	for {
		// อ่าน header (8 bytes)
		header := make([]byte, 8)
		_, err := conn.Read(header)
		if err != nil {
			fmt.Println("❌ การเชื่อมต่อถูกปิด:", err)
			break
		}

		// ถอดรหัส header
		topic, messageLength, err := decodeHeader(string(header))
		if err != nil {
			fmt.Println("⚠️ ไม่สามารถถอดรหัส header ได้:", err)
			continue
		}

		fmt.Printf("\n📥 รับข้อมูล Topic: %d, ความยาวข้อความ: %d bytes\n", topic, messageLength)

		// อ่าน XML message
		message := make([]byte, messageLength)
		_, err = conn.Read(message)
		if err != nil {
			fmt.Println("⚠️ ไม่สามารถอ่านข้อความ XML ได้:", err)
			break
		}

		// แปลงข้อความเป็น UTF-16LE
		xmlMessage := string(message)
		fmt.Printf("📜 ข้อความ XML: %s\n", xmlMessage)
	}
}
