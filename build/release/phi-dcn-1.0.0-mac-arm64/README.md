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

## API Endpoints

- `GET /api/status`: ดูสถานะการเชื่อมต่อ
- `GET /api/clients`: ดูรายชื่อไคลเอนต์ที่เชื่อมต่ออยู่
- `GET /api/mics`: ดูรายการไมโครโฟนที่ใช้งานอยู่
- `GET /api/config`: ดูการตั้งค่าปัจจุบัน
- `POST /api/config`: อัปเดตการตั้งค่า
- `GET /api/test`: ทดสอบการเชื่อมต่อกับ API
- `GET /api/start`: เริ่ม TCP Server
- `GET /api/stop`: หยุด TCP Server 