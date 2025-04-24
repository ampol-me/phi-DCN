package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"phi-DCN/client/api"
	"phi-DCN/client/config"
	"phi-DCN/client/proxy"
)

const (
	REST_API_PORT = "3002" // REST API Port
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
