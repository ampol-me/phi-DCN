# phi-DCN (Phi Discussion Control Network)

ระบบสำหรับเชื่อมต่อและแสดงสถานะไมโครโฟนในการประชุม

## ความต้องการของระบบ

- Go 1.22 หรือสูงกว่า
- เครื่อง Mac หรือ Windows

## การติดตั้ง

### สำหรับนักพัฒนา

1. ติดตั้ง [Go](https://golang.org/doc/install)
2. โคลนโปรเจคนี้
3. ใช้คำสั่ง `go build` เพื่อคอมไพล์

### สำหรับผู้ใช้ทั่วไป

ดาวน์โหลดไฟล์ที่คอมไพล์แล้วจากโฟลเดอร์ `build` หรือจาก releases

## การสร้างไฟล์สำหรับแจกจ่าย (Build)

### สำหรับ Mac

วิธีที่ 1: ใช้ไฟล์ command script
1. เปิด Finder และไปที่โฟลเดอร์ของโปรเจค
2. ดับเบิลคลิกที่ไฟล์ `build/build-mac.command` (คลิกขวาและเลือก "เปิด" หากมีคำเตือนเรื่องความปลอดภัย)
3. ระบบจะทำการ build โปรแกรมโดยอัตโนมัติ

วิธีที่ 2: ใช้ Terminal
1. เปิด Terminal และนำทางไปยังโฟลเดอร์ของโปรเจค
2. รันคำสั่ง `./build/build.sh`

### สำหรับ Windows

วิธีที่ 1: ใช้ไฟล์ batch script
1. เปิด File Explorer และไปที่โฟลเดอร์ของโปรเจค
2. ดับเบิลคลิกที่ไฟล์ `build/build-windows.bat`
3. ระบบจะทำการ build โปรแกรมโดยอัตโนมัติ

วิธีที่ 2: ใช้ Command Prompt
1. เปิด Command Prompt และนำทางไปยังโฟลเดอร์ของโปรเจค
2. รันคำสั่ง `build\build-windows.bat`

### การสร้างไฟล์ ZIP สำหรับแจกจ่าย (Release)

หลังจาก build โปรแกรมเสร็จแล้ว สามารถสร้างไฟล์ ZIP สำหรับแจกจ่ายได้ดังนี้:

1. เปิด Terminal และนำทางไปยังโฟลเดอร์ของโปรเจค
2. รันคำสั่ง `./build/make-release.sh`
3. ไฟล์ ZIP จะถูกสร้างขึ้นในโฟลเดอร์ `build/release` ดังนี้:
   - `phi-dcn-1.0.0-mac-arm64.zip` - สำหรับ Mac ที่ใช้ Apple Silicon (M1/M2)
   - `phi-dcn-1.0.0-mac-amd64.zip` - สำหรับ Mac ที่ใช้ Intel
   - `phi-dcn-1.0.0-windows.zip` - สำหรับ Windows

## การใช้งาน

### Server

เริ่มการทำงานของ Server:
```
./phi-dcn-server
```

### Client

เริ่มการทำงานของ Client:
```
./phi-dcn-client
```

เริ่ม Client ด้วยพอร์ต API ที่กำหนดเอง:
```
./phi-dcn-client 3003
```

## การตั้งค่า

### config.ini
```ini
APIHost=localhost
APIPort=3000
APIPath=/api/speakers
TCPServerPort=20000
APIKey=your-api-key
```

## การทำงานของระบบ

### XML Server (Phi DCN Bridge)
XML Server ทำหน้าที่เป็นตัวกลางระหว่างระบบ DCN และระบบอื่นๆ โดยแปลงข้อมูลจาก API เป็น XML และส่งไปยังไคลเอนต์ผ่าน TCP connection

#### คุณสมบัติหลัก
1. **การเชื่อมต่อ TCP**
   - รองรับการเชื่อมต่อพร้อมกันสูงสุด 5 ไคลเอนต์
   - ใช้ keep-alive เพื่อรักษาการเชื่อมต่อ
   - มีระบบจัดการ timeout และการเชื่อมต่อที่ขาดหาย

2. **การแปลงข้อมูล**
   - แปลงข้อมูลจาก API เป็น XML
   - ส่งข้อมูลแบบ real-time เมื่อมีการเปลี่ยนแปลง
   - รองรับการส่งข้อมูลแบบ SeatActivity และ DiscussionActivity

3. **การจัดการสถานะ**
   - ติดตามสถานะไมค์ที่เปิด/ปิด
   - อัปเดตสถานะใน config
   - ส่ง XML เฉพาะเมื่อมีการเปลี่ยนแปลง

4. **ระบบ Logging**
   - บันทึกข้อมูลการทำงาน
   - บันทึกข้อผิดพลาด
   - แยกไฟล์ log ตามวัน

5. **การจัดการ Error**
   - มีระบบ retry สำหรับการเชื่อมต่อ API
   - มีการจัดการ error ที่เกิดขึ้นระหว่างการทำงาน
   - มีการบันทึกข้อผิดพลาดลง log

### การจัดการสถานะไมค์

ระบบจะจัดการสถานะไมค์ตามลำดับดังนี้:

1. **เมื่อมีไมค์เปิดใหม่:**
   - ส่ง SeatActivity ON สำหรับไมค์นั้น
   - เพิ่มไมค์เข้าไปใน currentSpeakers
   - ส่ง DiscussionActivity แสดงไมค์ทั้งหมดจนถึงลำดับนั้น
   - รอ 500ms ก่อนส่งไมค์ตัวต่อไป

2. **เมื่อมีไมค์ปิด:**
   - ส่ง SeatActivity OFF สำหรับไมค์นั้น
   - ลบไมค์ออกจาก currentSpeakers
   - ส่ง DiscussionActivity แสดงไมค์ที่เปิดอยู่ทั้งหมด

3. **เมื่อ API ส่งค่ากลับมาเป็น []:**
   - ส่ง SeatActivity OFF สำหรับไมค์ทั้งหมด
   - ล้าง currentSpeakers
   - ส่ง DiscussionActivity แสดงไมค์ที่เปิดอยู่ (ไม่มี)

### ตัวอย่างการทำงาน

1. **Event 1: API ส่ง mic on 4 id (MIC 1,2,3,4)**
   - ครั้งที่ 1:
     - ส่ง SeatActivity ON สำหรับ MIC 1
     - ส่ง DiscussionActivity แสดง MIC 1
   - ครั้งที่ 2:
     - ส่ง SeatActivity ON สำหรับ MIC 2
     - ส่ง DiscussionActivity แสดง MIC 1,2
   - ครั้งที่ 3:
     - ส่ง SeatActivity ON สำหรับ MIC 3
     - ส่ง DiscussionActivity แสดง MIC 1,2,3
   - ครั้งที่ 4:
     - ส่ง SeatActivity ON สำหรับ MIC 4
     - ส่ง DiscussionActivity แสดง MIC 1,2,3,4

2. **Event 2: API ส่ง mic on 2 id (MIC 1,3)**
   - ส่ง SeatActivity OFF สำหรับ MIC 2
   - ส่ง DiscussionActivity แสดง MIC 1,3
   - ส่ง SeatActivity OFF สำหรับ MIC 4
   - ส่ง DiscussionActivity แสดง MIC 1,3

3. **Event 3: API ส่งค่ากลับมาเป็น []**
   - ส่ง SeatActivity OFF สำหรับไมค์ทั้งหมด
   - ส่ง DiscussionActivity แสดงไมค์ที่เปิดอยู่ (ไม่มี)

## API Endpoints

- `GET /api/status`: ดูสถานะการเชื่อมต่อ
- `GET /api/clients`: ดูรายชื่อไคลเอนต์ที่เชื่อมต่ออยู่
- `GET /api/mics`: ดูรายการไมโครโฟนที่ใช้งานอยู่
- `GET /api/config`: ดูการตั้งค่าปัจจุบัน
- `POST /api/config`: อัปเดตการตั้งค่า
- `GET /api/test`: ทดสอบการเชื่อมต่อกับ API
- `GET /api/start`: เริ่ม TCP Server
- `GET /api/stop`: หยุด TCP Server


สำหรับการ build Windows executable บน macOS โดยใช้ icon นั้น สามารถทำได้ครับ โดยมีขั้นตอนดังนี้:
ต้องมีไฟล์ .rc (Resource Script) ที่อ้างอิงถึง icon
ใช้ windres (Windows Resource Compiler) เพื่อแปลง .rc เป็น .syso
เมื่อ build Go program จะใช้ .syso file โดยอัตโนมัติ
ผมจะสร้างไฟล์ icon.rc ให้ครับ:
```
1 ICON "phi-xml-icon.ico" 
windres -o icon.syso icon.rc
```

ติดตั้ง brew install mingw-w64

x86_64-w64-mingw32-windres icon.rc -O coff -o icon.syso

สำหรับการ build Windows executable บน macOS โดยใช้ icon นั้น สามารถทำได้ครับ โดยมีขั้นตอนดังนี้:
ต้องมีไฟล์ .rc (Resource Script) ที่อ้างอิงถึง icon
ใช้ windres (Windows Resource Compiler) เพื่อแปลง .rc เป็น .syso
เมื่อ build Go program จะใช้ .syso file โดยอัตโนมัติ
ผมจะสร้างไฟล์ icon.rc ให้ครับ:

cd client && GOOS=windows GOARCH=amd64 go build -o ../phi-dcn-bridge.exe