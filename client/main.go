package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"phi-DCN/client/api"
	"phi-DCN/client/config"
	"phi-DCN/client/proxy"

	"fyne.io/systray"
	"github.com/lxn/walk"
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
	mainWindow  *walk.MainWindow
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
	mShow := systray.AddMenuItem("Show Window", "Show the application window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// จัดการการคลิกที่ไอคอนใน system tray
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				showMainWindow()
			case <-mQuit.ClickedCh:
				systray.Quit()
				os.Exit(0)
			}
		}
	}()

	// ซ่อนหน้าต่างแอปพลิเคชัน
	if runtime.GOOS == "windows" {
		hideMainWindow()
	}
}

// showMainWindow แสดงหน้าต่างหลัก
func showMainWindow() {
	if mainWindow != nil {
		mainWindow.Show()
		mainWindow.SetMinMaxSize(walk.Size{Width: 800, Height: 600}, walk.Size{Width: 800, Height: 600})
		mainWindow.SetSize(walk.Size{Width: 800, Height: 600})
		mainWindow.SetVisible(true)
	}
}

// hideMainWindow ซ่อนหน้าต่างหลัก
func hideMainWindow() {
	if mainWindow != nil {
		mainWindow.SetVisible(false)
	}
}

// createMainWindow สร้างหน้าต่างหลัก
func createMainWindow() error {
	var err error
	mainWindow, err = walk.NewMainWindow()
	if err != nil {
		return err
	}

	mainWindow.SetTitle("Phi DCN Bridge")
	mainWindow.SetMinMaxSize(walk.Size{Width: 800, Height: 600}, walk.Size{Width: 800, Height: 600})
	mainWindow.SetSize(walk.Size{Width: 800, Height: 600})

	// จัดการการปิดหน้าต่าง
	mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		hideMainWindow()
	})

	return nil
}

func onExit() {
	// ทำความสะอาด resources
	config.Config.ClearInactiveMics()
}

func main() {
	// เริ่มต้นระบบบันทึกข้อมูล
	initLogging()

	// ตั้งค่าเริ่มต้น - โหลดจากไฟล์ config.ini
	if err := config.InitConfig(); err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		ErrorLogger.Printf("Failed to load configuration: %v", err)
		os.Exit(1)
	}
	InfoLogger.Println("Configuration loaded successfully")

	// ตรวจสอบความถูกต้องของการตั้งค่า
	if err := config.Config.ValidateConfig(); err != nil {
		fmt.Printf("❌ Invalid configuration: %v\n", err)
		ErrorLogger.Printf("Invalid configuration: %v", err)
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
			fmt.Printf("\n❌ API connection failed: %v\n", err)
			ErrorLogger.Printf("API connection failed: %v", err)

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
			os.Exit(1)
		}
		fmt.Println("✅ API connection successful")
		InfoLogger.Println("API connection successful")
	case <-time.After(API_TIMEOUT):
		fmt.Printf("\n❌ API connection timeout after %v\n", API_TIMEOUT)
		ErrorLogger.Printf("API connection timeout after %v", API_TIMEOUT)

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
		os.Exit(1)
	}

	// แสดงข้อความว่าแอปพลิเคชันพร้อมใช้งาน
	fmt.Println("\n✅ Application is ready!")
	InfoLogger.Println("Application is ready")
	fmt.Printf("- REST API running on port: %s\n", apiPort)
	fmt.Printf("- TCP Server port: %s (not started yet)\n", config.Config.TCPServerPort)
	fmt.Printf("- Connected to API: %s\n\n", config.Config.GetAPIURL())

	// ถ้าเป็น Windows ให้เริ่ม systray และแสดงหน้าต่างหลัก
	if runtime.GOOS == "windows" {
		if err := createMainWindow(); err != nil {
			ErrorLogger.Printf("Failed to create main window: %v", err)
			os.Exit(1)
		}
		go systray.Run(onReady, onExit)
		mainWindow.Show()
		mainWindow.Run()
	} else {
		// รอให้แอปพลิเคชันทำงานต่อไป
		select {}
	}
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
