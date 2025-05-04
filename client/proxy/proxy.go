package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"phi-DCN/client/api"
	"phi-DCN/client/config"
	"phi-DCN/client/license"
	"phi-DCN/client/xml"
)

// ตัวแปรสำหรับการ logging
var (
	activityLogger *log.Logger
	ErrorLogger    *log.Logger
	DebugLogger    *log.Logger
)

// ค่าคงที่สำหรับการตั้งค่า
const (
	MaxConnections     = 5           // จำนวนการเชื่อมต่อสูงสุด
	ReadTimeout        = 8 * 60 * 60 // timeout สำหรับการอ่านข้อมูลจาก client (วินาที) - 8 ชั่วโมง
	WriteTimeout       = 5           // timeout สำหรับการเขียนข้อมูลไปยัง client (วินาที)
	RetryAttempts      = 3           // จำนวนครั้งในการ retry เมื่อเกิดข้อผิดพลาด
	RetryDelay         = 500         // ระยะเวลาในการ retry (มิลลิวินาที)
	APIPollingInterval = 750         // ระยะเวลาในการดึงข้อมูลจาก API (มิลลิวินาที)
	KeepAliveInterval  = 30 * 60     // ระยะเวลาในการส่ง keep-alive (วินาที) - 30 นาที
)

// โครงสร้างสำหรับเก็บข้อมูล client
type Client struct {
	conn       net.Conn
	id         int
	lastActive time.Time
	sendQueue  chan []byte   // ช่องทางสำหรับส่งข้อมูลไปยัง client
	done       chan struct{} // ช่องทางสำหรับสัญญาณการปิดการเชื่อมต่อ
}

// ฟังก์ชันสำหรับส่งข้อมูลไปยัง client
func (c *Client) Send(data []byte) error {
	select {
	case c.sendQueue <- data:
		return nil
	case <-time.After(time.Duration(WriteTimeout) * time.Second):
		return errors.New("send timeout")
	}
}

// ProxyServer จัดการการเชื่อมต่อของ clients
type ProxyServer struct {
	clients       map[int]*Client
	nextID        int
	clientLock    sync.RWMutex
	bufferPool    sync.Pool // pool สำหรับ buffer เพื่อลดการจัดสรรหน่วยความจำใหม่
	isRunning     bool
	stopChan      chan struct{}
	connCount     int // จำนวนการเชื่อมต่อปัจจุบัน
	connCountLock sync.Mutex
	metrics       *Metrics // เก็บสถิติการทำงาน
	listener      net.Listener
	pid           int // เพิ่ม PID
}

// Metrics เก็บสถิติการทำงาน
type Metrics struct {
	TotalConnections   int64     // จำนวนการเชื่อมต่อทั้งหมดตั้งแต่เริ่มโปรแกรม
	CurrentConnections int32     // จำนวนการเชื่อมต่อปัจจุบัน
	TotalErrors        int64     // จำนวนข้อผิดพลาดทั้งหมด
	APIErrors          int64     // จำนวนข้อผิดพลาดจาก API
	ClientErrors       int64     // จำนวนข้อผิดพลาดจาก client
	LastError          string    // ข้อผิดพลาดล่าสุด
	LastErrorTime      time.Time // เวลาที่เกิดข้อผิดพลาดล่าสุด
	mu                 sync.RWMutex
}

// สร้าง logging system
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

	// สร้างโฟลเดอร์ logs ถ้ายังไม่มี
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", 0755)
	}

	// เปิดไฟล์ log
	//currentTime := time.Now().Format("2006-01-02")
	// เปิดไฟล์ log
	activityFile, err := os.OpenFile(fmt.Sprintf("%s/app_activity.log", dayDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	// เปิดไฟล์ log
	systemErrorFile, err := os.OpenFile(fmt.Sprintf("%s/system_error.log", dayDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	// สร้าง logger
	activityLogger = log.New(activityFile, "ACTIVITY: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(systemErrorFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// RecordError บันทึกข้อผิดพลาด
func (m *Metrics) RecordError(errorType string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalErrors++
	m.LastError = err.Error()
	m.LastErrorTime = time.Now()

	if errorType == "API" {
		m.APIErrors++
	} else if errorType == "Client" {
		m.ClientErrors++
	}

	// บันทึกข้อผิดพลาดลงไฟล์ log
	ErrorLogger.Printf("%s Error: %v", errorType, err)
}

// สร้าง ProxyServer ใหม่
func NewProxyServer() *ProxyServer {
	return &ProxyServer{
		clients:   make(map[int]*Client),
		nextID:    1,
		isRunning: false,
		stopChan:  make(chan struct{}),
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 4096)
			},
		},
		metrics: &Metrics{},
	}
}

// เพิ่ม client ใหม่
func (p *ProxyServer) AddClient(conn net.Conn) (*Client, error) {
	p.connCountLock.Lock()
	defer p.connCountLock.Unlock()

	// ตรวจสอบจำนวนการเชื่อมต่อ
	if p.connCount >= MaxConnections {
		return nil, errors.New("เกินจำนวนการเชื่อมต่อสูงสุด")
	}

	p.connCount++
	p.metrics.mu.Lock()
	p.metrics.TotalConnections++
	p.metrics.CurrentConnections++
	p.metrics.mu.Unlock()

	p.clientLock.Lock()
	defer p.clientLock.Unlock()

	client := &Client{
		conn:       conn,
		id:         p.nextID,
		lastActive: time.Now(),
		sendQueue:  make(chan []byte, 100), // buffer สำหรับส่งข้อมูล
		done:       make(chan struct{}),
	}
	p.clients[p.nextID] = client
	p.nextID++

	remoteAddr := conn.RemoteAddr().String()
	ipAddress := strings.Split(remoteAddr, ":")[0]

	// เพิ่ม client ลงในรายการ
	clientInfo := fmt.Sprintf("ID: %d, Connected at: %s", client.id, time.Now().Format("15:04:05"))
	config.Config.AddActiveClient(ipAddress, clientInfo)

	// เริ่ม goroutine สำหรับส่งข้อมูลไปยัง client
	go p.handleClientSend(client)

	fmt.Printf("👥 Client %d connected: %s\n", client.id, remoteAddr)
	activityLogger.Printf("Client %d connected: %s", client.id, remoteAddr)
	return client, nil
}

// ลบ client
func (p *ProxyServer) RemoveClient(id int) {
	p.clientLock.Lock()
	client, exists := p.clients[id]
	if exists {
		delete(p.clients, id)
		p.clientLock.Unlock()

		// ปิด sendQueue เพื่อหยุด goroutine ส่งข้อมูล
		close(client.done)
		close(client.sendQueue)

		remoteAddr := client.conn.RemoteAddr().String()
		ipAddress := strings.Split(remoteAddr, ":")[0]

		// ลบ client ออกจากรายการ
		config.Config.RemoveActiveClient(ipAddress)

		fmt.Printf("👋 Client %d disconnected: %s\n", id, remoteAddr)
		activityLogger.Printf("Client %d disconnected: %s", id, remoteAddr)

		client.conn.Close()

		p.connCountLock.Lock()
		p.connCount--
		p.metrics.mu.Lock()
		p.metrics.CurrentConnections--
		p.metrics.mu.Unlock()
		p.connCountLock.Unlock()
	} else {
		p.clientLock.Unlock()
	}
}

// handleClientSend จัดการการส่งข้อมูลไปยัง client
func (p *ProxyServer) handleClientSend(client *Client) {
	for {
		select {
		case data, ok := <-client.sendQueue:
			if !ok {
				return
			}
			// ตั้งค่า timeout สำหรับการเขียนข้อมูล
			client.conn.SetWriteDeadline(time.Now().Add(time.Duration(WriteTimeout) * time.Second))
			_, err := client.conn.Write(data)
			if err != nil {
				p.metrics.RecordError("Client", fmt.Errorf("ไม่สามารถส่งข้อมูลไปยัง Client %d: %v", client.id, err))
				p.RemoveClient(client.id)
				return
			}
			client.lastActive = time.Now()
		case <-client.done:
			return
		}
	}
}

// ส่งข้อมูลไปยังทุก clients
func (p *ProxyServer) Broadcast(data []byte) {
	p.clientLock.RLock()
	defer p.clientLock.RUnlock()

	for _, client := range p.clients {
		// ใช้ goroutine ในการส่งเพื่อไม่ให้การส่งข้อมูลไปยัง client ที่ช้าทำให้การส่งข้อมูลไปยัง client อื่นล่าช้า
		go func(c *Client, d []byte) {
			if err := c.Send(d); err != nil {
				p.metrics.RecordError("Client", fmt.Errorf("ไม่สามารถส่งข้อมูลไปยัง Client %d: %v", c.id, err))
				p.RemoveClient(c.id)
			}
		}(client, data)
	}
}

// จัดการการเชื่อมต่อจาก client
func HandleClientConnection(proxy *ProxyServer, conn net.Conn) {
	// ตั้งค่า keep-alive
	tcpConn := conn.(*net.TCPConn)
	tcpConn.SetKeepAlive(true)
	tcpConn.SetKeepAlivePeriod(time.Duration(KeepAliveInterval) * time.Second)

	// ตั้งค่า timeout
	conn.SetReadDeadline(time.Now().Add(time.Duration(ReadTimeout) * time.Second))

	client, err := proxy.AddClient(conn)
	if err != nil {
		conn.Close()
		proxy.metrics.RecordError("Client", err)
		fmt.Printf("❌ Cannot add client: %v\n", err)
		return
	}

	defer proxy.RemoveClient(client.id)

	// รอรับข้อมูลจาก client
	buffer := proxy.bufferPool.Get().([]byte)
	defer proxy.bufferPool.Put(buffer)

	for {
		// ตั้งค่า timeout สำหรับการอ่านข้อมูล
		conn.SetReadDeadline(time.Now().Add(time.Duration(ReadTimeout) * time.Second))
		_, err := conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// ถ้าเป็น timeout ให้ตรวจสอบว่า client ยังคงเชื่อมต่ออยู่หรือไม่
				if time.Since(client.lastActive) > time.Duration(ReadTimeout)*time.Second {
					proxy.metrics.RecordError("Client", fmt.Errorf("Client %d timeout", client.id))
					return
				}
				// ถ้ายังเชื่อมต่ออยู่ ให้ตั้งค่า timeout ใหม่
				continue
			}
			// ถ้าเป็น error อื่นๆ ให้ปิดการเชื่อมต่อ
			return
		}
		client.lastActive = time.Now()
	}
}

// retry ทำซ้ำเมื่อเกิดข้อผิดพลาด
func retry(attempts int, sleep time.Duration, f func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = f()
		if err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(sleep * time.Millisecond)
		}
	}
	return err
}

// ฟังก์ชันดึงข้อมูลจาก API และส่งไปยัง clients
func (p *ProxyServer) ProcessAndBroadcast() {
	speakerStates := make(map[int]bool)
	var currentSpeakers []api.Speaker
	var allMicsOffSent bool

	for {
		speakers, err := api.GetSpeakers()
		if err != nil {
			fmt.Println("⚠️ Cannot fetch speakers data:", err)
			time.Sleep(time.Second)
			continue
		}

		// ตรวจสอบกรณีที่ API ส่งค่ากลับมาเป็น [] (ปิดทั้งหมด)
		if len(speakers) == 0 {
			// ส่ง SeatActivity OFF และ DiscussionActivity ว่างแค่ครั้งเดียว
			if !allMicsOffSent {
				// ส่ง SeatActivity OFF สำหรับไมค์ที่เปิดอยู่ทั้งหมด
				for _, speaker := range currentSpeakers {
					seatXML := xml.GenerateSeatXML(speaker, false)
					header := make([]byte, 8)
					binary.LittleEndian.PutUint32(header[0:4], 5)
					binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
					p.Broadcast(append(header, seatXML...))

					// เคลียร์สถานะไมค์ใน config
					config.Config.UpdateActiveMic(speaker.SeatName, false)

				}

				// ส่ง DiscussionActivity ว่าง
				discussionXML := xml.GenerateDiscussionXML([]api.Speaker{})
				header := make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 3)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(discussionXML)))
				p.Broadcast(append(header, discussionXML...))

				// ล้างข้อมูลไมค์ทั้งหมด
				currentSpeakers = []api.Speaker{}
				speakerStates = make(map[int]bool)
				allMicsOffSent = true
				activityLogger.Println("SeatActivity OFF: All mics off")
				fmt.Println("SeatActivity OFF: All mics off")
			}
			time.Sleep(time.Second)
			continue
		}

		// ถ้ามีไมค์เปิดใหม่ ให้รีเซ็ตสถานะ allMicsOffSent
		allMicsOffSent = false

		// ตรวจสอบไมค์ที่เปิดใหม่
		for _, speaker := range speakers {
			if speaker.MicOn == 1 && !speakerStates[speaker.ID] {
				// ส่ง SeatActivity ON สำหรับไมค์ที่เปิดใหม่
				seatXML := xml.GenerateSeatXML(speaker, true)
				header := make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 5)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
				p.Broadcast(append(header, seatXML...))

				// บันทึกกิจกรรม
				activityLogger.Printf("Mic ON: %s", speaker.SeatName)
				fmt.Printf("Mic ON: %s\n", speaker.SeatName)

				// อัปเดตสถานะ
				speakerStates[speaker.ID] = true
				currentSpeakers = append(currentSpeakers, speaker)
				config.Config.UpdateActiveMic(speaker.SeatName, true)

				// ส่ง DiscussionActivity แสดงไมค์ทั้งหมดจนถึงลำดับนี้
				discussionXML := xml.GenerateDiscussionXML(currentSpeakers)
				header = make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 3)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(discussionXML)))
				p.Broadcast(append(header, discussionXML...))

				// รอสักครู่ก่อนส่งไมค์ตัวต่อไป
				time.Sleep(500 * time.Millisecond)
			}
		}

		// ตรวจสอบไมค์ที่ปิด
		for i := 0; i < len(currentSpeakers); i++ {
			speaker := currentSpeakers[i]
			found := false
			for _, s := range speakers {
				if s.ID == speaker.ID {
					found = true
					if s.MicOn == 0 {
						// ส่ง SeatActivity OFF สำหรับไมค์ที่ปิด
						seatXML := xml.GenerateSeatXML(speaker, false)
						header := make([]byte, 8)
						binary.LittleEndian.PutUint32(header[0:4], 5)
						binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
						p.Broadcast(append(header, seatXML...))
						// บันทึกกิจกรรม
						activityLogger.Printf("Mic OFF: %s", speaker.SeatName)
						fmt.Printf("Mic OFF: %s\n", speaker.SeatName)

						// เคลียร์สถานะไมค์ใน config
						config.Config.UpdateActiveMic(speaker.SeatName, false)

						// อัปเดตสถานะ
						speakerStates[speaker.ID] = false
						currentSpeakers = append(currentSpeakers[:i], currentSpeakers[i+1:]...)
						i--

						// ส่ง DiscussionActivity แสดงไมค์ที่เปิดอยู่ทั้งหมด
						discussionXML := xml.GenerateDiscussionXML(currentSpeakers)
						header = make([]byte, 8)
						binary.LittleEndian.PutUint32(header[0:4], 3)
						binary.LittleEndian.PutUint32(header[4:8], uint32(len(discussionXML)))
						p.Broadcast(append(header, discussionXML...))
					}
					break
				}
			}

			if !found {
				// ส่ง SeatActivity OFF สำหรับไมค์ที่ปิด
				seatXML := xml.GenerateSeatXML(speaker, false)
				header := make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 5)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
				p.Broadcast(append(header, seatXML...))
				// บันทึกกิจกรรม
				activityLogger.Printf("Mic OFF: %s", speaker.SeatName)
				fmt.Printf("Mic OFF: %s\n", speaker.SeatName)

				// เคลียร์สถานะไมค์ใน config
				config.Config.UpdateActiveMic(speaker.SeatName, false)

				// อัปเดตสถานะ
				speakerStates[speaker.ID] = false
				currentSpeakers = append(currentSpeakers[:i], currentSpeakers[i+1:]...)
				i--

				// ส่ง DiscussionActivity แสดงไมค์ที่เปิดอยู่ทั้งหมด
				discussionXML := xml.GenerateDiscussionXML(currentSpeakers)
				header = make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 3)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(discussionXML)))
				p.Broadcast(append(header, discussionXML...))
			}
		}
		time.Sleep(time.Second)
	}
}

// cleanInactiveClients ตรวจสอบและลบ clients ที่ไม่ได้ใช้งานเป็นเวลานาน
func (p *ProxyServer) cleanInactiveClients() {
	ticker := time.NewTicker(3000 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.clientLock.Lock()
			now := time.Now()
			for id, client := range p.clients {
				if now.Sub(client.lastActive) > time.Duration(ReadTimeout)*time.Second {
					p.clientLock.Unlock()
					p.metrics.RecordError("Client", fmt.Errorf("Client %d inactive for too long", id))
					p.RemoveClient(id)
					p.clientLock.Lock()
				}
			}
			p.clientLock.Unlock()
		}
	}
}

// StartProxy เริ่มทำงาน Proxy
func StartProxy() {
	// เริ่มต้น logging system
	initLogging()

	// โหลดและตรวจสอบ license
	if err := license.LoadLicense(); err != nil {
		ErrorLogger.Printf("Failed to load license: %v", err)
		fmt.Printf("❌ Failed to load license: %v\n", err)
		return
	}

	if err := license.ValidateLicense(); err != nil {
		ErrorLogger.Printf("License validation failed: %v", err)
		fmt.Printf("❌ License validation failed: %v\n", err)
		return
	}

	// เริ่มการตรวจสอบ license เป็นระยะ
	license.StartLicenseCheck()

	// สร้าง proxy server
	proxy := NewProxyServer()
	proxy.isRunning = true
	proxy.pid = os.Getpid() // เก็บ PID ของ process

	// เริ่ม goroutine สำหรับตรวจสอบ clients ที่ไม่ได้ใช้งาน
	go proxy.cleanInactiveClients()

	// อัปเดตสถานะของ server
	config.Config.UpdateTCPServerStatus("Initializing...")
	activityLogger.Println("TCP Server initializing...")

	// เริ่ม proxy server ที่ port 20000
	var err error
	port := "20000" // ใช้ port 20000 เท่านั้น

	// สร้าง TCP listener ด้วย SO_REUSEADDR
	addr, err := net.ResolveTCPAddr("tcp", ":"+port)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot resolve TCP address: %v", err)
		fmt.Printf("❌ %s\n", errMsg)
		ErrorLogger.Println(errMsg)
		config.Config.UpdateTCPServerStatus(errMsg)
		return
	}

	proxy.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot start TCP server: %v", err)
		fmt.Printf("❌ %s\n", errMsg)
		ErrorLogger.Println(errMsg)
		config.Config.UpdateTCPServerStatus(errMsg)
		return
	}

	// ถ้าเริ่ม server สำเร็จ
	if proxy.listener != nil {
		// อัปเดตสถานะ และแสดงข้อความ
		statusMsg := fmt.Sprintf("Listening on port %s (PID: %d)", port, proxy.pid)
		config.Config.UpdateTCPServerStatus("Running")
		fmt.Printf("🚀 %s\n", statusMsg)
		activityLogger.Println(statusMsg)

		// เริ่ม goroutine สำหรับดึงข้อมูลจาก API
		go proxy.ProcessAndBroadcast()

		// รับการเชื่อมต่อจาก clients
		for {
			conn, err := proxy.listener.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					break
				}
				ErrorLogger.Printf("Error accepting connection: %v", err)
				continue
			}

			// จัดการการเชื่อมต่อในแต่ละ goroutine
			go HandleClientConnection(proxy, conn)
		}
	}
}

// StopProxy หยุดการทำงานของ Proxy
func StopProxy(proxy *ProxyServer) {
	if proxy.isRunning {
		// ส่งสัญญาณให้หยุดการทำงาน
		close(proxy.stopChan)
		proxy.isRunning = false

		// ปิดการเชื่อมต่อกับทุก clients
		proxy.clientLock.Lock()
		for id, client := range proxy.clients {
			// ปิดการเชื่อมต่อแบบ graceful
			if tcpConn, ok := client.conn.(*net.TCPConn); ok {
				tcpConn.SetLinger(0) // ปิดการเชื่อมต่อทันที
				tcpConn.CloseWrite() // ปิดการเขียน
				tcpConn.CloseRead()  // ปิดการอ่าน
				tcpConn.Close()      // ปิดการเชื่อมต่อ
			}
			delete(proxy.clients, id)
		}
		proxy.clientLock.Unlock()

		// ปิด listener เพื่อปล่อย port
		if proxy.listener != nil {
			// ปิด listener แบบ graceful
			if tcpListener, ok := proxy.listener.(*net.TCPListener); ok {
				tcpListener.SetDeadline(time.Now()) // บังคับให้ปิดการเชื่อมต่อ
				tcpListener.Close()                 // ปิด listener
			}
			proxy.listener = nil
		}

		// รอให้การเชื่อมต่อทั้งหมดถูกปิด
		time.Sleep(2 * time.Second)

		// ถ้ายังมี process ที่ใช้ port 20000 อยู่ และเป็น process เดิม ให้ kill process นั้น
		if proxy.pid > 0 {
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("taskkill", "/F", "/PID", strconv.Itoa(proxy.pid))
			} else {
				cmd = exec.Command("kill", "-9", strconv.Itoa(proxy.pid))
			}
			cmd.Run()
		}

		activityLogger.Println("TCP Server stopped")
		config.Config.UpdateTCPServerStatus("Stopped")
	}
}

// GetServerMetrics คืนค่าสถิติของ server
func (p *ProxyServer) GetServerMetrics() *Metrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	// สร้าง copy ของ metrics เพื่อป้องกันการแก้ไขข้อมูลจากภายนอก
	return &Metrics{
		TotalConnections:   p.metrics.TotalConnections,
		CurrentConnections: p.metrics.CurrentConnections,
		TotalErrors:        p.metrics.TotalErrors,
		APIErrors:          p.metrics.APIErrors,
		ClientErrors:       p.metrics.ClientErrors,
		LastError:          p.metrics.LastError,
		LastErrorTime:      p.metrics.LastErrorTime,
	}
}

// ValidateInput ตรวจสอบความถูกต้องของข้อมูลที่รับเข้ามา
func ValidateInput(data []byte) bool {
	// ตรวจสอบความถูกต้องของข้อมูล
	if len(data) < 8 {
		return false
	}

	// ตรวจสอบความถูกต้องของ header
	messageType := binary.LittleEndian.Uint32(data[0:4])
	messageLength := binary.LittleEndian.Uint32(data[4:8])

	if messageType != 3 && messageType != 5 {
		return false
	}

	if int(messageLength) != len(data)-8 {
		return false
	}

	return true
}

// SanitizeOutput ทำความสะอาดข้อมูลก่อนส่งออก
func SanitizeOutput(data []byte) []byte {
	// ทำความสะอาดข้อมูลก่อนส่งออก
	// ในที่นี้ เราเพียงตรวจสอบความถูกต้องของข้อมูล
	if !ValidateInput(data) {
		// ถ้าข้อมูลไม่ถูกต้อง ให้ส่งข้อมูลว่าง
		header := make([]byte, 8)
		binary.LittleEndian.PutUint32(header[0:4], 3)
		binary.LittleEndian.PutUint32(header[4:8], 0)
		return header
	}

	return data
}

// Setup เตรียมพร้อมสำหรับการทำงานของ Proxy
func Setup() {
	// โหลดการตั้งค่าจากไฟล์ config.ini
	if err := config.InitConfig(); err != nil {
		log.Fatalf("ไม่สามารถโหลดการตั้งค่าได้: %v", err)
	}

	// ตั้งค่า StartServerFunc
	config.StartServerFunc = StartProxy

	// ตั้งค่าเริ่มต้นสำหรับ proxy
	proxy := NewProxyServer()
	proxy.isRunning = false
	config.Config.UpdateTCPServerStatus("Stopped")
}
