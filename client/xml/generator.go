package xml

import (
	"fmt"
	"time"
	"unicode/utf16"

	"phi-DCN/client/api"
)

// เพิ่มฟังก์ชันสำหรับแปลง string เป็น UTF-16LE with BOM
func ToUTF16LEString(s string) []byte {
	// แปลงเป็น UTF-16LE
	u16 := utf16.Encode([]rune(s))

	// สร้าง buffer สำหรับเก็บ BOM และข้อมูล
	bytes := make([]byte, 2+len(u16)*2)

	// ใส่ BOM (0xFF 0xFE สำหรับ UTF-16LE)
	bytes[0] = 0xFF
	bytes[1] = 0xFE

	// แปลง []uint16 เป็น []byte
	for i, v := range u16 {
		bytes[2+i*2] = byte(v)
		bytes[2+i*2+1] = byte(v >> 8)
	}

	return bytes
}

// แก้ไขฟังก์ชัน generateDiscussionXML
func GenerateDiscussionXML(speakers []api.Speaker) []byte {
	// ตรวจสอบว่ามีไมค์ที่เปิดอยู่หรือไม่
	hasMicOn := false
	participantsXML := ""

	for _, speaker := range speakers {
		if speaker.MicOn {
			hasMicOn = true
			participantXML := fmt.Sprintf(`<ParticipantContainer Id="%d"><Seat Id="%d"><SeatData Name="%s" MicrophoneActive="true" SeatType="Delegate" IsSpecialStation="false" /><IsReposnding>false</IsReposnding></Seat></ParticipantContainer>`,
				speaker.ParticipantID,
				speaker.ID,
				speaker.SeatName,
			)
			participantsXML += participantXML
		}
	}

	var xmlStr string
	if !hasMicOn {
		xmlStr = fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="71"><ActiveList></ActiveList></Discussion></DiscussionActivity>`,
			time.Now().Format("2006-01-02T15:04:05.0000000-07:00"),
		)
	} else {
		xmlStr = fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><DiscussionActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Discussion" Type="ActiveListUpdated"><Discussion Id="71"><ActiveList><Participants>%s</Participants></ActiveList></Discussion></DiscussionActivity>`,
			time.Now().Format("2006-01-02T15:04:05.0000000-07:00"),
			participantsXML,
		)
	}

	return ToUTF16LEString(xmlStr)
}

// แก้ไขฟังก์ชัน generateSeatXML
func GenerateSeatXML(speaker api.Speaker, micState bool) []byte {
	xmlStr := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><SeatActivity xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" Version="1" TimeStamp="%s" Topic="Seat" Type="SeatUpdated"><Seat Id="%d"><SeatData Name="%s" MicrophoneActive="%v" SeatType="Delegate" IsSpecialStation="false" /><Participant Id="%d"><ParticipantData Present="false" VotingWeight="1" VotingAuthorisation="true" MicrophoneAuthorisation="true" FirstName="" MiddleName="" LastName="%s" Title="" Country="" RemainingSpeechTime="-1" SpeechTimerOnHold="false" /></Participant><IsReposnding>false</IsReposnding></Seat></SeatActivity>`,
		time.Now().Format("2006-01-02T15:04:05.0000000-07:00"),
		speaker.ID,
		speaker.SeatName,
		micState,
		speaker.ParticipantID,
		speaker.SeatName,
	)

	return ToUTF16LEString(xmlStr)
}
