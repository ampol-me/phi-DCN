package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

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
	// สร้างโฟลเดอร์ logs ถ้ายังไม่มี
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", 0755)
	}

	// เปิดไฟล์ log
	currentTime := time.Now().Format("2006-01-02")
	infoFile, err := os.OpenFile(fmt.Sprintf("logs/app_info_%s.log", currentTime), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	errorFile, err := os.OpenFile(fmt.Sprintf("logs/app_error_%s.log", currentTime), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	// สร้าง logger
	InfoLogger = log.New(infoFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(errorFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
	// เริ่มต้นระบบบันทึกข้อมูล
	initLogging()

	// สร้าง GUI window
	var mw *walk.MainWindow
	var textEdit *walk.TextEdit

	MainWindow{
		AssignTo: &mw,
		Title:    "Phi DCN Client",
		MinSize:  Size{400, 300},
		Layout:   VBox{},
		Children: []Widget{
			TextEdit{
				AssignTo: &textEdit,
				ReadOnly: true,
				VScroll:  true,
			},
		},
	}.Create()

	// ตั้งค่าเริ่มต้น - โหลดจากไฟล์ config.ini
	if err := config.InitConfig(); err != nil {
		textEdit.AppendText(fmt.Sprintf("❌ ไม่สามารถโหลดการตั้งค่าได้: %v\n", err))
		ErrorLogger.Printf("ไม่สามารถโหลดการตั้งค่าได้: %v", err)
		os.Exit(1)
	}
	InfoLogger.Println("โหลดการตั้งค่าเรียบร้อยแล้ว")

	// ตรวจสอบความถูกต้องของการตั้งค่า
	if err := config.Config.ValidateConfig(); err != nil {
		textEdit.AppendText(fmt.Sprintf("❌ การตั้งค่าไม่ถูกต้อง: %v\n", err))
		ErrorLogger.Printf("การตั้งค่าไม่ถูกต้อง: %v", err)
		os.Exit(1)
	}

	// ตั้งค่า proxy
	proxy.Setup()
	InfoLogger.Println("ตั้งค่า proxy เรียบร้อยแล้ว")

	// ตรวจสอบพอร์ตสำหรับ REST API จากอาร์กิวเมนต์
	apiPort := REST_API_PORT
	if len(os.Args) > 1 {
		apiPort = os.Args[1]
		InfoLogger.Printf("ใช้พอร์ต REST API จากอาร์กิวเมนต์: %s", apiPort)
	}

	// จัดการกับสัญญาณหยุดการทำงาน (Ctrl+C)
	setupSignalHandler()

	// เริ่ม REST API ก่อน
	textEdit.AppendText("Starting REST API server...\n")
	InfoLogger.Println("กำลังเริ่ม REST API server...")
	restServerChan := make(chan error, 1)
	go func() {
		restServerChan <- api.StartRESTServer(apiPort)
	}()

	// รอให้ REST API server เริ่มทำงาน
	textEdit.AppendText("Waiting for REST API server to start...\n")
	InfoLogger.Println("กำลังรอให้ REST API server เริ่มทำงาน...")
	time.Sleep(2 * time.Second)

	// ตรวจสอบการเชื่อมต่อ API
	textEdit.AppendText("Checking API connection...\n")
	textEdit.AppendText(fmt.Sprintf("Trying to connect to API at: %s\n", config.Config.GetAPIURL()))
	InfoLogger.Printf("กำลังตรวจสอบการเชื่อมต่อ API ที่: %s", config.Config.GetAPIURL())

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
			textEdit.AppendText(fmt.Sprintf("\n❌ API connection failed: %v\n", err))
			ErrorLogger.Printf("การเชื่อมต่อ API ล้มเหลว: %v", err)

			textEdit.AppendText("\nPossible causes:\n")
			textEdit.AppendText("1. API server is not running\n")
			textEdit.AppendText("2. Incorrect API configuration\n")
			textEdit.AppendText("3. Network connectivity issues\n")
			textEdit.AppendText("\nPlease check your configuration:\n")
			textEdit.AppendText(fmt.Sprintf("- API Host: %s\n", config.Config.APIHost))
			textEdit.AppendText(fmt.Sprintf("- API Port: %s\n", config.Config.APIPort))
			textEdit.AppendText(fmt.Sprintf("- API Path: %s\n", config.Config.APIPath))
			textEdit.AppendText(fmt.Sprintf("- API Key: %s\n", config.Config.APIKey))
			textEdit.AppendText("\nPlease update your configuration using:\n")
			textEdit.AppendText(fmt.Sprintf("curl -X POST http://localhost:%s/api/config -H \"Content-Type: application/json\" -d '{\"APIHost\":\"your-api-host\",\"APIPort\":\"your-api-port\",\"APIPath\":\"/api/speakers\",\"APIKey\":\"your-api-key\"}'\n", apiPort))
			os.Exit(1)
		}
		textEdit.AppendText("✅ API connection successful\n")
		InfoLogger.Println("การเชื่อมต่อ API สำเร็จ")
	case <-time.After(API_TIMEOUT):
		textEdit.AppendText(fmt.Sprintf("\n❌ API connection timeout after %v\n", API_TIMEOUT))
		ErrorLogger.Printf("การเชื่อมต่อ API หมดเวลาหลังจาก %v", API_TIMEOUT)

		textEdit.AppendText("\nPossible causes:\n")
		textEdit.AppendText("1. API server is not responding\n")
		textEdit.AppendText("2. Network latency is too high\n")
		textEdit.AppendText("3. Firewall blocking the connection\n")
		textEdit.AppendText("\nPlease check your configuration:\n")
		textEdit.AppendText(fmt.Sprintf("- API Host: %s\n", config.Config.APIHost))
		textEdit.AppendText(fmt.Sprintf("- API Port: %s\n", config.Config.APIPort))
		textEdit.AppendText(fmt.Sprintf("- API Path: %s\n", config.Config.APIPath))
		textEdit.AppendText(fmt.Sprintf("- API Key: %s\n", config.Config.APIKey))
		textEdit.AppendText("\nPlease update your configuration using:\n")
		textEdit.AppendText(fmt.Sprintf("curl -X POST http://localhost:%s/api/config -H \"Content-Type: application/json\" -d '{\"APIHost\":\"your-api-host\",\"APIPort\":\"your-api-port\",\"APIPath\":\"/api/speakers\",\"APIKey\":\"your-api-key\"}'\n", apiPort))
		os.Exit(1)
	}

	// แสดงข้อความว่าแอปพลิเคชันพร้อมใช้งาน
	textEdit.AppendText("\n✅ Application is ready!\n")
	InfoLogger.Println("แอปพลิเคชันพร้อมใช้งาน")
	textEdit.AppendText(fmt.Sprintf("- REST API running on port: %s\n", apiPort))
	textEdit.AppendText(fmt.Sprintf("- TCP Server running on port: %s\n", config.Config.TCPServerPort))
	textEdit.AppendText(fmt.Sprintf("- Connected to API: %s\n\n", config.Config.GetAPIURL()))

	// รัน GUI
	mw.Run()
}

// setupSignalHandler จัดการกับสัญญาณหยุดการทำงาน
func setupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nShutting down application...")
		InfoLogger.Println("กำลังปิดแอปพลิเคชัน...")

		// ทำความสะอาด resources
		config.Config.ClearInactiveMics()

		fmt.Println("Application closed successfully")
		InfoLogger.Println("ปิดแอปพลิเคชันเรียบร้อยแล้ว")
		os.Exit(0)
	}()
}
