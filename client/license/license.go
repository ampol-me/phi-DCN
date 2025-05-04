// client/license/license.go
package license

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"phi-DCN/client/config"
)

// LicenseType ประเภทของ license
type LicenseType string

const (
	// Trial license ประเภททดลองใช้
	Trial LicenseType = "trial"
	// Standard license ประเภทมาตรฐาน
	Standard LicenseType = "standard"
	// Enterprise license ประเภทองค์กร
	Enterprise LicenseType = "enterprise"
)

// ProxyInfo ข้อมูล proxy
type ProxyInfo struct {
	Address string
	Port    int
	Status  string
}

// ตัวแปร global
var (
	currentLicense *LicenseInfo
	ErrorLogger    = log.New(os.Stderr, "LICENSE ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	checkInterval  = 1 * time.Hour
	proxy          *ProxyInfo
)

// LicenseInfo ข้อมูล license
type LicenseInfo struct {
	ID         string      `json:"id"`
	Key        string      `json:"key"`
	Type       LicenseType `json:"type"`
	ExpiryDate time.Time   `json:"expiryDate"`
	MaxClients int         `json:"maxClients"`
	Features   string      `json:"features"`
	Status     string      `json:"status"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
	LastSync   time.Time   `json:"lastSync"`
	BackupPath string      `json:"backupPath"`
}

// ตรวจสอบเวลาของเครื่อง client
func checkSystemTime() error {
	// ตรวจสอบเวลาของเครื่อง client กับ NTP server
	ntpTime, err := getNTPTime()
	if err != nil {
		return fmt.Errorf("failed to get NTP time: %v", err)
	}

	localTime := time.Now()
	diff := localTime.Sub(ntpTime)

	// ถ้าเวลาต่างกันเกิน 5 นาที
	if diff > 5*time.Minute || diff < -5*time.Minute {
		return fmt.Errorf("system time is incorrect (diff: %v)", diff)
	}

	return nil
}

// ดึงเวลาจาก NTP server
func getNTPTime() (time.Time, error) {
	// ใช้ NTP server ของ Google
	resp, err := http.Get("https://time.google.com")
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	return time.Parse(time.RFC1123, resp.Header.Get("Date"))
}

// สร้าง backup license
func // `createLicenseBackup` is a function that creates a backup of the current license information.
// It first checks if there is a current license available. If there is, it creates a backup
// directory if it doesn't already exist. Then, it creates a backup file in the backup directory
// with the current timestamp as part of the file name. The license information is then marshaled
// into JSON format and written to the backup file. Finally, the backup file path is stored in the
// current license information.
createLicenseBackup() error {
	if currentLicense == nil {
		return errors.New("no license to backup")
	}

	// สร้างโฟลเดอร์ backup ถ้ายังไม่มี
	backupDir := filepath.Join(config.GetConfigDir(), "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// สร้างไฟล์ backup
	backupFile := filepath.Join(backupDir, fmt.Sprintf("license_%s.json", time.Now().Format("20060102_150405")))
	data, err := json.MarshalIndent(currentLicense, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal license: %v", err)
	}

	if err := ioutil.WriteFile(backupFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %v", err)
	}

	currentLicense.BackupPath = backupFile
	return nil
}

// แจ้งเตือนก่อน license หมดอายุ
func checkLicenseExpiry() {
	if currentLicense == nil {
		return
	}

	// ตรวจสอบทุกวัน
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			daysLeft := int(currentLicense.ExpiryDate.Sub(time.Now()).Hours() / 24)

			// แจ้งเตือนเมื่อเหลือ 7 วัน
			if daysLeft == 7 {
				ErrorLogger.Printf("License will expire in 7 days")
				fmt.Printf("⚠️ License will expire in 7 days\n")
			}

			// แจ้งเตือนเมื่อเหลือ 3 วัน
			if daysLeft == 3 {
				ErrorLogger.Printf("License will expire in 3 days")
				fmt.Printf("⚠️ License will expire in 3 days\n")
			}

			// แจ้งเตือนเมื่อเหลือ 1 วัน
			if daysLeft == 1 {
				ErrorLogger.Printf("License will expire in 1 day")
				fmt.Printf("⚠️ License will expire in 1 day\n")
			}
		}
	}()
}

// sync ข้อมูลกับ server
func syncWithServer() error {
	if currentLicense == nil {
		return errors.New("no license to sync")
	}

	// ตรวจสอบการเชื่อมต่ออินเตอร์เน็ต
	if !isInternetAvailable() {
		return errors.New("no internet connection")
	}

	// ส่งข้อมูลไปยัง server
	client := &http.Client{Timeout: 10 * time.Second}
	data, err := json.Marshal(currentLicense)
	if err != nil {
		return fmt.Errorf("failed to marshal license: %v", err)
	}

	resp, err := client.Post("http://lic.phim.in.th/api/license/validate", "application/json", strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to sync with server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status code: %d", resp.StatusCode)
	}

	// อัพเดทเวลาที่ sync ล่าสุด
	currentLicense.LastSync = time.Now()
	return nil
}

// ตรวจสอบการเชื่อมต่ออินเตอร์เน็ต
func isInternetAvailable() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	_, err := client.Get("https://www.google.com")
	return err == nil
}

// StartLicenseCheck เริ่มการตรวจสอบ license เป็นระยะ
func StartLicenseCheck() {
	// ตรวจสอบเวลาของเครื่อง
	if err := checkSystemTime(); err != nil {
		ErrorLogger.Printf("System time check failed: %v", err)
		fmt.Printf("⚠️ System time check failed: %v\n", err)
	}

	// สร้าง backup license
	if err := createLicenseBackup(); err != nil {
		ErrorLogger.Printf("Failed to create license backup: %v", err)
		fmt.Printf("⚠️ Failed to create license backup: %v\n", err)
	}

	// เริ่มการตรวจสอบ license เป็นระยะ
	ticker := time.NewTicker(checkInterval)
	go func() {
		for range ticker.C {
			if err := ValidateLicense(); err != nil {
				ErrorLogger.Printf("License validation failed: %v", err)
				fmt.Printf("❌ License validation failed: %v\n", err)

				if proxy != nil {
					StopProxy(proxy)
				}
			}

			// พยายาม sync กับ server
			if err := syncWithServer(); err != nil {
				ErrorLogger.Printf("Failed to sync with server: %v", err)
				fmt.Printf("⚠️ Failed to sync with server: %v\n", err)
			}
		}
	}()

	// เริ่มการตรวจสอบวันหมดอายุ
	checkLicenseExpiry()
}

// ValidateLicense ตรวจสอบความถูกต้องของ license
func ValidateLicense() error {
	if currentLicense == nil {
		return errors.New("no license found")
	}

	// ตรวจสอบวันหมดอายุ
	if time.Now().After(currentLicense.ExpiryDate) {
		return errors.New("license has expired")
	}

	// ตรวจสอบสถานะ
	if currentLicense.Status != "active" {
		return fmt.Errorf("license status is %s", currentLicense.Status)
	}

	return nil
}

// StopProxy หยุดการทำงานของ proxy
func StopProxy(p *ProxyInfo) error {
	if p == nil {
		return errors.New("no proxy to stop")
	}

	p.Status = "stopped"
	return nil
}

// LoadLicense โหลดและตรวจสอบ license จาก server หรือไฟล์ backup
func LoadLicense() error {
	// โหลด license key จาก config
	licenseKey := config.Config.GetLicenseKey()
	if licenseKey == "" {
		return errors.New("no license key found")
	}

	// พยายามโหลดจาก server ก่อน
	err := loadLicenseFromServer(licenseKey)
	if err == nil {
		return nil // สำเร็จ
	}

	// ถ้าเชื่อมต่อ server ไม่ได้ ให้โหลดจากไฟล์ backup
	fmt.Printf("⚠️ Cannot connect to license server: %v\n", err)
	fmt.Printf("⚠️ Trying to load from backup file...\n")

	return loadLicenseFromBackup(licenseKey)
}

// loadLicenseFromServer โหลดและตรวจสอบ license จาก server
func loadLicenseFromServer(licenseKey string) error {
	// สร้าง request body
	reqBody := map[string]string{
		"key": licenseKey,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// ส่ง request ไปยัง server
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(
		"http://lic.phim.in.th/api/license/validate",
		"application/json",
		strings.NewReader(string(data)),
	)
	if err != nil {
		return fmt.Errorf("failed to validate license: %v", err)
	}
	defer resp.Body.Close()

	// อ่าน response
	var result struct {
		Valid   bool        `json:"valid"`
		License LicenseInfo `json:"license"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if !result.Valid {
		return errors.New("invalid license")
	}

	// อัพเดท license
	currentLicense = &result.License
	currentLicense.LastSync = time.Now()

	// สร้าง backup
	if err := createLicenseBackup(); err != nil {
		fmt.Printf("⚠️ Failed to create backup: %v\n", err)
	}

	logLicenseInfo()
	return nil
}

// loadLicenseFromBackup โหลด license จากไฟล์ backup
func loadLicenseFromBackup(licenseKey string) error {
	// หาไฟล์ backup ล่าสุด
	backupDir := filepath.Join(config.GetConfigDir(), "backup")
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %v", err)
	}

	var latestBackup string
	var latestTime time.Time

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "license_") && strings.HasSuffix(file.Name(), ".json") {
			// แปลงชื่อไฟล์เป็นเวลา
			timeStr := strings.TrimPrefix(file.Name(), "license_")
			timeStr = strings.TrimSuffix(timeStr, ".json")
			fileTime, err := time.Parse("20060102_150405", timeStr)
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
		return errors.New("no backup file found")
	}

	// อ่านไฟล์ backup
	backupFile := filepath.Join(backupDir, latestBackup)
	data, err := ioutil.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %v", err)
	}

	// แปลงข้อมูลเป็น license
	var license LicenseInfo
	if err := json.Unmarshal(data, &license); err != nil {
		return fmt.Errorf("failed to unmarshal backup: %v", err)
	}

	// ตรวจสอบว่าเป็น license key เดียวกัน
	if license.Key != licenseKey {
		return errors.New("license key mismatch")
	}

	// ตรวจสอบวันหมดอายุ
	if time.Now().After(license.ExpiryDate) {
		return errors.New("license has expired")
	}

	// ตรวจสอบสถานะ
	if license.Status != "active" {
		return fmt.Errorf("license status is %s", license.Status)
	}

	// อัพเดท license
	currentLicense = &license
	fmt.Printf("✅ Loaded license from backup (last sync: %s)\n",
		license.LastSync.Format("2006-01-02 15:04:05"))

	logLicenseInfo()
	return nil
}

// logLicenseInfo แสดงข้อมูล license
func logLicenseInfo() {
	daysLeft := int(time.Until(currentLicense.ExpiryDate).Hours() / 24)
	licenseInfo := fmt.Sprintf(
		"License Info:\n"+
			"  ID: %s\n"+
			"  Type: %s\n"+
			"  Expiry Date: %s (%d days left)\n"+
			"  Max Clients: %d\n"+
			"  Features: %s\n"+
			"  Status: %s",
		currentLicense.ID,
		currentLicense.Type,
		currentLicense.ExpiryDate.Format("2006-01-02"),
		daysLeft,
		currentLicense.MaxClients,
		currentLicense.Features,
		currentLicense.Status,
	)

	fmt.Printf("✅ License validated successfully\n%s\n", licenseInfo)
}
