package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	APIPort:         "3000",
	APIPath:         "/api/speakers",
	TCPServerPort:   "20000",
	APIKey:          "",
	TCPServerStatus: "Waiting to start",
	APIStatus:       "Not connected",
	ActiveMics:      make(map[string]bool),
	Clients:         make(map[string]string),
}

// InitConfig เริ่มต้นการตั้งค่าโดยโหลดจากไฟล์ config.ini
func InitConfig() error {
	// โหลดการตั้งค่าจากไฟล์ config.ini
	if err := LoadConfig(); err != nil {
		// ถ้าโหลดไม่ได้ ให้ใช้ค่าเริ่มต้นและบันทึกไฟล์ config.ini
		if err := SaveConfig(); err != nil {
			return fmt.Errorf("ไม่สามารถบันทึกไฟล์ config.ini: %v", err)
		}
	}
	return nil
}

// LoadConfig โหลดการตั้งค่าจากไฟล์ config.ini
func LoadConfig() error {
	configPath := filepath.Join("config", "config.ini")
	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("ไม่สามารถเปิดไฟล์ config.ini: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "APIHost":
			Config.APIHost = value
		case "APIPort":
			Config.APIPort = value
		case "APIPath":
			Config.APIPath = value
		case "TCPServerPort":
			Config.TCPServerPort = value
		case "APIKey":
			Config.APIKey = value
		}
	}

	return scanner.Err()
}

// SaveConfig บันทึกการตั้งค่าลงไฟล์ config.ini
func SaveConfig() error {
	configPath := filepath.Join("config", "config.ini")

	// สร้างโฟลเดอร์ config ถ้ายังไม่มี
	if err := os.MkdirAll("config", 0755); err != nil {
		return fmt.Errorf("ไม่สามารถสร้างโฟลเดอร์ config: %v", err)
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("ไม่สามารถสร้างไฟล์ config.ini: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// เขียนการตั้งค่าลงไฟล์
	fmt.Fprintf(writer, "APIHost=%s\n", Config.APIHost)
	fmt.Fprintf(writer, "APIPort=%s\n", Config.APIPort)
	fmt.Fprintf(writer, "APIPath=%s\n", Config.APIPath)
	fmt.Fprintf(writer, "TCPServerPort=%s\n", Config.TCPServerPort)
	fmt.Fprintf(writer, "APIKey=%s\n", Config.APIKey)

	return nil
}

// GetAPIURL คืนค่า URL เต็มของ API
func (c *AppConfig) GetAPIURL() string {
	return "http://" + c.APIHost + ":" + c.APIPort + c.APIPath
	//return "http://" + c.APIHost + ":" + c.APIPort + c.APIPath + "?isPolling=true"
}

// GetAPIKey คืนค่า API Key
func (c *AppConfig) GetAPIKey() string {
	return c.APIKey
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

// ClearInactiveMics ล้างข้อมูลไมค์ที่ไม่ได้ใช้งานแล้ว
func (c *AppConfig) ClearInactiveMics() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// ล้างข้อมูลไมค์ทั้งหมด
	c.ActiveMics = make(map[string]bool)
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
