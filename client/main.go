package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"phi-DCN/client/api"
	"phi-DCN/client/config"
	"phi-DCN/client/proxy"

	"fyne.io/systray"
)

const (
	REST_API_PORT = "3002"            // REST API Port
	API_TIMEOUT   = 5 * time.Second   // Timeout for API connection check
	APP_VERSION   = "1.0.0"           // รุ่นของแอปพลิเคชัน
	ICON_PATH     = "assets/icon.ico" // Path to icon file
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
	yearDir := filepath.Join(logsDir, fmt.Sprintf("%d", currentTime.Year()))
	monthDir := filepath.Join(yearDir, fmt.Sprintf("%02d", currentTime.Month()))
	dayDir := filepath.Join(monthDir, fmt.Sprintf("%02d", currentTime.Day()))

	// สร้างโฟลเดอร์ทั้งหมด
	dirs := []string{logsDir, yearDir, monthDir, dayDir}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Fatalf("ไม่สามารถสร้างโฟลเดอร์ log ได้: %v", err)
			}
		}
	}

	// เปิดไฟล์ log
	infoFile, err := os.OpenFile(filepath.Join(dayDir, "app_info.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	errorFile, err := os.OpenFile(filepath.Join(dayDir, "app_error.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	// สร้าง logger
	InfoLogger = log.New(infoFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(errorFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func onReady() {
	// โหลดไอคอนจากไฟล์
	iconData, err := os.ReadFile(ICON_PATH)
	if err != nil {
		ErrorLogger.Printf("Failed to load icon: %v", err)
		// ใช้ไอคอนเริ่มต้นถ้าโหลดไม่ได้
		systray.SetIcon([]byte{})
	} else {
		systray.SetIcon(iconData)
	}

	// ตั้งค่าไอคอนและเมนู
	systray.SetTitle("Phi DCN Bridge")
	systray.SetTooltip("Phi DCN Bridge is running")

	// เพิ่มเมนู
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// จัดการการคลิกที่ไอคอนใน system tray
	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				ErrorLogger.Println("User requested application quit")
				systray.Quit()
				os.Exit(0)
			}
		}
	}()
}

func onExit() {
	// ทำความสะอาด resources
	InfoLogger.Println("Cleaning up resources...")
	config.Config.ClearInactiveMics()
	InfoLogger.Println("Cleanup completed")
}

func main() {
	// เริ่มต้นระบบบันทึกข้อมูล
	initLogging()
	InfoLogger.Println("Starting application...")

	// ตั้งค่าเริ่มต้น - โหลดจากไฟล์ config.ini
	if err := config.InitConfig(); err != nil {
		errorMsg := fmt.Sprintf("Failed to load configuration: %v", err)
		fmt.Printf("❌ %s\n", errorMsg)
		ErrorLogger.Printf(errorMsg)
		fmt.Println("\nPlease check your configuration file (config.ini) and restart the application.")
		fmt.Println("Press Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}
	InfoLogger.Println("Configuration loaded successfully")

	// ตรวจสอบความถูกต้องของการตั้งค่า
	if err := config.Config.ValidateConfig(); err != nil {
		errorMsg := fmt.Sprintf("Invalid configuration: %v", err)
		fmt.Printf("❌ %s\n", errorMsg)
		ErrorLogger.Printf(errorMsg)
		fmt.Println("\nPlease check your configuration file (config.ini) and restart the application.")
		fmt.Println("Press Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}

	// ตั้งค่า proxy โดยไม่เริ่ม TCP Server ทันที
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
	fmt.Println("Starting REST API server...")
	InfoLogger.Println("Starting REST API server...")
	restServerChan := make(chan error, 1)
	go func() {
		restServerChan <- api.StartRESTServer(apiPort)
	}()

	// รอให้ REST API server เริ่มทำงาน
	fmt.Println("Waiting for REST API server to start...")
	InfoLogger.Println("Waiting for REST API server to start...")
	time.Sleep(2 * time.Second)

	// ตรวจสอบการเชื่อมต่อ API
	fmt.Println("Checking API connection...")
	fmt.Printf("Trying to connect to API at: %s\n", config.Config.GetAPIURL())
	InfoLogger.Printf("Checking API connection at: %s", config.Config.GetAPIURL())

	apiCheckChan := make(chan error, 1)
	go func() {
		var err error
		for i := 0; i < 3; i++ {
			err = api.TestConnection()
			if err == nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		apiCheckChan <- err
	}()

	select {
	case err := <-apiCheckChan:
		if err != nil {
			errorMsg := fmt.Sprintf("API connection failed: %v", err)
			fmt.Printf("\n❌ %s\n", errorMsg)
			ErrorLogger.Printf(errorMsg)
			fmt.Println("\nPossible causes:")
			fmt.Println("1. API server is not running")
			fmt.Println("2. Incorrect API configuration")
			fmt.Println("3. Network connectivity issues")
			fmt.Println("\nPlease check your configuration:")
			fmt.Printf("- API Host: %s\n", config.Config.APIHost)
			fmt.Printf("- API Port: %s\n", config.Config.APIPort)
			fmt.Printf("- API Path: %s\n", config.Config.APIPath)
			fmt.Printf("- API Key: %s\n", config.Config.APIKey)
			fmt.Println("\nPlease update your configuration using:")
			fmt.Printf("curl -X POST http://localhost:%s/api/config -H \"Content-Type: application/json\" -d '{\"APIHost\":\"your-api-host\",\"APIPort\":\"your-api-port\",\"APIPath\":\"/api/speakers\",\"APIKey\":\"your-api-key\"}'\n", apiPort)
			fmt.Println("\nPress Enter to exit...")
			fmt.Scanln()
			os.Exit(1)
		}
		fmt.Println("✅ API connection successful")
		InfoLogger.Println("API connection successful")
	case <-time.After(API_TIMEOUT):
		errorMsg := fmt.Sprintf("API connection timeout after %v", API_TIMEOUT)
		fmt.Printf("\n❌ %s\n", errorMsg)
		ErrorLogger.Printf(errorMsg)
		fmt.Println("\nPossible causes:")
		fmt.Println("1. API server is not responding")
		fmt.Println("2. Network latency is too high")
		fmt.Println("3. Firewall blocking the connection")
		fmt.Println("\nPlease check your configuration:")
		fmt.Printf("- API Host: %s\n", config.Config.APIHost)
		fmt.Printf("- API Port: %s\n", config.Config.APIPort)
		fmt.Printf("- API Path: %s\n", config.Config.APIPath)
		fmt.Printf("- API Key: %s\n", config.Config.APIKey)
		fmt.Println("\nPlease update your configuration using:")
		fmt.Printf("curl -X POST http://localhost:%s/api/config -H \"Content-Type: application/json\" -d '{\"APIHost\":\"your-api-host\",\"APIPort\":\"your-api-port\",\"APIPath\":\"/api/speakers\",\"APIKey\":\"your-api-key\"}'\n", apiPort)
		fmt.Println("\nPress Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}

	// แสดงข้อความว่าแอปพลิเคชันพร้อมใช้งาน
	fmt.Println("\n✅ Application is ready!")
	InfoLogger.Println("Application is ready")
	fmt.Printf("- REST API running on port: %s\n", apiPort)
	fmt.Printf("- TCP Server port: %s (not started yet)\n", config.Config.TCPServerPort)
	fmt.Printf("- Connected to API: %s\n\n", config.Config.GetAPIURL())

	// เริ่ม systray
	systray.Run(onReady, onExit)
}

// setupSignalHandler จัดการกับสัญญาณหยุดการทำงาน
func setupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		ErrorLogger.Println("Received termination signal")
		fmt.Println("\nReceived termination signal. Cleaning up...")
		onExit()
		os.Exit(0)
	}()
}
