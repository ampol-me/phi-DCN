package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"phi-DCN/client/config"
)

var (
	mockMicState = true
	lastToggle   = time.Now()
)

// โครงสร้างข้อมูลจาก API
type Speaker struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	SeatName      string `json:"seatName"`
	Prio          bool   `json:"prio"`
	PrioOn        bool   `json:"prioOn"`
	ParticipantID int    `json:"participantId"`
	MicOn         bool   `json:"micOn"`
}

// ฟังก์ชันจำลองข้อมูล API
func GetMockSpeakers() ([]Speaker, error) {
	// สลับสถานะไมค์ทุก 5 วินาที
	if time.Since(lastToggle) >= 5*time.Second {
		mockMicState = !mockMicState
		lastToggle = time.Now()
		fmt.Printf("🔄 Toggle mic state to: %v\n", mockMicState)
	}

	return []Speaker{
		{
			ID:            3539,
			Name:          "A05",
			SeatName:      "A05",
			Prio:          false,
			PrioOn:        false,
			ParticipantID: 0,
			MicOn:         mockMicState,
		},
	}, nil
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

	client := &http.Client{
		Timeout: 5 * time.Second, // เพิ่ม timeout
	}
	resp, err := client.Do(req)
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

// ฟังก์ชันดึงข้อมูลจาก API
func GetSpeakers() ([]Speaker, error) {
	// สร้าง request ใหม่
	req, err := http.NewRequest("GET", config.Config.GetAPIURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %v", err)
	}

	// เพิ่ม Header สำหรับการตรวจสอบสิทธิ์
	req.Header.Set("Bosch-Sid", config.Config.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
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
		config.Config.UpdateActiveMic(speaker.Name, speaker.MicOn)
	}

	return speakers, nil
}
