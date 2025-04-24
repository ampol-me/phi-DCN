package main

import (
	"fmt"
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
)

func main() {
	fmt.Println("Initializing application...")

	// ตั้งค่าเริ่มต้น
	config.InitServer()
	proxy.Setup()

	// ตรวจสอบพอร์ตสำหรับ REST API จากอาร์กิวเมนต์
	apiPort := REST_API_PORT
	if len(os.Args) > 1 {
		apiPort = os.Args[1]
	}

	// ตรวจสอบการเชื่อมต่อ API
	fmt.Println("Checking API connection...")
	fmt.Printf("Trying to connect to API at: %s\n", config.Config.GetAPIURL())
	apiCheckChan := make(chan error, 1)
	go func() {
		apiCheckChan <- api.TestConnection()
	}()

	select {
	case err := <-apiCheckChan:
		if err != nil {
			fmt.Printf("\n❌ API connection failed: %v\n", err)
			fmt.Println("\nPossible causes:")
			fmt.Println("1. API server is not running")
			fmt.Println("2. Incorrect API configuration")
			fmt.Println("3. Network connectivity issues")
			fmt.Println("\nPlease check your configuration:")
			fmt.Printf("- API Host: %s\n", config.Config.APIHost)
			fmt.Printf("- API Port: %s\n", config.Config.APIPort)
			fmt.Printf("- API Path: %s\n", config.Config.APIPath)
			fmt.Printf("- API Key: %s\n", config.Config.APIKey)
			fmt.Println("\nPlease update your configuration before starting the server")
			os.Exit(1)
		}
		fmt.Println("✅ API connection successful")
	case <-time.After(API_TIMEOUT):
		fmt.Printf("\n❌ API connection timeout after %v\n", API_TIMEOUT)
		fmt.Println("\nPossible causes:")
		fmt.Println("1. API server is not responding")
		fmt.Println("2. Network latency is too high")
		fmt.Println("3. Firewall blocking the connection")
		fmt.Println("\nPlease check your configuration:")
		fmt.Printf("- API Host: %s\n", config.Config.APIHost)
		fmt.Printf("- API Port: %s\n", config.Config.APIPort)
		fmt.Printf("- API Path: %s\n", config.Config.APIPath)
		fmt.Printf("- API Key: %s\n", config.Config.APIKey)
		fmt.Println("\nPlease update your configuration before starting the server")
		os.Exit(1)
	}

	// จัดการกับสัญญาณหยุดการทำงาน (Ctrl+C)
	setupSignalHandler()

	// เริ่ม REST API
	api.StartRESTServer(apiPort)

	// สำหรับ Windows GUI
	if runtime.GOOS == "windows" {
		// รอให้ผู้ใช้กดปุ่มเพื่อปิดโปรแกรม
		fmt.Println("Press Enter to exit...")
		fmt.Scanln()
	}
}

// setupSignalHandler จัดการกับสัญญาณหยุดการทำงาน
func setupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nShutting down application...")
		config.StopTCPServer()
		fmt.Println("Application closed successfully")
		os.Exit(0)
	}()
}
