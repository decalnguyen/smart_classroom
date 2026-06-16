package handlers

import (
	"log"
	"os"
	"sync"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
)

// Timetable-based device automation — implements the báo cáo tóm tắt (III):
// "tự động điều khiển đèn/quạt theo ngưỡng môi trường VÀ thời khóa biểu", and the
// energy-saving motivation in Ch 1.1 ("lãng phí điện năng khi phòng trống").
//
// Behaviour: when a classroom has NO ongoing class (per the timetable, respecting
// holidays/makeup sessions via findOngoingClass), the server turns its light + fan
// OFF to save energy. It acts only on the occupied→empty transition (plus once at
// startup for rooms that are already empty), so it never spams commands and never
// fights the on-device threshold logic or a teacher's manual control while a class
// is in session. Threshold-based ON (đèn theo ánh sáng, quạt theo nhiệt độ) stays
// on the ESP32; the timetable only contributes the auto-OFF when the room is empty.
//
// Opt out with SCHEDULE_AUTOCONTROL=off; interval via SCHEDULE_CONTROL_SECONDS (default 60).

var (
	schedMu      sync.Mutex
	roomOccupied = map[uint]bool{}
	schedFirst   = true
)

// ScheduleAutoControl starts the timetable-driven energy-saving controller.
func ScheduleAutoControl() {
	if os.Getenv("SCHEDULE_AUTOCONTROL") == "off" {
		log.Println("Schedule auto-control disabled (SCHEDULE_AUTOCONTROL=off)")
		return
	}
	every := time.Duration(envInt("SCHEDULE_CONTROL_SECONDS", 60)) * time.Second
	go func() {
		for {
			runScheduleAutoControl()
			time.Sleep(every)
		}
	}()
}

func runScheduleAutoControl() {
	var rooms []models.Classroom
	if err := db.DB.Find(&rooms).Error; err != nil {
		return
	}
	schedMu.Lock()
	defer schedMu.Unlock()
	first := schedFirst
	schedFirst = false

	for _, r := range rooms {
		_, occupied := findOngoingClass(r.ClassroomID)
		prev, seen := roomOccupied[r.ClassroomID]
		roomOccupied[r.ClassroomID] = occupied

		// Act when a class just ended (occupied→empty) or, on the first sweep, for
		// rooms that are already empty — then ensure light + fan are OFF.
		if !occupied && ((seen && prev) || (first && !seen)) {
			PublishDeviceCommand(r.ClassroomName, "light", "off", 0, "tự động tắt — không có tiết (TKB)")
			PublishDeviceCommand(r.ClassroomName, "fan", "off", 0, "tự động tắt — không có tiết (TKB)")
			log.Printf("[schedule-control] %s: no ongoing class → auto-off light+fan", r.ClassroomName)
		}
	}
}
