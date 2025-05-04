package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	ConfigVersion   string
	LicenseKey      string
	LastUpdated     time.Time
	mu              sync.RWMutex
}

// Constants
const (
	CONFIG_VERSION = "1.0.0" // รุ่นปัจจุบันของไฟล์การตั้งค่า
)

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
	ConfigVersion:   CONFIG_VERSION,
	LicenseKey:      "",
	LastUpdated:     time.Now(),
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

	// โหลดไฟล์ license
	if err := LoadLicense(); err != nil {
		return fmt.Errorf("ไม่สามารถโหลดไฟล์ license: %v", err)
	}

	return nil
}

// LoadConfig โหลดการตั้งค่าจากไฟล์ config.ini
func LoadConfig() error {
	// ค้นหาไฟล์ config.ini ในตำแหน่งต่างๆ
	possiblePaths := []string{
		"config.ini",                                // ในโฟลเดอร์ปัจจุบัน
		"config/config.ini",                         // ในโฟลเดอร์ config
		filepath.Join("..", "config.ini"),           // ในโฟลเดอร์แม่
		filepath.Join("..", "config", "config.ini"), // ในโฟลเดอร์ config ของโฟลเดอร์แม่
	}

	var configPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return fmt.Errorf("ไม่พบไฟล์ config.ini ในตำแหน่งใดๆ")
	}

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
		case "ConfigVersion":
			Config.ConfigVersion = value
		case "LicenseKey":
			Config.LicenseKey = value
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("เกิดข้อผิดพลาดในการอ่านไฟล์ config.ini: %v", err)
	}

	// อัปเดตเวลาล่าสุด
	Config.LastUpdated = time.Now()
	return nil
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

	// เขียนข้อมูลเกี่ยวกับไฟล์
	fmt.Fprintf(writer, "# DCN Configuration File\n")
	fmt.Fprintf(writer, "# Generated on: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "# Version: %s\n\n", CONFIG_VERSION)

	// เขียนการตั้งค่าลงไฟล์
	fmt.Fprintf(writer, "APIHost=%s\n", Config.APIHost)
	fmt.Fprintf(writer, "APIPort=%s\n", Config.APIPort)
	fmt.Fprintf(writer, "APIPath=%s\n", Config.APIPath)
	fmt.Fprintf(writer, "TCPServerPort=%s\n", Config.TCPServerPort)
	fmt.Fprintf(writer, "APIKey=%s\n", Config.APIKey)
	fmt.Fprintf(writer, "ConfigVersion=%s\n", CONFIG_VERSION)
	fmt.Fprintf(writer, "LicenseKey=%s\n", Config.LicenseKey)
	// อัปเดตเวลาล่าสุด
	Config.LastUpdated = time.Now()
	Config.ConfigVersion = CONFIG_VERSION

	return nil
}

// BackupConfig สร้างไฟล์สำรองก่อนการปรับปรุงการตั้งค่า
func BackupConfig() error {
	configPath := filepath.Join("config", "config.ini")
	backupPath := filepath.Join("config", fmt.Sprintf("config_backup_%s.ini", time.Now().Format("20060102150405")))

	// ทำสำเนาไฟล์ config.ini
	input, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("ไม่สามารถอ่านไฟล์ config.ini: %v", err)
	}

	// เขียนไฟล์สำรอง
	err = os.WriteFile(backupPath, input, 0644)
	if err != nil {
		return fmt.Errorf("ไม่สามารถเขียนไฟล์สำรอง: %v", err)
	}

	return nil
}

// RestoreConfig คืนค่าการตั้งค่าจากไฟล์สำรอง
func RestoreConfig(backupFile string) error {
	backupPath := filepath.Join("config", backupFile)
	configPath := filepath.Join("config", "config.ini")

	// อ่านไฟล์สำรอง
	input, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("ไม่สามารถอ่านไฟล์สำรอง: %v", err)
	}

	// เขียนทับไฟล์ config.ini
	err = os.WriteFile(configPath, input, 0644)
	if err != nil {
		return fmt.Errorf("ไม่สามารถเขียนไฟล์ config.ini: %v", err)
	}

	// โหลดการตั้งค่าใหม่
	return LoadConfig()
}

// GetAPIURL คืนค่า URL เต็มของ API
func (c *AppConfig) GetAPIURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return "http://" + c.APIHost + ":" + c.APIPort + c.APIPath
}

// GetLicenseKey คืนค่า License Key
func (c *AppConfig) GetLicenseKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LicenseKey
}

// GetAPIKey คืนค่า API Key
func (c *AppConfig) GetAPIKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
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

// UpdateConfig อัปเดตการตั้งค่าหลายค่าพร้อมกัน
func (c *AppConfig) UpdateConfig(updates map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// สำรองการตั้งค่าก่อนการเปลี่ยนแปลง
	if err := BackupConfig(); err != nil {
		return fmt.Errorf("ไม่สามารถสำรองการตั้งค่า: %v", err)
	}

	// อัปเดตค่าต่างๆ
	for key, value := range updates {
		switch key {
		case "APIHost":
			c.APIHost = value
		case "APIPort":
			c.APIPort = value
		case "APIPath":
			c.APIPath = value
		case "TCPServerPort":
			c.TCPServerPort = value
		case "APIKey":
			c.APIKey = value
		case "LicenseKey":
			c.LicenseKey = value
		}
	}

	// บันทึกการตั้งค่าใหม่
	c.LastUpdated = time.Now()
	return SaveConfig()
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

// ValidateConfig ตรวจสอบความถูกต้องของการตั้งค่า
func (c *AppConfig) ValidateConfig() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// ตรวจสอบค่าต่างๆ
	if c.APIHost == "" {
		return fmt.Errorf("APIHost ไม่สามารถเป็นค่าว่างได้")
	}

	if c.APIPort == "" {
		return fmt.Errorf("APIPort ไม่สามารถเป็นค่าว่างได้")
	}

	if c.APIPath == "" {
		return fmt.Errorf("APIPath ไม่สามารถเป็นค่าว่างได้")
	}

	if c.TCPServerPort == "" {
		return fmt.Errorf("TCPServerPort ไม่สามารถเป็นค่าว่างได้")
	}

	return nil
}

// RollbackConfig คืนค่าการตั้งค่าเป็นค่าล่าสุดในกรณีเกิดข้อผิดพลาด
func (c *AppConfig) RollbackConfig() error {
	// หาไฟล์สำรองล่าสุด
	backupDir := filepath.Join("config")
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("ไม่สามารถอ่านโฟลเดอร์ config: %v", err)
	}

	var latestBackup string
	var latestTime time.Time

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "config_backup_") && strings.HasSuffix(file.Name(), ".ini") {
			// แปลงชื่อไฟล์เป็นเวลา
			timeStr := strings.TrimPrefix(file.Name(), "config_backup_")
			timeStr = strings.TrimSuffix(timeStr, ".ini")
			fileTime, err := time.Parse("20060102150405", timeStr)
			if err != nil {
				continue
			}

			if latestBackup == "" || fileTime.After(latestTime) {
				latestBackup = file.Name()
				latestTime = fileTime
			}
		}
	}

	if latestBackup == "" {
		return fmt.Errorf("ไม่พบไฟล์สำรอง")
	}

	// คืนค่าการตั้งค่า
	return RestoreConfig(latestBackup)
}

// GetConfigInfo คืนค่าข้อมูลการตั้งค่า
func (c *AppConfig) GetConfigInfo() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]string{
		"APIHost":       c.APIHost,
		"APIPort":       c.APIPort,
		"APIPath":       c.APIPath,
		"TCPServerPort": c.TCPServerPort,
		"APIKey":        c.APIKey,
		"ConfigVersion": c.ConfigVersion,
		"LastUpdated":   c.LastUpdated.Format("2006-01-02 15:04:05"),
		"LicenseKey":    c.LicenseKey,
	}
}

// GetConfigDir รับ path ของโฟลเดอร์ config
func GetConfigDir() string {
	// ใช้โฟลเดอร์โปรเจค
	configDir := "./config"

	// สร้างโฟลเดอร์ถ้ายังไม่มี
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "."
	}

	return configDir
}

// LoadLicense คืนค่า LicenseKey จาก config.ini
func LoadLicense() error {
	licenseKey := Config.GetLicenseKey()
	if licenseKey == "" {
		return fmt.Errorf("ไม่พบ LicenseKey ใน config.ini")
	}
	// ตั้งค่า LicenseKey ใน config
	Config.LicenseKey = licenseKey
	return nil
}
