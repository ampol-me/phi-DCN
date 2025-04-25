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

	fmt.Println("Initializing application...")
	InfoLogger.Println("เริ่มต้นแอปพลิเคชัน...")

	// แสดงข้อมูลของแอปพลิเคชัน
	fmt.Printf("DCN Application version %s\n", APP_VERSION)
	fmt.Printf("Running on %s %s\n", runtime.GOOS, runtime.GOARCH)
	InfoLogger.Printf("DCN Application version %s, Running on %s %s", APP_VERSION, runtime.GOOS, runtime.GOARCH)

	// ตั้งค่าเริ่มต้น - โหลดจากไฟล์ config.ini
	if err := config.InitConfig(); err != nil {
		fmt.Printf("❌ ไม่สามารถโหลดการตั้งค่าได้: %v\n", err)
		ErrorLogger.Printf("ไม่สามารถโหลดการตั้งค่าได้: %v", err)
		os.Exit(1)
	}
	InfoLogger.Println("โหลดการตั้งค่าเรียบร้อยแล้ว")

	// ตรวจสอบความถูกต้องของการตั้งค่า
	if err := config.Config.ValidateConfig(); err != nil {
		fmt.Printf("❌ การตั้งค่าไม่ถูกต้อง: %v\n", err)
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
	fmt.Println("Starting REST API server...")
	InfoLogger.Println("กำลังเริ่ม REST API server...")
	restServerChan := make(chan error, 1)
	go func() {
		restServerChan <- api.StartRESTServer(apiPort)
	}()

	// รอให้ REST API server เริ่มทำงาน
	fmt.Println("Waiting for REST API server to start...")
	InfoLogger.Println("กำลังรอให้ REST API server เริ่มทำงาน...")
	time.Sleep(2 * time.Second)

	// ตรวจสอบการเชื่อมต่อ API
	fmt.Println("Checking API connection...")
	fmt.Printf("Trying to connect to API at: %s\n", config.Config.GetAPIURL())
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
			fmt.Printf("\n❌ API connection failed: %v\n", err)
			ErrorLogger.Printf("การเชื่อมต่อ API ล้มเหลว: %v", err)

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

			// รอให้ผู้ใช้กดปุ่มเพื่อปิดโปรแกรม
			fmt.Println("\nPress Enter to exit...")
			fmt.Scanln()
			os.Exit(1)
		}
		fmt.Println("✅ API connection successful")
		InfoLogger.Println("การเชื่อมต่อ API สำเร็จ")
	case <-time.After(API_TIMEOUT):
		fmt.Printf("\n❌ API connection timeout after %v\n", API_TIMEOUT)
		ErrorLogger.Printf("การเชื่อมต่อ API หมดเวลาหลังจาก %v", API_TIMEOUT)

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

		// รอให้ผู้ใช้กดปุ่มเพื่อปิดโปรแกรม
		fmt.Println("\nPress Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}

	// แสดงข้อความว่าแอปพลิเคชันพร้อมใช้งาน
	fmt.Println("\n✅ Application is ready!")
	InfoLogger.Println("แอปพลิเคชันพร้อมใช้งาน")
	fmt.Printf("- REST API running on port: %s\n", apiPort)
	fmt.Printf("- TCP Server running on port: %s\n", config.Config.TCPServerPort)
	fmt.Printf("- Connected to API: %s\n\n", config.Config.GetAPIURL())

	// สำหรับ Windows GUI
	if runtime.GOOS == "windows" {
		// รอให้ผู้ใช้กดปุ่มเพื่อปิดโปรแกรม
		fmt.Println("Press Enter to exit...")
		fmt.Scanln()
	} else {
		// สำหรับ Mac/Linux รอให้ REST server หยุดทำงาน
		err := <-restServerChan
		if err != nil {
			ErrorLogger.Printf("REST API server หยุดทำงานเนื่องจาก: %v", err)
		}
	}
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
