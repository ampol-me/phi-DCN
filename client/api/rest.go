package api

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"phi-DCN/client/config"
)

// StartRESTServer เริ่ม REST API server
func StartRESTServer(port string) {
	app := fiber.New(fiber.Config{
		AppName: "DCN TCP Bridge API",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))

	// API endpoints
	api := app.Group("/api")
	api.Get("/config", handleGetConfig)
	api.Post("/config", handleUpdateConfig)
	api.Get("/status", handleStatus)
	api.Get("/test", handleAPITest)
	api.Get("/start", handleStartServer)
	api.Get("/stop", handleStopServer)
	api.Get("/clients", handleClients)
	api.Get("/mics", handleMics)

	// เริ่ม server
	fmt.Printf("🚀 REST API running on port %s\n", port)
	log.Fatal(app.Listen(":" + port))
}

// handleGetConfig จัดการคำขอดูการตั้งค่า
func handleGetConfig(c *fiber.Ctx) error {
	return c.JSON(config.Config)
}

// handleUpdateConfig จัดการคำขออัปเดตการตั้งค่า
func handleUpdateConfig(c *fiber.Ctx) error {
	var newConfig config.AppConfig
	if err := c.BodyParser(&newConfig); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid data",
		})
	}

	// อัปเดตค่าตั้งค่า
	if newConfig.APIHost != "" {
		config.Config.APIHost = newConfig.APIHost
	}
	if newConfig.APIPort != "" {
		config.Config.APIPort = newConfig.APIPort
	}
	if newConfig.APIPath != "" {
		config.Config.APIPath = newConfig.APIPath
	}
	if newConfig.APIKey != "" {
		config.Config.APIKey = newConfig.APIKey
	}
	if newConfig.TCPServerPort != "" {
		config.Config.TCPServerPort = newConfig.TCPServerPort
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Configuration updated successfully",
	})
}

// handleStatus แสดงสถานะปัจจุบัน
func handleStatus(c *fiber.Ctx) error {
	status := fiber.Map{
		"apiStatus":    config.Config.APIStatus,
		"serverStatus": config.Config.TCPServerStatus,
	}

	return c.JSON(status)
}

// handleAPITest ทดสอบการเชื่อมต่อกับ API
func handleAPITest(c *fiber.Ctx) error {
	err := TestConnection()
	if err != nil {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Connected to API successfully",
	})
}

// handleStartServer เริ่ม TCP Server
func handleStartServer(c *fiber.Ctx) error {
	// ตรวจสอบว่า TCP Server กำลังทำงานอยู่หรือไม่
	if config.IsServerRunning() {
		return c.JSON(fiber.Map{
			"status":  "error",
			"message": "TCP Server is already running",
		})
	}

	// เริ่ม TCP Server ในพื้นหลัง
	go func() {
		config.StartTCPServer()
	}()

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Starting TCP Server",
	})
}

// handleStopServer หยุด TCP Server
func handleStopServer(c *fiber.Ctx) error {
	config.StopTCPServer()

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "TCP Server stopped",
	})
}

// handleClients แสดงรายการ clients ที่เชื่อมต่ออยู่
func handleClients(c *fiber.Ctx) error {
	clients := config.Config.GetActiveClients()

	return c.JSON(fiber.Map{
		"count":   len(clients),
		"clients": clients,
	})
}

// handleMics แสดงรายการไมค์ที่เปิดอยู่
func handleMics(c *fiber.Ctx) error {
	mics := config.Config.GetActiveMics()

	return c.JSON(fiber.Map{
		"count": len(mics),
		"mics":  mics,
	})
}
