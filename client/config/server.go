package config

import (
	"sync"
	"time"
)

var (
	tcpServerRunning bool
	serverMutex      sync.Mutex
	stopServerChan   chan struct{}
)

// InitServer เริ่มต้นค่าตัวแปรสำหรับ Server
func InitServer() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	tcpServerRunning = false
	stopServerChan = make(chan struct{})
}

// StartTCPServer เริ่มการทำงานของ TCP Server
func StartTCPServer() {
	serverMutex.Lock()

	// ตรวจสอบว่า server กำลังทำงานอยู่หรือไม่
	if tcpServerRunning {
		serverMutex.Unlock()
		return
	}

	tcpServerRunning = true
	stopServerChan = make(chan struct{})
	serverMutex.Unlock()

	// อัปเดตสถานะเป็น Initializing
	Config.UpdateTCPServerStatus("Initializing...")

	// รอให้การหยุดเสร็จสมบูรณ์
	time.Sleep(1 * time.Second)

	// เรียกใช้ฟังก์ชันภายนอกสำหรับเริ่ม TCP Server
	go func() {
		// เรียก StartServerFunc และรอผลลัพธ์
		StartServerFunc()

		// ตรวจสอบสถานะหลังจากเริ่ม server
		time.Sleep(2 * time.Second)
		if Config.TCPServerStatus == "Initializing..." {
			// ถ้าสถานะยังเป็น Initializing แสดงว่าเกิดปัญหา
			Config.UpdateTCPServerStatus("Failed to start server")
			tcpServerRunning = false
		} else if Config.TCPServerStatus == "TCP Server service is running" {
			// ถ้าเริ่ม server สำเร็จ
			tcpServerRunning = true
		}
	}()
}

// StopTCPServer หยุดการทำงานของ TCP Server
func StopTCPServer() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if !tcpServerRunning {
		return
	}

	// ส่งสัญญาณให้หยุดการทำงาน
	close(stopServerChan)
	tcpServerRunning = false

	// รอให้การหยุดเสร็จสมบูรณ์
	time.Sleep(1 * time.Second)

	Config.UpdateTCPServerStatus("Stopped")
}

// StartProxy เริ่มการทำงานของ Proxy Server
func StartProxy() {
	// ตรวจสอบว่า server กำลังทำงานอยู่หรือไม่
	if tcpServerRunning {
		return
	}

	// รอให้ port ถูกปล่อย (ถ้ามี)
	time.Sleep(2 * time.Second)

	// เริ่ม TCP Server
	StartTCPServer()
}

// StopProxy หยุดการทำงานของ Proxy Server
func StopProxy() {
	// หยุด TCP Server
	StopTCPServer()
}

// IsServerRunning ตรวจสอบว่า TCP Server กำลังทำงานอยู่หรือไม่
func IsServerRunning() bool {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	// ตรวจสอบทั้งสถานะการทำงานและสถานะข้อความ
	return tcpServerRunning && Config.TCPServerStatus != "Failed to start server" && Config.TCPServerStatus != "Stopped"
}

// GetStopChannel คืนค่า channel สำหรับหยุดการทำงาน
func GetStopChannel() chan struct{} {
	return stopServerChan
}

// ฟังก์ชันที่จะถูกกำหนดจากภายนอก
var StartServerFunc = func() {
	// ฟังก์ชันนี้จะถูกแทนที่ด้วยฟังก์ชันจริงจากโมดูลอื่น
}
