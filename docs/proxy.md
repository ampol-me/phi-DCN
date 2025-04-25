# คู่มือการใช้งาน proxy.go

## ภาพรวม
ไฟล์ `proxy.go` ทำหน้าที่เป็นตัวกลางในการรับข้อมูลจาก API แล้วส่งต่อไปยังไคลเอนต์ที่เชื่อมต่อผ่าน TCP ไฟล์นี้ประกอบด้วยฟังก์ชันและโครงสร้างข้อมูลสำหรับการจัดการการเชื่อมต่อและการส่งข้อมูล

## โครงสร้างข้อมูล

### Client
```go
type Client struct {
    conn        net.Conn
    id          int
    lastActive  time.Time
    sendQueue   chan []byte
    done        chan struct{}
}
```
- `conn` - การเชื่อมต่อ TCP กับไคลเอนต์
- `id` - รหัสประจำตัวของไคลเอนต์
- `lastActive` - เวลาล่าสุดที่ไคลเอนต์มีการทำงาน
- `sendQueue` - ช่องทางสำหรับส่งข้อมูลไปยังไคลเอนต์
- `done` - ช่องทางสำหรับสัญญาณการปิดการเชื่อมต่อ

### ProxyServer
```go
type ProxyServer struct {
    clients       map[int]*Client
    nextID        int
    clientLock    sync.RWMutex
    bufferPool    sync.Pool
    isRunning     bool
    stopChan      chan struct{}
    connCount     int
    connCountLock sync.Mutex
    metrics       *Metrics
}
```
- `clients` - แมปของไคลเอนต์ทั้งหมดที่เชื่อมต่ออยู่ (ID -> Client)
- `nextID` - รหัสประจำตัวสำหรับไคลเอนต์ถัดไป
- `clientLock` - mutex สำหรับการเข้าถึงแมปไคลเอนต์
- `bufferPool` - pool ของบัฟเฟอร์สำหรับการอ่านข้อมูล
- `isRunning` - สถานะการทำงานของเซิร์ฟเวอร์
- `stopChan` - ช่องทางสำหรับสัญญาณการหยุดเซิร์ฟเวอร์
- `connCount` - จำนวนการเชื่อมต่อปัจจุบัน
- `connCountLock` - mutex สำหรับการเข้าถึงจำนวนการเชื่อมต่อ
- `metrics` - ข้อมูลสถิติการทำงาน

### Metrics
```go
type Metrics struct {
    TotalConnections    int64
    CurrentConnections  int32
    TotalErrors         int64
    APIErrors           int64
    ClientErrors        int64
    LastError           string
    LastErrorTime       time.Time
    mu                  sync.RWMutex
}
```
- `TotalConnections` - จำนวนการเชื่อมต่อทั้งหมดตั้งแต่เริ่มโปรแกรม
- `CurrentConnections` - จำนวนการเชื่อมต่อปัจจุบัน
- `TotalErrors` - จำนวนข้อผิดพลาดทั้งหมด
- `APIErrors` - จำนวนข้อผิดพลาดจาก API
- `ClientErrors` - จำนวนข้อผิดพลาดจากไคลเอนต์
- `LastError` - ข้อผิดพลาดล่าสุด
- `LastErrorTime` - เวลาที่เกิดข้อผิดพลาดล่าสุด
- `mu` - mutex สำหรับการเข้าถึงข้อมูลสถิติ

## ค่าคงที่
```go
const (
    MaxConnections     = 5    // จำนวนการเชื่อมต่อสูงสุด
    ReadTimeout        = 30   // timeout สำหรับการอ่านข้อมูลจาก client (วินาที)
    WriteTimeout       = 5    // timeout สำหรับการเขียนข้อมูลไปยัง client (วินาที)
    RetryAttempts      = 3    // จำนวนครั้งในการ retry เมื่อเกิดข้อผิดพลาด
    RetryDelay         = 500  // ระยะเวลาในการ retry (มิลลิวินาที)
    APIPollingInterval = 750  // ระยะเวลาในการดึงข้อมูลจาก API (มิลลิวินาที)
)
```

## ฟังก์ชันหลัก

### Setup
```go
func Setup()
```
เตรียมพร้อมสำหรับการทำงานของ Proxy โดยโหลดการตั้งค่าจากไฟล์ config.ini และเริ่มทำงาน Proxy

### StartProxy
```go
func StartProxy()
```
เริ่มทำงาน Proxy โดยสร้าง ProxyServer และเริ่มรับการเชื่อมต่อจากไคลเอนต์

### StopProxy
```go
func StopProxy(proxy *ProxyServer)
```
หยุดการทำงานของ Proxy โดยปิดการเชื่อมต่อกับทุกไคลเอนต์

### ProcessAndBroadcast
```go
func (p *ProxyServer) ProcessAndBroadcast()
```
ดึงข้อมูลจาก API และส่งไปยังไคลเอนต์ทั้งหมดที่เชื่อมต่ออยู่ เป็นฟังก์ชันหลักที่ทำงานตลอดเวลาในการดึงข้อมูลและส่งข้อมูล

### HandleClientConnection
```go
func HandleClientConnection(proxy *ProxyServer, conn net.Conn)
```
จัดการการเชื่อมต่อจากไคลเอนต์ โดยสร้าง Client และรอรับข้อมูลจากไคลเอนต์

### AddClient
```go
func (p *ProxyServer) AddClient(conn net.Conn) (*Client, error)
```
เพิ่มไคลเอนต์ใหม่เข้าสู่ระบบ โดยตรวจสอบจำนวนการเชื่อมต่อก่อนและสร้าง Client ใหม่

### RemoveClient
```go
func (p *ProxyServer) RemoveClient(id int)
```
ลบไคลเอนต์ออกจากระบบเมื่อไคลเอนต์ยกเลิกการเชื่อมต่อหรือเกิดข้อผิดพลาด

### Broadcast
```go
func (p *ProxyServer) Broadcast(data []byte)
```
ส่งข้อมูลไปยังไคลเอนต์ทั้งหมดที่เชื่อมต่ออยู่

## ฟังก์ชันเสริม

### initLogging
```go
func initLogging()
```
เริ่มต้นระบบบันทึกข้อมูล โดยสร้างโฟลเดอร์ logs และเปิดไฟล์สำหรับบันทึกข้อมูล

### RecordError
```go
func (m *Metrics) RecordError(errorType string, err error)
```
บันทึกข้อผิดพลาด โดยเพิ่มจำนวนข้อผิดพลาดและบันทึกข้อผิดพลาดล่าสุด

### retry
```go
func retry(attempts int, sleep time.Duration, f func() error) error
```
ทำซ้ำเมื่อเกิดข้อผิดพลาด โดยลองทำฟังก์ชันซ้ำตามจำนวนครั้งที่กำหนด

### cleanInactiveClients
```go
func (p *ProxyServer) cleanInactiveClients()
```
ตรวจสอบและลบไคลเอนต์ที่ไม่ได้ใช้งานเป็นเวลานาน

### GetServerMetrics
```go
func (p *ProxyServer) GetServerMetrics() *Metrics
```
คืนค่าข้อมูลสถิติการทำงานของเซิร์ฟเวอร์

### ValidateInput
```go
func ValidateInput(data []byte) bool
```
ตรวจสอบความถูกต้องของข้อมูลที่รับเข้ามา

### SanitizeOutput
```go
func SanitizeOutput(data []byte) []byte
```
ทำความสะอาดข้อมูลก่อนส่งออก

## การใช้งานพื้นฐาน

### การเริ่มต้น Proxy
```go
proxy.Setup() // เริ่มต้น Proxy ในไฟล์ main.go
```

### การส่งข้อมูลไปยังไคลเอนต์
```go
data := []byte("Hello, World!")
proxy.Broadcast(data)
```

### การตรวจสอบสถิติการทำงาน
```go
metrics := proxy.GetServerMetrics()
fmt.Printf("จำนวนการเชื่อมต่อปัจจุบัน: %d\n", metrics.CurrentConnections)
```

## ตัวอย่างรหัสเพิ่มเติม

### การแตกเช่น XML ออกเป็นส่วนๆ
```go
// สร้าง XML สำหรับข้อมูลที่นั่ง
seatXML := xml.GenerateSeatXML(speaker, microphoneActive)

// สร้าง header สำหรับ XML
header := make([]byte, 8)
binary.LittleEndian.PutUint32(header[0:4], 5) // ประเภทข้อมูล
binary.LittleEndian.PutUint32(header[4:8], uint32(len(seatXML))) // ความยาวข้อมูล

// ส่งข้อมูลไปยังไคลเอนต์ทั้งหมด
proxy.Broadcast(append(header, seatXML...))
```

### การจัดการข้อผิดพลาด
```go
err := api.GetSpeakers()
if err != nil {
    proxy.metrics.RecordError("API", fmt.Errorf("Cannot fetch speakers data: %v", err))
    fmt.Println("⚠️ Cannot fetch speakers data:", err)
    config.Config.UpdateAPIStatus("Cannot connect to API: " + err.Error())
}
```

## คำแนะนำในการพัฒนาต่อ

1. การเพิ่มการรองรับการเชื่อมต่อแบบ WebSocket
   - สร้างไฟล์ใหม่ เช่น `websocket.go` สำหรับการจัดการการเชื่อมต่อแบบ WebSocket
   - เพิ่มโครงสร้าง `WebSocketClient` และฟังก์ชันที่เกี่ยวข้อง

2. การเพิ่มการรองรับการเข้ารหัสข้อมูล
   - เพิ่มฟังก์ชัน `EncryptData` และ `DecryptData` สำหรับการเข้ารหัสและถอดรหัสข้อมูล
   - เพิ่มการตั้งค่าสำหรับการเข้ารหัสในไฟล์ `config.ini`

3. การเพิ่มการจำกัดการใช้งานตามที่อยู่ IP
   - เพิ่มฟังก์ชัน `IsIPAllowed` สำหรับการตรวจสอบว่าที่อยู่ IP ได้รับอนุญาตหรือไม่
   - เพิ่มการตั้งค่าสำหรับการจำกัดการใช้งานในไฟล์ `config.ini` 