package config

import (
	"sync"
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

	Config.UpdateTCPServerStatus("Initializing...")

	// เรียกใช้ฟังก์ชันภายนอกสำหรับเริ่ม TCP Server
	StartServerFunc()
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

	Config.UpdateTCPServerStatus("Stopped")
}

// IsServerRunning ตรวจสอบว่า TCP Server กำลังทำงานอยู่หรือไม่
func IsServerRunning() bool {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	return tcpServerRunning
}

// GetStopChannel คืนค่า channel สำหรับหยุดการทำงาน
func GetStopChannel() chan struct{} {
	return stopServerChan
}

// ฟังก์ชันที่จะถูกกำหนดจากภายนอก
var StartServerFunc = func() {
	// ฟังก์ชันนี้จะถูกแทนที่ด้วยฟังก์ชันจริงจากโมดูลอื่น
}
