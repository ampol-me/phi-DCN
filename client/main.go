package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"phi-DCN/client/api"
	"phi-DCN/client/config"
	"phi-DCN/client/proxy"
)

const (
	REST_API_PORT = "3002" // REST API Port
)

func main() {
	fmt.Println("======= Phila Live DCN - TCP Bridge =======")
	fmt.Println("Initializing application...")

	// ตั้งค่าเริ่มต้น
	config.InitServer()
	proxy.Setup()

	// ตรวจสอบพอร์ตสำหรับ REST API จากอาร์กิวเมนต์
	apiPort := REST_API_PORT
	if len(os.Args) > 1 {
		apiPort = os.Args[1]
	}

	// จัดการกับสัญญาณหยุดการทำงาน (Ctrl+C)
	setupSignalHandler()

	// เริ่ม REST API ด้วย GoFiber
	fmt.Println("🔄 Using GoFiber for REST API")
	fmt.Printf("🚀 Starting REST API on port %s\n", apiPort)
	fmt.Println("You can use the following API endpoints:")
	fmt.Println("- GET  /api/status      : View connection status")
	fmt.Println("- GET  /api/clients     : View connected clients")
	fmt.Println("- GET  /api/mics        : View active microphones")
	fmt.Println("Press Ctrl+C to exit the application")

	api.StartRESTServer(apiPort)
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
