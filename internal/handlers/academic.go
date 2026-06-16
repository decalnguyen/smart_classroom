package handlers

import (
	"os"
	"strconv"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
)

var vnLoc = time.FixedZone("UTC+7", 7*60*60)

func nowVN() time.Time          { return time.Now().In(vnLoc) }
func minutesOf(t time.Time) int { return t.Hour()*60 + t.Minute() }
func uintStr(n uint) string     { return strconv.FormatUint(uint64(n), 10) }

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// graceMin: a check-in within StartMin+grace counts as present, otherwise late.
func graceMin() int { return envInt("LATE_GRACE_MIN", 5) }

func isHoliday(date string) bool {
	var n int64
	db.DB.Model(&models.Holiday{}).Where("date = ?", date).Count(&n)
	return n > 0
}

// findOngoingClass returns the class/period currently in session for a classroom
// (respecting holidays and makeup sessions). ok=false if none / holiday.
func findOngoingClass(classroomID uint) (models.Class, bool) {
	now := nowVN()
	date := now.Format("2006-01-02")
	if isHoliday(date) {
		return models.Class{}, false
	}
	m := minutesOf(now)

	// Makeup session on this date for this classroom takes priority.
	var makeups []models.MakeupSession
	db.DB.Where("date = ? AND start_min <= ? AND end_min > ?", date, m, m).Find(&makeups)
	for _, mk := range makeups {
		var cl models.Class
		if db.DB.Where("class_id = ? AND classroom_id = ?", mk.ClassID, classroomID).First(&cl).Error == nil {
			return cl, true
		}
	}

	var cl models.Class
	if db.DB.Where("classroom_id = ? AND day_of_week = ? AND start_min <= ? AND end_min > ?",
		classroomID, now.Weekday().String(), m, m).
		Order("start_min asc").First(&cl).Error == nil {
		return cl, true
	}
	return models.Class{}, false
}

// checkinStatus applies the late policy: present within grace, else late.
func checkinStatus(cl models.Class) string {
	if minutesOf(nowVN()) <= cl.StartMin+graceMin() {
		return models.StatusPresent
	}
	return models.StatusLate
}

// writeAudit records a sensitive action (immutable).
func writeAudit(c *gin.Context, action, entity, entityID, detail string) {
	actorID := c.GetString("account_id")
	name := ""
	if actorID != "" {
		var u models.User
		if db.DB.Where("account_id = ?", actorID).First(&u).Error == nil {
			name = u.Username
		}
	}
	db.DB.Create(&models.AuditLog{
		ActorID: actorID, ActorName: name, ActorRole: c.GetString("role"),
		Action: action, Entity: entity, EntityID: entityID, Detail: detail,
		CreatedAt: nowVN(),
	})
}

// ----- Attendance daily roll-up (per classroom, per student) -----
// Keeps the dashboard / reports / overview CONSISTENT: one status per student
// per classroom per day (precedence present > late > excused > absent).

type RoomDaily struct {
	Present  int     `json:"present"`
	Late     int     `json:"late"`
	Excused  int     `json:"excused"`
	Absent   int     `json:"absent"`
	Enrolled int     `json:"enrolled"`
	Rate     float64 `json:"rate"`
}

// dailyRollup computes per-classroom attendance for a date across that day's periods.
func dailyRollup(classroomIDs []uint, date, weekday string) map[uint]*RoomDaily {
	res := map[uint]*RoomDaily{}
	for _, id := range classroomIDs {
		res[id] = &RoomDaily{}
	}
	if len(classroomIDs) == 0 {
		return res
	}

	// Enrolled students per classroom today (distinct).
	type pair struct {
		ClassroomID uint
		StudentID   uint
	}
	var enr []pair
	db.DB.Table("class_students cs").
		Select("DISTINCT classes.classroom_id, cs.student_id").
		Joins("JOIN classes ON classes.class_id = cs.class_id").
		Where("classes.day_of_week = ? AND classes.classroom_id IN ?", weekday, classroomIDs).
		Scan(&enr)
	enrolled := map[uint]map[uint]bool{} // classroom -> set(student)
	for _, p := range enr {
		if enrolled[p.ClassroomID] == nil {
			enrolled[p.ClassroomID] = map[uint]bool{}
		}
		enrolled[p.ClassroomID][p.StudentID] = true
	}

	// Attendance rows for today's periods.
	type arow struct {
		ClassroomID uint
		StudentID   uint
		Status      string
	}
	var ars []arow
	db.DB.Table("attendances a").
		Select("classes.classroom_id, a.student_id, a.attendance_status as status").
		Joins("JOIN classes ON classes.class_id = a.class_id").
		Where("a.date = ? AND classes.day_of_week = ? AND classes.classroom_id IN ?", date, weekday, classroomIDs).
		Scan(&ars)
	// best status per (classroom, student): present > late
	best := map[uint]map[uint]string{}
	rank := map[string]int{models.StatusPresent: 3, models.StatusLate: 2, models.StatusExcused: 1}
	for _, a := range ars {
		if best[a.ClassroomID] == nil {
			best[a.ClassroomID] = map[uint]string{}
		}
		cur := best[a.ClassroomID][a.StudentID]
		if rank[a.Status] > rank[cur] {
			best[a.ClassroomID][a.StudentID] = a.Status
		}
	}

	// Approved leaves for the date -> excused (if not already present/late).
	var leaveStudents []uint
	db.DB.Model(&models.LeaveRequest{}).Where("date = ? AND status = ?", date, "approved").Pluck("student_id", &leaveStudents)
	leaveSet := map[uint]bool{}
	for _, s := range leaveStudents {
		leaveSet[s] = true
	}

	for cid, students := range enrolled {
		rd := res[cid]
		if rd == nil {
			rd = &RoomDaily{}
			res[cid] = rd
		}
		rd.Enrolled = len(students)
		for sid := range students {
			st := best[cid][sid]
			if st == "" && leaveSet[sid] {
				st = models.StatusExcused
			}
			switch st {
			case models.StatusPresent:
				rd.Present++
			case models.StatusLate:
				rd.Late++
			case models.StatusExcused:
				rd.Excused++
			default:
				rd.Absent++
			}
		}
		denom := rd.Enrolled - rd.Excused
		if denom > 0 {
			rd.Rate = float64(rd.Present+rd.Late) / float64(denom)
		}
	}
	return res
}
