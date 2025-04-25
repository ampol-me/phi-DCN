package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"phi-DCN/client/api"
	"phi-DCN/client/config"
	"phi-DCN/client/xml"
)

// ตัวแปรสำหรับการ logging
var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
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
	// สร้างโฟลเดอร์ logs ถ้ายังไม่มี
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", 0755)
	}

	// เปิดไฟล์ log
	currentTime := time.Now().Format("2006-01-02")
	infoFile, err := os.OpenFile(fmt.Sprintf("logs/info_%s.log", currentTime), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	errorFile, err := os.OpenFile(fmt.Sprintf("logs/error_%s.log", currentTime), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ไม่สามารถเปิดไฟล์ log ได้: %v", err)
	}

	// สร้าง logger
	InfoLogger = log.New(infoFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(errorFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
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
	InfoLogger.Printf("Client %d connected: %s", client.id, remoteAddr)
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
		InfoLogger.Printf("Client %d disconnected: %s", id, remoteAddr)

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
	// เก็บสถานะล่าสุดของไมค์ทั้งหมด
	var lastSpeakers []api.Speaker
	// เก็บสถานะปัจจุบันของไมค์แต่ละตัว (true = on, false = off)
	speakerStates := make(map[int]bool)
	// เก็บลำดับการส่งข้อมูล
	var currentSpeakers []api.Speaker
	// เก็บเวลาที่ไมค์ปิดทั้งหมด
	var allMicsOffTime time.Time
	// เก็บสถานะว่ากำลังรอรีเซ็ตหรือไม่
	var waitingForReset bool
	// เก็บจำนวนครั้งที่ API ส่งข้อมูลว่างกลับมา
	var emptyResponseCount int

	for {
		// ดึงข้อมูลไมค์ที่เปิดอยู่จาก API
		speakers, err := api.GetSpeakers()
		if err != nil {
			// ... จัดการ error
			continue
		}

		// ตรวจสอบว่าข้อมูลจาก API เป็นข้อมูลว่างหรือไม่
		if len(speakers) == 0 {
			emptyResponseCount++
			// ถ้า API ส่งข้อมูลว่างกลับมาเกิน 3 ครั้งติดต่อกัน ให้รีเซ็ตสถานะ
			if emptyResponseCount >= 3 {
				// รีเซ็ตสถานะของไมค์ทั้งหมด
				speakerStates = make(map[int]bool)
				currentSpeakers = []api.Speaker{}
				waitingForReset = false
				emptyResponseCount = 0
				fmt.Println("🔄 Reset all microphone states due to empty API response")
				InfoLogger.Println("Reset all microphone states due to empty API response")
			}
			time.Sleep(time.Second)
			continue
		}

		// รีเซ็ตตัวนับข้อมูลว่าง
		emptyResponseCount = 0

		// สร้าง map สำหรับไมค์ที่เปิดอยู่ตอนนี้
		currentActiveMics := make(map[int]bool)
		for _, speaker := range speakers {
			currentActiveMics[speaker.ID] = true
		}

		// ตรวจสอบว่ามีไมค์ที่เปิดอยู่หรือไม่
		hasActiveMic := false
		for _, speaker := range speakers {
			if speaker.MicOn == 1 {
				hasActiveMic = true
				break
			}
		}

		// ถ้าไม่มีไมค์ที่เปิดอยู่
		if !hasActiveMic {
			// ถ้ายังไม่ได้เริ่มนับเวลา
			if !waitingForReset {
				allMicsOffTime = time.Now()
				waitingForReset = true
			} else {
				// ถ้าเวลาผ่านไป 2 วินาที
				if time.Since(allMicsOffTime) >= 2*time.Second {
					// รีเซ็ตสถานะของไมค์ทั้งหมด
					speakerStates = make(map[int]bool)
					currentSpeakers = []api.Speaker{}
					waitingForReset = false
					fmt.Println("🔄 Reset all microphone states after 2 seconds of inactivity")
					InfoLogger.Println("Reset all microphone states after 2 seconds of inactivity")
				}
			}
		} else {
			// ถ้ามีไมค์ที่เปิดอยู่ ให้รีเซ็ตตัวนับเวลา
			waitingForReset = false
		}

		// ตรวจสอบไมค์ที่เปิดอยู่ (อยู่ใน speakers)
		for _, speaker := range speakers {
			// ถ้าไมค์นี้ยังไม่เคยเปิดมาก่อน หรือเคยปิดไปแล้ว
			if !speakerStates[speaker.ID] {
				// แสดงสถานะไมค์เปิด
				fmt.Printf("🎙️ Mic ON: %s (ID: %d)\n", speaker.SeatName, speaker.ID)
				InfoLogger.Printf("Mic ON: %s (ID: %d)", speaker.SeatName, speaker.ID)

				// ส่ง SeatActivity เมื่อไมค์เปิด
				seatXML := xml.GenerateSeatXML(speaker, true)
				header := make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 5)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
				p.Broadcast(append(header, seatXML...))

				// อัปเดตสถานะเป็นเปิด
				speakerStates[speaker.ID] = true

				// เพิ่มไมค์นี้เข้าไปในรายการปัจจุบัน
				currentSpeakers = append(currentSpeakers, speaker)

				// ส่ง DiscussionActivity แสดงไมค์ที่เปิดอยู่ทั้งหมดจนถึงลำดับนี้
				discussionXML := xml.GenerateDiscussionXML(currentSpeakers)
				header = make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 3)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(discussionXML)))
				p.Broadcast(append(header, discussionXML...))
			}
		}

		// ตรวจสอบไมค์ที่ปิด (ไม่อยู่ใน speakers แต่เคยเปิด)
		for id, state := range speakerStates {
			// ตรวจสอบว่าสถานะปัจจุบันของไมค์นี้เคยเปิดอยู่ และตอนนี้ไม่อยู่ในรายการไมค์ที่เปิดอยู่
			if state && !currentActiveMics[id] {
				// หาข้อมูลไมค์ที่ปิดจาก lastSpeakers
				for _, oldSpeaker := range lastSpeakers {
					if oldSpeaker.ID == id {
						// แสดงสถานะไมค์ปิด
						fmt.Printf("🎙️ Mic OFF: %s (ID: %d)\n", oldSpeaker.SeatName, oldSpeaker.ID)
						InfoLogger.Printf("Mic OFF: %s (ID: %d)", oldSpeaker.SeatName, oldSpeaker.ID)

						// ส่ง SeatActivity เมื่อไมค์ปิด
						seatXML := xml.GenerateSeatXML(oldSpeaker, false)
						header := make([]byte, 8)
						binary.LittleEndian.PutUint32(header[0:4], 5)
						binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
						p.Broadcast(append(header, seatXML...))

						// อัปเดตสถานะเป็นปิด
						speakerStates[id] = false

						// ลบไมค์นี้ออกจากรายการปัจจุบัน
						for i, s := range currentSpeakers {
							if s.ID == id {
								currentSpeakers = append(currentSpeakers[:i], currentSpeakers[i+1:]...)
								break
							}
						}

						// ส่ง DiscussionActivity แสดงไมค์ที่เปิดอยู่ทั้งหมด
						discussionXML := xml.GenerateDiscussionXML(currentSpeakers)
						header = make([]byte, 8)
						binary.LittleEndian.PutUint32(header[0:4], 3)
						binary.LittleEndian.PutUint32(header[4:8], uint32(len(discussionXML)))
						p.Broadcast(append(header, discussionXML...))
						break
					}
				}
			}
		}

		lastSpeakers = speakers
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

	// สร้าง proxy server
	proxy := NewProxyServer()
	proxy.isRunning = true

	// เริ่ม goroutine สำหรับตรวจสอบ clients ที่ไม่ได้ใช้งาน
	go proxy.cleanInactiveClients()

	// อัปเดตสถานะของ server
	config.Config.UpdateTCPServerStatus("Initializing...")
	InfoLogger.Println("TCP Server initializing...")

	// เริ่ม proxy server ด้วยการลอง port หลายครั้ง
	var proxyListener net.Listener
	var err error
	maxRetries := 5
	basePort := config.Config.TCPServerPort
	currentPort := basePort

	for i := 0; i < maxRetries; i++ {
		proxyListener, err = net.Listen("tcp", ":"+currentPort)
		if err == nil {
			break
		}

		// ถ้า port ถูกใช้งานอยู่แล้ว ให้ลอง port ถัดไป
		if strings.Contains(err.Error(), "address already in use") {
			portNum, _ := strconv.Atoi(currentPort)
			currentPort = strconv.Itoa(portNum + 1)
			continue
		}

		// ถ้าเกิด error อื่นๆ ให้แสดง error และหยุดการทำงาน
		errMsg := fmt.Sprintf("Cannot start TCP server: %v", err)
		fmt.Printf("❌ %s\n", errMsg)
		ErrorLogger.Println(errMsg)
		config.Config.UpdateTCPServerStatus(errMsg)
		return
	}

	// ถ้าเริ่ม server สำเร็จ
	if proxyListener != nil {
		// อัปเดตสถานะ และแสดงข้อความ
		statusMsg := fmt.Sprintf("Listening on port %s", currentPort)
		config.Config.UpdateTCPServerStatus(statusMsg)
		fmt.Printf("🚀 %s\n", statusMsg)
		InfoLogger.Println(statusMsg)

		// ถ้า port ที่ใช้ไม่ใช่ port ที่ตั้งค่าไว้ ให้บันทึกลงไฟล์ config
		if currentPort != basePort {
			config.Config.TCPServerPort = currentPort
			if err := config.SaveConfig(); err != nil {
				ErrorLogger.Printf("Cannot save config: %v", err)
			}
		}

		// เริ่ม goroutine สำหรับดึงข้อมูลจาก API
		go proxy.ProcessAndBroadcast()

		// รับการเชื่อมต่อจาก clients
		for {
			conn, err := proxyListener.Accept()
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
		close(proxy.stopChan)
		proxy.isRunning = false

		// ปิดการเชื่อมต่อกับทุก clients
		proxy.clientLock.Lock()
		for id, client := range proxy.clients {
			client.conn.Close()
			delete(proxy.clients, id)
		}
		proxy.clientLock.Unlock()

		InfoLogger.Println("TCP Server stopped")
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

	// เริ่มทำงาน Proxy
	go StartProxy()
}
