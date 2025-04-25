# คู่มือการทดสอบและแก้ไขปัญหา

## การทดสอบระบบ

### การทดสอบการเชื่อมต่อ API

1. **ทดสอบการเชื่อมต่อโดยตรง**
   ```bash
   curl -v http://<API_HOST>:<API_PORT>/api/speakers
   ```
   ผลลัพธ์ที่ได้ควรเป็น JSON ที่มีข้อมูลของผู้พูด

2. **ทดสอบผ่าน REST API ของระบบ**
   ```bash
   curl http://localhost:3002/api/test-connection
   ```
   ผลลัพธ์ที่ได้ควรเป็น `{"status":"success"}` หากการเชื่อมต่อสำเร็จ

3. **ตรวจสอบสถานะการเชื่อมต่อ API**
   ```bash
   curl http://localhost:3002/api/status
   ```
   ผลลัพธ์ที่ได้ควรมีฟิลด์ `apiStatus` ที่แสดงสถานะการเชื่อมต่อ

### การทดสอบการเชื่อมต่อ TCP

1. **ทดสอบการเชื่อมต่อ TCP โดยใช้ telnet**
   ```bash
   telnet localhost <TCP_SERVER_PORT>
   ```
   หากเชื่อมต่อสำเร็จ คุณจะเห็นข้อความต้อนรับ

2. **ทดสอบการเชื่อมต่อโดยใช้ netcat**
   ```bash
   nc -v localhost <TCP_SERVER_PORT>
   ```
   หากเชื่อมต่อสำเร็จ คุณจะเห็นข้อความแสดงการเชื่อมต่อ

3. **ตรวจสอบรายการไคลเอนต์ที่เชื่อมต่ออยู่**
   ```bash
   curl http://localhost:3002/api/clients
   ```
   ผลลัพธ์ที่ได้ควรแสดงรายการของไคลเอนต์ที่เชื่อมต่ออยู่

### การทดสอบการส่งข้อมูล

1. **ทดสอบการส่งข้อมูลโดยใช้ไคลเอนต์จำลอง**
   ```bash
   # สร้างไฟล์ test-client.go
   go run test-client.go
   ```

2. **ตรวจสอบข้อมูลที่ได้รับจากเซิร์ฟเวอร์**
   ข้อมูลที่ได้รับควรเป็น XML ที่มีข้อมูลของที่นั่งและสถานะไมค์

3. **ตรวจสอบ log การส่งข้อมูล**
   ```bash
   tail -f logs/info_YYYY-MM-DD.log
   ```

## การทดสอบหน่วย (Unit Testing)

### การเขียน Unit Test สำหรับส่วนต่างๆ

1. **การทดสอบส่วน config**
   ```go
   // ไฟล์ config/config_test.go
   package config

   import (
       "testing"
       "os"
   )

   func TestLoadConfig(t *testing.T) {
       // สร้างไฟล์ config.ini ทดสอบ
       os.MkdirAll("testdata", 0755)
       os.WriteFile("testdata/config.ini", []byte("APIHost=testhost\nAPIPort=1234"), 0644)
       
       // โหลดการตั้งค่า
       err := LoadConfig()
       if err != nil {
           t.Errorf("LoadConfig failed: %v", err)
       }
       
       // ตรวจสอบค่าที่โหลด
       if Config.APIHost != "testhost" {
           t.Errorf("Expected APIHost to be 'testhost', got '%s'", Config.APIHost)
       }
   }
   ```

2. **การทดสอบส่วน proxy**
   ```go
   // ไฟล์ proxy/proxy_test.go
   package proxy

   import (
       "testing"
       "net"
   )

   func TestAddClient(t *testing.T) {
       server := NewProxyServer()
       
       // สร้างการเชื่อมต่อจำลอง
       client, server := net.Pipe()
       
       // เพิ่มไคลเอนต์
       c, err := server.AddClient(client)
       if err != nil {
           t.Errorf("AddClient failed: %v", err)
       }
       
       // ตรวจสอบว่าเพิ่มไคลเอนต์สำเร็จ
       if server.clients[c.id] == nil {
           t.Errorf("Client not added to server")
       }
   }
   ```

### การรัน Unit Test

1. **รัน Unit Test ทั้งหมด**
   ```bash
   go test ./...
   ```

2. **รัน Unit Test เฉพาะส่วน**
   ```bash
   go test ./client/config
   go test ./client/proxy
   ```

3. **รัน Unit Test พร้อมข้อมูลเพิ่มเติม**
   ```bash
   go test -v ./...
   ```

## การแก้ไขปัญหาทั่วไป

### ปัญหาการเชื่อมต่อ API

1. **API ไม่ตอบสนอง**
   - ตรวจสอบว่า API server กำลังทำงานอยู่หรือไม่
   - ตรวจสอบการตั้งค่า API host และ port ในไฟล์ config.ini
   - ตรวจสอบว่ามีไฟร์วอลล์ที่บล็อกการเชื่อมต่อหรือไม่

   **การแก้ไข**:
   - ตรวจสอบการเชื่อมต่อโดยตรงด้วย curl หรือเว็บเบราว์เซอร์
   - ปรับการตั้งค่า API host และ port ให้ถูกต้อง
   - ตรวจสอบการตั้งค่าไฟร์วอลล์

2. **API คืนค่า error**
   - ตรวจสอบ log ของระบบเพื่อดูข้อความ error
   - ตรวจสอบว่า API key ถูกต้องหรือไม่ (ถ้าใช้)

   **การแก้ไข**:
   - ตรวจสอบ format ของ API path
   - ตรวจสอบความถูกต้องของ API key
   - ตรวจสอบ log ของ API server

### ปัญหาการเชื่อมต่อ TCP

1. **ไม่สามารถเชื่อมต่อกับ TCP server**
   - ตรวจสอบว่า proxy server กำลังทำงานอยู่หรือไม่
   - ตรวจสอบพอร์ตที่เซิร์ฟเวอร์ใช้งาน
   - ตรวจสอบว่ามีแอปพลิเคชันอื่นใช้พอร์ตเดียวกันหรือไม่

   **การแก้ไข**:
   - รีสตาร์ทเซิร์ฟเวอร์
   - เปลี่ยนพอร์ตที่ใช้งาน
   - ตรวจสอบสถานะของเซิร์ฟเวอร์ด้วย `curl http://localhost:3002/api/server`

2. **เชื่อมต่อหลุดบ่อย**
   - ตรวจสอบการตั้งค่า timeout
   - ตรวจสอบปัญหาเครือข่าย

   **การแก้ไข**:
   - เพิ่มค่า ReadTimeout และ WriteTimeout ในไฟล์ proxy.go
   - ตรวจสอบการเชื่อมต่อเครือข่าย
   - ตรวจสอบการทำงานของไคลเอนต์

### ปัญหาการส่งข้อมูล

1. **ข้อมูลไม่ถูกส่งไปยังไคลเอนต์**
   - ตรวจสอบว่าไคลเอนต์ยังคงเชื่อมต่ออยู่หรือไม่
   - ตรวจสอบว่าข้อมูลจาก API ถูกต้องหรือไม่

   **การแก้ไข**:
   - ตรวจสอบสถานะไคลเอนต์
   - ตรวจสอบข้อมูลที่ได้รับจาก API
   - ตรวจสอบ log การส่งข้อมูล

2. **ข้อมูลที่ส่งไม่ถูกต้อง**
   - ตรวจสอบการแปลงข้อมูลเป็น XML
   - ตรวจสอบว่า XML ที่สร้างถูกต้องตามที่ไคลเอนต์คาดหวังหรือไม่

   **การแก้ไข**:
   - ตรวจสอบการสร้าง XML ในไฟล์ xml.go
   - ตรวจสอบ format ของข้อมูลที่ส่ง
   - ตรวจสอบการเข้ารหัส UTF-16LE

### ปัญหาการใช้หน่วยความจำสูงหรือ CPU สูง

1. **การใช้หน่วยความจำเพิ่มขึ้นเรื่อยๆ**
   - อาจเกิดจาก memory leak
   - อาจเกิดจากการสร้างข้อมูลมากเกินไปโดยไม่มีการเก็บกวาด

   **การแก้ไข**:
   - ตรวจสอบการใช้ goroutine
   - ตรวจสอบการใช้ buffer ในฟังก์ชัน ProcessAndBroadcast
   - ใช้ pprof ในการวิเคราะห์การใช้หน่วยความจำ

2. **การใช้ CPU สูงมาก**
   - อาจเกิดจาก tight loop
   - อาจเกิดจากการประมวลผลข้อมูลที่มากเกินไป

   **การแก้ไข**:
   - ตรวจสอบและปรับปรุงการใช้ for loop
   - เพิ่มการพักการทำงานด้วย time.Sleep
   - ใช้ pprof ในการวิเคราะห์การใช้ CPU

## การวิเคราะห์ Log

1. **การตรวจสอบ log ข้อผิดพลาด**
   ```bash
   grep ERROR logs/error_YYYY-MM-DD.log
   ```

2. **การตรวจสอบ log การเชื่อมต่อ**
   ```bash
   grep "Client.*connected" logs/info_YYYY-MM-DD.log
   ```

3. **การตรวจสอบ log การส่งข้อมูล**
   ```bash
   grep "Mic" logs/info_YYYY-MM-DD.log
   ```

## การใช้ Debugging Tools

1. **ใช้ pprof สำหรับการวิเคราะห์ประสิทธิภาพ**
   ```go
   import _ "net/http/pprof"

   func main() {
       go func() {
           http.ListenAndServe("localhost:6060", nil)
       }()
       // ...
   }
   ```
   แล้วเข้าถึงผ่าน `http://localhost:6060/debug/pprof/`

2. **ใช้ tcpdump สำหรับการวิเคราะห์การเชื่อมต่อ**
   ```bash
   sudo tcpdump -i lo port <TCP_SERVER_PORT> -vv
   ```

3. **ใช้ Wireshark สำหรับการวิเคราะห์การเชื่อมต่อที่ละเอียดมากขึ้น**
   - เปิด Wireshark และจับการเชื่อมต่อที่พอร์ตที่ต้องการ
   - วิเคราะห์ข้อมูลที่รับส่ง 