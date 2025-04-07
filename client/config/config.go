package config

import (
	"sync"
)

// AppConfig เก็บการตั้งค่าของแอปพลิเคชัน
type AppConfig struct {
	APIHost         string
	APIPort         string
	APIPath         string
	TCPServerPort   string
	APIKey          string
	TCPServerStatus string
	APIStatus       string
	ActiveMics      map[string]bool   // SeatName -> status
	Clients         map[string]string // IP -> info
	mu              sync.RWMutex
}

// Config คือ global configuration instance
var Config = &AppConfig{
	APIHost:         "localhost",
	APIPort:         "3001",
	APIPath:         "/api/speakers",
	TCPServerPort:   "20000",
	APIKey:          "",
	TCPServerStatus: "Waiting to start",
	APIStatus:       "Not connected",
	ActiveMics:      make(map[string]bool),
	Clients:         make(map[string]string),
}

// GetAPIURL คืนค่า URL เต็มของ API
func (c *AppConfig) GetAPIURL() string {
	return "http://" + c.APIHost + ":" + c.APIPort + c.APIPath
}

// UpdateAPIStatus อัปเดตสถานะการเชื่อมต่อ API
func (c *AppConfig) UpdateAPIStatus(status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.APIStatus = status
}

// UpdateTCPServerStatus อัปเดตสถานะของ TCP Server
func (c *AppConfig) UpdateTCPServerStatus(status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TCPServerStatus = status
}

// AddActiveClient เพิ่ม client ที่เชื่อมต่อ
func (c *AppConfig) AddActiveClient(ip, info string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Clients[ip] = info
}

// RemoveActiveClient ลบ client ที่ยกเลิกการเชื่อมต่อ
func (c *AppConfig) RemoveActiveClient(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Clients, ip)
}

// GetActiveClients ดึงรายการ client ที่เชื่อมต่ออยู่
func (c *AppConfig) GetActiveClients() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// สร้าง map ใหม่เพื่อป้องกันการแก้ไขข้อมูลต้นฉบับ
	clients := make(map[string]string)
	for ip, info := range c.Clients {
		clients[ip] = info
	}

	return clients
}

// UpdateActiveMic อัปเดตสถานะไมค์ที่กำลังใช้งาน
func (c *AppConfig) UpdateActiveMic(seatName string, active bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if active {
		c.ActiveMics[seatName] = true
	} else {
		delete(c.ActiveMics, seatName)
	}
}

// GetActiveMics ดึงรายการไมค์ที่กำลังเปิดอยู่
func (c *AppConfig) GetActiveMics() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var activeMics []string
	for name, active := range c.ActiveMics {
		if active {
			activeMics = append(activeMics, name)
		}
	}

	return activeMics
}
