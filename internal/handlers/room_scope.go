package handlers

import (
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
)

// roomWindow is one class period the actor participates in, in a given room.
type roomWindow struct {
	Day      string // "Monday".."Sunday"
	StartMin int    // minutes from midnight
	EndMin   int
}

// actorRoomScope returns the rooms (by name) the caller may view sensor/overview
// data for, plus each room's class time-windows. Per the requirement "giáo viên
// chỉ xem phòng có tiết dạy của mình theo khung giờ; sinh viên tương tự":
//
//	admin   -> isAll=true  (every room, no time limit)
//	teacher -> rooms they TEACH      (classes.teacher_id) + those periods' windows
//	student -> rooms of ENROLLED classes (class_students) + those periods' windows
//
// Windows come from the same classes (day_of_week + start_min/end_min) and are
// used to clamp the time-range a teacher/student may view.
func actorRoomScope(c *gin.Context) (rooms map[string]bool, windows map[string][]roomWindow, isAll bool) {
	role := c.GetString("role")
	if role == "admin" || role == "" {
		return nil, nil, true
	}
	account := c.GetString("account_id")
	rooms = map[string]bool{}
	windows = map[string][]roomWindow{}

	type cw struct {
		ClassroomName string
		DayOfWeek     string
		StartMin      int
		EndMin        int
	}
	var rowsCW []cw
	q := db.DB.Table("classes").
		Select("classrooms.classroom_name, classes.day_of_week, classes.start_min, classes.end_min").
		Joins("JOIN classrooms ON classrooms.classroom_id = classes.classroom_id")

	switch role {
	case "teacher":
		var t models.Teacher
		if err := db.DB.Where("account_id = ?", account).First(&t).Error; err != nil {
			return rooms, windows, false
		}
		q = q.Where("classes.teacher_id = ?", t.TeacherID)
	default: // student (and any other non-admin role): scope to enrolled classes
		var s models.Student
		if err := db.DB.Where("account_id = ?", account).First(&s).Error; err != nil {
			return rooms, windows, false
		}
		q = q.Joins("JOIN class_students cs ON cs.class_id = classes.class_id").
			Where("cs.student_id = ?", s.StudentID)
	}
	q.Scan(&rowsCW)
	for _, r := range rowsCW {
		rooms[r.ClassroomName] = true
		windows[r.ClassroomName] = append(windows[r.ClassroomName], roomWindow{Day: r.DayOfWeek, StartMin: r.StartMin, EndMin: r.EndMin})
	}
	return rooms, windows, false
}

// actorClassroomIDs resolves the caller's scoped rooms to classroom IDs.
// admin => isAll=true (nil ids). teacher/student => IDs of their taught/enrolled rooms.
func actorClassroomIDs(c *gin.Context) (ids []uint, isAll bool) {
	rooms, _, all := actorRoomScope(c)
	if all {
		return nil, true
	}
	if len(rooms) == 0 {
		return []uint{}, false
	}
	db.DB.Model(&models.Classroom{}).Where("classroom_name IN ?", roomNames(rooms)).Pluck("classroom_id", &ids)
	return ids, false
}

// roomNames returns the keys of a room set (for an IN query).
func roomNames(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// inAnyWindow reports whether instant t falls within any of the given class
// windows (matched by weekday + minute-of-day). Sensor timestamps are stored as
// Vietnam wall-clock, so the bare Weekday()/Hour() values are already local.
func inAnyWindow(wins []roomWindow, t time.Time) bool {
	day := t.Weekday().String()
	m := t.Hour()*60 + t.Minute()
	for _, w := range wins {
		if w.Day == day && m >= w.StartMin && m < w.EndMin {
			return true
		}
	}
	return false
}
