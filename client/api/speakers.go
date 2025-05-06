package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"phi-DCN/client/config"
)

// โครงสร้างข้อมูลจาก API
type Speaker struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	SeatName      string `json:"seatName"`
	Prio          int    `json:"-"`
	PrioOn        int    `json:"-"`
	ParticipantID int    `json:"participantId"`
	MicOn         int    `json:"-"`
}

// UnmarshalJSON จัดการกับการแปลงค่า JSON ให้เข้ากับโครงสร้าง Speaker
func (s *Speaker) UnmarshalJSON(data []byte) error {
	// สร้างโครงสร้างชั่วคราวที่มีฟิลด์เป็น interface{} สำหรับฟิลด์ที่อาจเป็นได้ทั้ง boolean และ int
	type SpeakerTemp struct {
		ID            int         `json:"id"`
		Name          string      `json:"name"`
		SeatName      string      `json:"seatName"`
		Prio          interface{} `json:"prio"`
		PrioOn        interface{} `json:"prioOn"`
		ParticipantID int         `json:"participantId"`
		MicOn         interface{} `json:"micOn"`
	}

	var temp SpeakerTemp
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// คัดลอกค่าที่ไม่ต้องแปลง
	s.ID = temp.ID
	s.Name = temp.Name
	s.SeatName = temp.SeatName
	s.ParticipantID = temp.ParticipantID

	// แปลงค่า Prio
	switch v := temp.Prio.(type) {
	case bool:
		if v {
			s.Prio = 1
		} else {
			s.Prio = 0
		}
	case float64: // JSON numbers แปลงเป็น float64
		s.Prio = int(v)
	}

	// แปลงค่า PrioOn
	switch v := temp.PrioOn.(type) {
	case bool:
		if v {
			s.PrioOn = 1
		} else {
			s.PrioOn = 0
		}
	case float64:
		s.PrioOn = int(v)
	}

	// แปลงค่า MicOn
	switch v := temp.MicOn.(type) {
	case bool:
		if v {
			s.MicOn = 1
		} else {
			s.MicOn = 0
		}
	case float64:
		s.MicOn = int(v)
	}

	return nil
}

// สร้าง HTTP client แบบ global เพื่อ reuse connection
var httpClient = &http.Client{
	Timeout: 2 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second,
			KeepAlive: 15 * time.Second,
			// เพิ่มการตั้งค่าสำหรับ Windows
			FallbackDelay: 100 * time.Millisecond,
		}).DialContext,
		TLSHandshakeTimeout:   2 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// เพิ่มการตั้งค่าสำหรับ Windows
		MaxConnsPerHost:    100,
		DisableCompression: false,
	},
}

// ฟังก์ชันดึงข้อมูลจาก API
func GetSpeakers() ([]Speaker, error) {
	// สร้าง request ใหม่
	req, err := http.NewRequest("GET", config.Config.GetAPIURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %v", err)
	}

	// เพิ่ม Header สำหรับการตรวจสอบสิทธิ์
	req.Header.Set("Bosch-Sid", config.Config.GetAPIKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Accept-Encoding", "gzip")
	// เพิ่ม header สำหรับ Windows
	req.Header.Set("User-Agent", "DCN-Client/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read data from API: %v", err)
	}

	var speakers []Speaker
	if err := json.Unmarshal(body, &speakers); err != nil {
		return nil, fmt.Errorf("Failed to parse JSON data: %v", err)
	}

	// อัปเดตสถานะไมค์ที่กำลังใช้งาน
	for _, speaker := range speakers {
		config.Config.UpdateActiveMic(speaker.SeatName, speaker.MicOn == 1)
	}

	return speakers, nil
}

// ฟังก์ชันทดสอบการเชื่อมต่อกับ API
func TestConnection() error {
	// สร้าง request ใหม่
	req, err := http.NewRequest("GET", config.Config.GetAPIURL(), nil)
	if err != nil {
		return fmt.Errorf("Failed to create request: %v", err)
	}

	// เพิ่ม Header สำหรับการตรวจสอบสิทธิ์
	req.Header.Set("Bosch-Sid", config.Config.APIKey)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Accept-Encoding", "gzip")
	// เพิ่ม header สำหรับ Windows
	req.Header.Set("User-Agent", "DCN-Client/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to connect to API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned error status: %s", resp.Status)
	}

	// อ่านข้อมูลจาก API เพื่อตรวจสอบว่าเป็น JSON ที่ถูกต้องหรือไม่
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read data from API: %v", err)
	}

	var speakers []Speaker
	if err := json.Unmarshal(body, &speakers); err != nil {
		return fmt.Errorf("Failed to parse JSON data: %v", err)
	}

	return nil
}
