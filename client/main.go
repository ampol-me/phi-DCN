package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"phi-DCN/client/api"
	"phi-DCN/client/config"
	"phi-DCN/client/proxy"
)

const (
	REST_API_PORT = "3002"          // REST API Port
	API_TIMEOUT   = 5 * time.Second // Timeout for API connection check
	APP_VERSION   = "1.0.0"         // รุ่นของแอปพลิเคชัน
)

// สร้าง logger
var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

// initLogging เริ่มต้นระบบบันทึกข้อมูล
func initLogging() {
	// สร้างโครงสร้างโฟลเดอร์ logs/YYYY/MM/DD
	currentTime := time.Now()
	logsDir := "logs"
	yearDir := fmt.Sprintf("%s/%d", logsDir, currentTime.Year())
	monthDir := fmt.Sprintf("%s/%02d", yearDir, currentTime.Month())
	dayDir := fmt.Sprintf("%s/%02d", monthDir, currentTime.Day())

	// สร้างโฟลเดอร์ทั้งหมด
	dirs := []string{logsDir, yearDir, monthDir, dayDir}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, 0755)
		}
	}

	// เปิดไฟล์ log
	infoFile, err := os.OpenFile(fmt.Sprintf("%s/app_info.log", dayDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	errorFile, err := os.OpenFile(fmt.Sprintf("%s/app_error.log", dayDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	// สร้าง logger
	InfoLogger = log.New(infoFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(errorFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// ลบ logs เก่าที่เกิน 1 ปี
	go cleanupOldLogs()
}

// cleanupOldLogs ลบ logs เก่าที่เกิน 1 ปี
func cleanupOldLogs() {
	// รอ 1 นาทีหลังเริ่มแอปพลิเคชันก่อนเริ่มลบ logs
	time.Sleep(1 * time.Minute)

	// ตรวจสอบทุก 24 ชั่วโมง
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// ดึงปีปัจจุบัน
			currentYear := time.Now().Year()

			// ตรวจสอบโฟลเดอร์ logs
			logsDir := "logs"
			entries, err := os.ReadDir(logsDir)
			if err != nil {
				ErrorLogger.Printf("ไม่สามารถอ่านโฟลเดอร์ logs: %v", err)
				continue
			}

			// ตรวจสอบแต่ละปี
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				year, err := strconv.Atoi(entry.Name())
				if err != nil {
					continue
				}

				// ถ้าเป็นปีที่เก่ากว่า 1 ปี ให้ลบทั้งโฟลเดอร์
				if year < currentYear-1 {
					yearPath := filepath.Join(logsDir, entry.Name())
					if err := os.RemoveAll(yearPath); err != nil {
						ErrorLogger.Printf("ไม่สามารถลบโฟลเดอร์ปี %d: %v", year, err)
					} else {
						InfoLogger.Printf("ลบ logs ปี %d เรียบร้อยแล้ว", year)
					}
				}
			}
		}
	}
}

// loadLogs โหลดและแสดง logs ใน Text widget
func loadLogs(logText *widget.TextGrid) {
	// โหลด logs จากไฟล์ของวันนี้
	currentTime := time.Now()
	logFile := fmt.Sprintf("logs/%d/%02d/%02d/app_info.log",
		currentTime.Year(),
		currentTime.Month(),
		currentTime.Day())

	content, err := os.ReadFile(logFile)
	if err != nil {
		logText.SetText(logText.Text() + fmt.Sprintf("❌ ไม่สามารถโหลดไฟล์ log: %v\n", err))
		return
	}

	// แสดง logs ใน Text widget
	logText.SetText(logText.Text() + string(content) + "\n============\n")
}

// darkTheme กำหนด theme สีเข้ม
type darkTheme struct{}

func (t *darkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{40, 40, 40, 255} // สีพื้นหลังเทาเข้ม
	case theme.ColorNameForeground:
		return color.RGBA{200, 200, 200, 255} // สีตัวอักษรขาว
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *darkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *darkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *darkTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func main() {
	// ปิดการแสดงข้อความแจ้งเตือนของ Fyne
	os.Setenv("FYNE_DEBUG", "0")

	// เริ่มต้นระบบบันทึกข้อมูล
	initLogging()

	// สร้าง GUI application
	a := app.New()
	w := a.NewWindow("Phi DCN Client")
	w.Resize(fyne.NewSize(800, 600))

	// สร้าง Text widget สำหรับแสดง logs
	logText := widget.NewTextGrid()
	logText.SetText("=== Logs ===\n")

	// สร้าง Scroll container
	scroll := container.NewScroll(logText)
	scroll.Resize(fyne.NewSize(800, 600))

	// ตั้งค่า theme ให้เป็น dark mode
	a.Settings().SetTheme(&darkTheme{})

	// ตั้งค่า content ของ window
	w.SetContent(scroll)

	// ตั้งค่าเริ่มต้น - โหลดจากไฟล์ config.ini
	if err := config.InitConfig(); err != nil {
		logText.SetText(logText.Text() + fmt.Sprintf("❌ Failed to load configuration: %v\n", err))
		ErrorLogger.Printf("Failed to load configuration: %v", err)
		os.Exit(1)
	}
	InfoLogger.Println("Configuration loaded successfully")

	// ตรวจสอบความถูกต้องของการตั้งค่า
	if err := config.Config.ValidateConfig(); err != nil {
		logText.SetText(logText.Text() + fmt.Sprintf("❌ Invalid configuration: %v\n", err))
		ErrorLogger.Printf("Invalid configuration: %v", err)
		os.Exit(1)
	}

	// ตั้งค่า proxy
	proxy.Setup()
	InfoLogger.Println("Proxy setup completed")

	// ตรวจสอบพอร์ตสำหรับ REST API จากอาร์กิวเมนต์
	apiPort := REST_API_PORT
	if len(os.Args) > 1 {
		apiPort = os.Args[1]
		InfoLogger.Printf("Using REST API port from argument: %s", apiPort)
	}

	// จัดการกับสัญญาณหยุดการทำงาน (Ctrl+C)
	setupSignalHandler()

	// เริ่ม REST API ก่อน
	logText.SetText(logText.Text() + "Starting REST API server...\n")
	InfoLogger.Println("Starting REST API server...")
	restServerChan := make(chan error, 1)
	go func() {
		restServerChan <- api.StartRESTServer(apiPort)
	}()

	// รอให้ REST API server เริ่มทำงาน
	logText.SetText(logText.Text() + "Waiting for REST API server to start...\n")
	InfoLogger.Println("Waiting for REST API server to start...")
	time.Sleep(2 * time.Second)

	// ตรวจสอบการเชื่อมต่อ API
	logText.SetText(logText.Text() + "Checking API connection...\n")
	logText.SetText(logText.Text() + fmt.Sprintf("Trying to connect to API at: %s\n", config.Config.GetAPIURL()))
	InfoLogger.Printf("Checking API connection at: %s", config.Config.GetAPIURL())

	apiCheckChan := make(chan error, 1)
	go func() {
		// ใช้ retry mechanism
		var err error
		for i := 0; i < 3; i++ { // ลองเชื่อมต่อ 3 ครั้ง
			err = api.TestConnection()
			if err == nil {
				break
			}
			time.Sleep(500 * time.Millisecond) // รอสักครู่ก่อนลองอีกครั้ง
		}
		apiCheckChan <- err
	}()

	select {
	case err := <-apiCheckChan:
		if err != nil {
			logText.SetText(logText.Text() + fmt.Sprintf("\n❌ API connection failed: %v\n", err))
			ErrorLogger.Printf("API connection failed: %v", err)

			logText.SetText(logText.Text() + "\nPossible causes:\n")
			logText.SetText(logText.Text() + "1. API server is not running\n")
			logText.SetText(logText.Text() + "2. Incorrect API configuration\n")
			logText.SetText(logText.Text() + "3. Network connectivity issues\n")
			logText.SetText(logText.Text() + "\nPlease check your configuration:\n")
			logText.SetText(logText.Text() + fmt.Sprintf("- API Host: %s\n", config.Config.APIHost))
			logText.SetText(logText.Text() + fmt.Sprintf("- API Port: %s\n", config.Config.APIPort))
			logText.SetText(logText.Text() + fmt.Sprintf("- API Path: %s\n", config.Config.APIPath))
			logText.SetText(logText.Text() + fmt.Sprintf("- API Key: %s\n", config.Config.APIKey))
			logText.SetText(logText.Text() + "\nPlease update your configuration using:\n")
			logText.SetText(logText.Text() + fmt.Sprintf("curl -X POST http://localhost:%s/api/config -H \"Content-Type: application/json\" -d '{\"APIHost\":\"your-api-host\",\"APIPort\":\"your-api-port\",\"APIPath\":\"/api/speakers\",\"APIKey\":\"your-api-key\"}'\n", apiPort))
			os.Exit(1)
		}
		logText.SetText(logText.Text() + "✅ API connection successful\n")
		InfoLogger.Println("API connection successful")
	case <-time.After(API_TIMEOUT):
		logText.SetText(logText.Text() + fmt.Sprintf("\n❌ API connection timeout after %v\n", API_TIMEOUT))
		ErrorLogger.Printf("API connection timeout after %v", API_TIMEOUT)

		logText.SetText(logText.Text() + "\nPossible causes:\n")
		logText.SetText(logText.Text() + "1. API server is not responding\n")
		logText.SetText(logText.Text() + "2. Network latency is too high\n")
		logText.SetText(logText.Text() + "3. Firewall blocking the connection\n")
		logText.SetText(logText.Text() + "\nPlease check your configuration:\n")
		logText.SetText(logText.Text() + fmt.Sprintf("- API Host: %s\n", config.Config.APIHost))
		logText.SetText(logText.Text() + fmt.Sprintf("- API Port: %s\n", config.Config.APIPort))
		logText.SetText(logText.Text() + fmt.Sprintf("- API Path: %s\n", config.Config.APIPath))
		logText.SetText(logText.Text() + fmt.Sprintf("- API Key: %s\n", config.Config.APIKey))
		logText.SetText(logText.Text() + "\nPlease update your configuration using:\n")
		logText.SetText(logText.Text() + fmt.Sprintf("curl -X POST http://localhost:%s/api/config -H \"Content-Type: application/json\" -d '{\"APIHost\":\"your-api-host\",\"APIPort\":\"your-api-port\",\"APIPath\":\"/api/speakers\",\"APIKey\":\"your-api-key\"}'\n", apiPort))
		os.Exit(1)
	}

	// โหลดและแสดง logs
	loadLogs(logText)

	// แสดงข้อความว่าแอปพลิเคชันพร้อมใช้งาน
	logText.SetText(logText.Text() + "\n✅ Application is ready!\n")
	InfoLogger.Println("Application is ready")
	logText.SetText(logText.Text() + fmt.Sprintf("- REST API running on port: %s\n", apiPort))
	logText.SetText(logText.Text() + fmt.Sprintf("- TCP Server running on port: %s\n", config.Config.TCPServerPort))
	logText.SetText(logText.Text() + fmt.Sprintf("- Connected to API: %s\n\n", config.Config.GetAPIURL()))

	// แสดง window
	w.ShowAndRun()
}

// setupSignalHandler จัดการกับสัญญาณหยุดการทำงาน
func setupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nShutting down application...")
		InfoLogger.Println("Shutting down application...")

		// ทำความสะอาด resources
		config.Config.ClearInactiveMics()

		fmt.Println("Application closed successfully")
		InfoLogger.Println("Application closed successfully")
		os.Exit(0)
	}()
}
