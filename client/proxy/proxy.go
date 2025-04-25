package proxy

import (
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"phi-DCN/client/api"
	"phi-DCN/client/config"
	"phi-DCN/client/xml"
)

// โครงสร้างสำหรับเก็บข้อมูล client
type Client struct {
	conn net.Conn
	id   int
}

// ฟังก์ชันสำหรับส่งข้อมูลไปยัง client
func (c *Client) Send(data []byte) error {
	_, err := c.conn.Write(data)
	return err
}

// ProxyServer จัดการการเชื่อมต่อของ clients
type ProxyServer struct {
	clients    map[int]*Client
	nextID     int
	clientLock sync.Mutex
	isRunning  bool
	stopChan   chan struct{}
}

// สร้าง ProxyServer ใหม่
func NewProxyServer() *ProxyServer {
	return &ProxyServer{
		clients:   make(map[int]*Client),
		nextID:    1,
		isRunning: false,
		stopChan:  make(chan struct{}),
	}
}

// เพิ่ม client ใหม่
func (p *ProxyServer) AddClient(conn net.Conn) *Client {
	p.clientLock.Lock()
	defer p.clientLock.Unlock()

	client := &Client{
		conn: conn,
		id:   p.nextID,
	}
	p.clients[p.nextID] = client
	p.nextID++

	remoteAddr := conn.RemoteAddr().String()
	ipAddress := strings.Split(remoteAddr, ":")[0]

	// เพิ่ม client ลงในรายการ
	clientInfo := fmt.Sprintf("ID: %d, Connected at: %s", client.id, time.Now().Format("15:04:05"))
	config.Config.AddActiveClient(ipAddress, clientInfo)

	fmt.Printf("👥 Client %d connected: %s\n", client.id, remoteAddr)
	return client
}

// ลบ client
func (p *ProxyServer) RemoveClient(id int) {
	p.clientLock.Lock()
	defer p.clientLock.Unlock()

	if client, exists := p.clients[id]; exists {
		remoteAddr := client.conn.RemoteAddr().String()
		ipAddress := strings.Split(remoteAddr, ":")[0]

		// ลบ client ออกจากรายการ
		config.Config.RemoveActiveClient(ipAddress)

		fmt.Printf("👋 Client %d disconnected: %s\n", id, remoteAddr)
		client.conn.Close()
		delete(p.clients, id)
	}
}

// ส่งข้อมูลไปยังทุก clients
func (p *ProxyServer) Broadcast(data []byte) {
	p.clientLock.Lock()
	defer p.clientLock.Unlock()

	for id, client := range p.clients {
		err := client.Send(data)
		if err != nil {
			fmt.Printf("⚠️ Cannot send data to Client %d: %v\n", id, err)
			// ถ้าส่งไม่ได้ให้ลบ client ออก
			go p.RemoveClient(id)
		}
	}
}

// จัดการการเชื่อมต่อจาก client
func HandleClientConnection(proxy *ProxyServer, conn net.Conn) {
	client := proxy.AddClient(conn)
	defer proxy.RemoveClient(client.id)

	// รอรับข้อมูลจาก client (ถ้าต้องการในอนาคต)
	buffer := make([]byte, 4096)
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			return
		}
	}
}

// ฟังก์ชันดึงข้อมูลจาก API และส่งไปยัง clients
func (p *ProxyServer) ProcessAndBroadcast() {
	var lastSpeakers []api.Speaker
	speakerStates := make(map[int]bool)

	for {
		select {
		case <-p.stopChan:
			return
		default:
			speakers, err := api.GetSpeakers()
			if err != nil {
				fmt.Println("⚠️ Cannot fetch speakers data:", err)
				config.Config.UpdateAPIStatus("Cannot connect to API: " + err.Error())

				// ส่ง XML ว่างเมื่อไม่มีข้อมูลจาก API
				emptyXML := xml.ToUTF16LEString(fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="80"><ActiveList><Participants></Participants></ActiveList></Discussion></DiscussionActivity>`,
					time.Now().Format("2006-01-02T15:04:05.0000000-07:00")))

				header := make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 3)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(emptyXML)))
				p.Broadcast(append(header, emptyXML...))
				time.Sleep(time.Second)
				continue
			}

			// อัปเดตสถานะ API เป็นเชื่อมต่อสำเร็จ
			config.Config.UpdateAPIStatus("Connected successfully")

			// ตรวจสอบการเปลี่ยนแปลงของแต่ละที่นั่ง
			currentSpeakerIDs := make(map[int]bool)
			for _, speaker := range speakers {
				currentSpeakerIDs[speaker.ID] = true

				// ตรวจสอบการเปลี่ยนแปลงสถานะไมค์
				lastState, exists := speakerStates[speaker.ID]
				microphoneActive := speaker.MicOn == 1
				if !exists || lastState != microphoneActive {
					// แสดงสถานะในคอนโซล
					micStatus := "Off 🔴"
					if microphoneActive {
						micStatus = "On 🟢"
					}
					fmt.Printf("🎙️ Mic %s: %s\n", speaker.SeatName, micStatus)

					// ส่ง SeatActivity เมื่อสถานะเปลี่ยน
					seatXML := xml.GenerateSeatXML(speaker, microphoneActive)
					header := make([]byte, 8)
					binary.LittleEndian.PutUint32(header[0:4], 5)
					binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
					p.Broadcast(append(header, seatXML...))
					speakerStates[speaker.ID] = microphoneActive
				}
			}

			// ตรวจสอบที่นั่งที่หายไป
			for id, state := range speakerStates {
				if !currentSpeakerIDs[id] && state {
					// สร้าง speaker ข้อมูลเดิมแต่ปิดไมค์
					for _, oldSpeaker := range lastSpeakers {
						if oldSpeaker.ID == id {
							// แสดงสถานะในคอนโซล
							fmt.Printf("🎙️ Mic %s: Canceled ⚫\n", oldSpeaker.SeatName)

							seatXML := xml.GenerateSeatXML(oldSpeaker, false)
							header := make([]byte, 8)
							binary.LittleEndian.PutUint32(header[0:4], 5)
							binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML)))
							p.Broadcast(append(header, seatXML...))
							speakerStates[id] = false

							// อัปเดตสถานะไมค์
							config.Config.UpdateActiveMic(oldSpeaker.SeatName, false)
							break
						}
					}
				}
			}

			// ส่ง DiscussionActivity เมื่อรายการที่นั่งเปลี่ยน
			if !reflect.DeepEqual(speakers, lastSpeakers) {
				discussionXML := xml.GenerateDiscussionXML(speakers)
				header := make([]byte, 8)
				binary.LittleEndian.PutUint32(header[0:4], 3)
				binary.LittleEndian.PutUint32(header[4:8], uint32(len(discussionXML)))
				p.Broadcast(append(header, discussionXML...))
				lastSpeakers = speakers
			}

			time.Sleep(time.Second)
		}
	}
}

// เริ่มทำงาน Proxy
func StartProxy() {
	// สร้าง proxy server
	proxy := NewProxyServer()
	proxy.isRunning = true

	// อัปเดตสถานะของ server
	config.Config.UpdateTCPServerStatus("Initializing...")

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
		config.Config.UpdateTCPServerStatus(errMsg)
		return
	}

	// ถ้าไม่สามารถหา port ที่ว่างได้
	if err != nil {
		errMsg := fmt.Sprintf("Cannot find available port after %d retries", maxRetries)
		fmt.Printf("❌ %s\n", errMsg)
		config.Config.UpdateTCPServerStatus(errMsg)
		return
	}

	// อัปเดต port ที่ใช้จริงใน config
	config.Config.TCPServerPort = currentPort

	// อัปเดตสถานะว่า server พร้อมใช้งาน
	config.Config.UpdateTCPServerStatus("Running")

	fmt.Printf("🚀 TCP Server running on port %s\n", currentPort)

	// เริ่มการประมวลผลและส่งข้อมูล
	go proxy.ProcessAndBroadcast()

	// ปิดการเชื่อมต่อและหยุด server เมื่อฟังก์ชันสิ้นสุด
	defer func() {
		close(proxy.stopChan)
		proxyListener.Close()
		proxy.isRunning = false
		config.Config.UpdateTCPServerStatus("Stopped")
	}()

	// รับการเชื่อมต่อจาก clients
	stopChan := config.GetStopChannel()

	for proxy.isRunning {
		// ตรวจสอบหาก server ถูกสั่งให้หยุด
		select {
		case <-stopChan:
			proxy.isRunning = false
			return
		default:
			// ตั้งค่า timeout เพื่อให้หลุดจากลูป accept เมื่อจำเป็น
			proxyListener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))

			clientConn, err := proxyListener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// timeout การรับเชื่อมต่อ ข้ามไปตรวจสอบ isRunning
					continue
				}

				fmt.Printf("⚠️ Cannot accept client connection: %v\n", err)
				continue
			}

			go HandleClientConnection(proxy, clientConn)
		}
	}
}

// Setup กำหนดฟังก์ชันเริ่มต้นให้กับ config
func Setup() {
	// กำหนดฟังก์ชันเริ่มต้น TCP Server
	config.StartServerFunc = StartProxy
}
