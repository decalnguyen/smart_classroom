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

// ----- Attendance daily roll-up (per classroom, summed over class-sessions) -----
// A classroom hosts many class-SESSIONS (tiết/môn) per day. The roll-up SUMS each
// session: numbers are attendance instances ("lượt"), not a distinct headcount —
// so a student with 9 periods in a room counts 9 slots. Keeps the dashboard /
// reports / overview CONSISTENT (single source of truth).

type RoomDaily struct {
	Present  int     `json:"present"`
	Late     int     `json:"late"`
	Excused  int     `json:"excused"`
	Absent   int     `json:"absent"`
	Enrolled int     `json:"enrolled"`
	Rate     float64 `json:"rate"`
}

// dailyRollup computes per-classroom attendance for a date as a SUM over that
// day's class-SESSIONS. A "slot" is one (class_id, student_id) pair; students are
// NOT deduped across the day's periods, so a room's totals are attendance instances
// ("lượt"), not a distinct headcount. Enrolled[room] = sum over the day's sessions
// of len(roster). Rate = (Present+Late)/(Enrolled-Excused).
//
// We treat the whole day as one period set (every session scheduled that weekday
// is summed): a no-show slot is excused if the student has an approved leave,
// otherwise absent. This is the "cộng dồn theo lớp/môn" the dashboard + reports
// expect. LIMITATION: makeup sessions (date-based, not day_of_week) are not summed.
func dailyRollup(classroomIDs []uint, date, weekday string) map[uint]*RoomDaily {
	res := map[uint]*RoomDaily{}
	for _, id := range classroomIDs {
		res[id] = &RoomDaily{}
	}
	if len(classroomIDs) == 0 {
		return res
	}

	// Load the day's sessions (one row per class/period) for these rooms.
	var classes []models.Class
	db.DB.Where("day_of_week = ? AND classroom_id IN ?", weekday, classroomIDs).Find(&classes)
	if len(classes) == 0 {
		return res
	}

	// classID -> classroomID; collect the class ids to sum.
	sessions := map[uint]uint{}
	classIDs := make([]uint, 0, len(classes))
	for _, cl := range classes {
		sessions[cl.ClassID] = cl.ClassroomID
		classIDs = append(classIDs, cl.ClassID)
	}

	// Roster per session (slot universe). Do NOT dedup students across sessions.
	type csRow struct {
		ClassID   uint
		StudentID uint
	}
	var rosterRows []csRow
	db.DB.Table("class_students").
		Select("class_id, student_id").
		Where("class_id IN ?", classIDs).
		Scan(&rosterRows)
	roster := map[uint][]uint{} // classID -> []studentID
	for _, r := range rosterRows {
		roster[r.ClassID] = append(roster[r.ClassID], r.StudentID)
	}

	// Per-slot status. The write-path dedup key (student_id, class_id, date)
	// guarantees at most one attendance row per slot, so no best-status precedence
	// is needed: each row contributes its own status to its session.
	type arow struct {
		ClassID   uint
		StudentID uint
		Status    string
	}
	var ars []arow
	db.DB.Table("attendances a").
		Select("a.class_id, a.student_id, a.attendance_status as status").
		Joins("JOIN classes ON classes.class_id = a.class_id").
		Where("a.date = ? AND classes.day_of_week = ? AND classes.classroom_id IN ?", date, weekday, classroomIDs).
		Scan(&ars)
	status := map[uint]map[uint]string{} // classID -> studentID -> status
	for _, a := range ars {
		if a.ClassID == 0 {
			continue
		}
		if status[a.ClassID] == nil {
			status[a.ClassID] = map[uint]string{}
		}
		status[a.ClassID][a.StudentID] = a.Status
	}

	// Approved leave for the date is the SINGLE source of truth for excused: a
	// missing slot whose student is on leave is excused in EVERY session that day.
	var leaveStudents []uint
	db.DB.Model(&models.LeaveRequest{}).
		Where("date = ? AND status = ?", date, "approved").
		Pluck("student_id", &leaveStudents)
	leaveSet := map[uint]bool{}
	for _, s := range leaveStudents {
		leaveSet[s] = true
	}

	// Tally per room = sum over the day's sessions (every slot counts).
	for cid, classroomID := range sessions {
		rd := res[classroomID]
		if rd == nil {
			rd = &RoomDaily{}
			res[classroomID] = rd
		}
		for _, sid := range roster[cid] {
			rd.Enrolled++
			switch status[cid][sid] {
			case models.StatusPresent:
				rd.Present++
			case models.StatusLate:
				rd.Late++
			case models.StatusExcused:
				rd.Excused++
			default:
				// No attendance row: excused if on approved leave, else absent.
				if leaveSet[sid] {
					rd.Excused++
				} else {
					rd.Absent++
				}
			}
		}
	}

	// Rate over slots (denominator excludes excused); guard against 0/negative.
	for _, rd := range res {
		denom := rd.Enrolled - rd.Excused
		if denom > 0 {
			rd.Rate = float64(rd.Present+rd.Late) / float64(denom)
		}
	}
	return res
}
