package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
)

// HandleClassroomsOverview returns a per-classroom snapshot (latest environment
// readings + today's attendance + safety state), scoped to the caller's role.
// Powers the "all classrooms" dashboard overview.
func HandleClassroomsOverview(c *gin.Context) {
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)
	today := now.Format("2006-01-02")
	weekday := now.Weekday().String()

	// Room scope: teacher = rooms they teach, student = enrolled rooms, admin = all.
	scopeRooms, scopeWins, isAll := actorRoomScope(c)
	var classrooms []models.Classroom
	if isAll {
		db.DB.Order("classroom_id asc").Find(&classrooms)
	} else {
		if len(scopeRooms) == 0 {
			c.JSON(http.StatusOK, []gin.H{})
			return
		}
		db.DB.Where("classroom_name IN ?", roomNames(scopeRooms)).Order("classroom_id asc").Find(&classrooms)
	}
	ids := make([]uint, 0, len(classrooms))
	for _, cr := range classrooms {
		ids = append(ids, cr.ClassroomID)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	buildings := map[uint]string{}
	var bs []models.Building
	db.DB.Find(&bs)
	for _, b := range bs {
		buildings[b.BuildingID] = b.BuildingName
	}

	// Teacher names, to label the class currently in each room.
	teacherNames := map[uint]string{}
	var ts []models.Teacher
	db.DB.Find(&ts)
	for _, t := range ts {
		teacherNames[t.TeacherID] = t.TeacherName
	}

	// Today's attendance per classroom (reuses the report helper).
	att := map[uint]classroomReport{}
	rows, _, _, _, _ := computeByClassroom(ids, today, weekday)
	for _, r := range rows {
		att[r.ClassroomID] = r
	}

	// Latest reading per device in the last 30 minutes.
	type lr struct {
		DeviceID   string
		DeviceType string
		Value      float64
	}
	var lrs []lr
	db.DB.Raw(`SELECT DISTINCT ON (device_id) device_id, device_type, value
	           FROM sen_sor_data WHERE timestamp > ? ORDER BY device_id, timestamp DESC`,
		now.Add(-30*time.Minute)).Scan(&lrs)

	smokeThr := dangerThreshold("smoke", "SMOKE_THRESHOLD", 300)
	tempThr := dangerThreshold("temp", "TEMP_THRESHOLD", 50)

	out := make([]gin.H, 0, len(classrooms))
	for _, cr := range classrooms {
		prefix := cr.ClassroomName + "-"
		sensors := map[string]float64{}
		for _, x := range lrs {
			if strings.HasPrefix(x.DeviceID, prefix) {
				// Bucket by CANONICAL short type so the dashboard reads the same
				// taxonomy telemetry actually stores (temp/humi, not temperature).
				sensors[canonicalType(x.DeviceType)] = x.Value
			}
		}
		a := att[cr.ClassroomID]
		// fresh = the room produced at least one reading in the 30-min window, so the
		// UI can tell "offline / no recent data" apart from a genuine 0 reading.
		fresh := len(sensors) > 0
		danger := sensors["smoke"] >= smokeThr || sensors["temp"] >= tempThr
		// inSession: for a teacher/student, is one of THEIR periods running now in
		// this room? (admin always true). Outside their teaching/study window, the
		// room is HIDDEN — a GV/SV chỉ giám sát phòng đang có tiết của mình ("khung giờ").
		inSession := isAll || inAnyWindow(scopeWins[cr.ClassroomName], now)
		if !isAll && !inSession {
			continue
		}
		// Which class is in this room right now (subject/period/time/teacher), if any.
		var currentClass interface{}
		if cl, ok := findOngoingClass(cr.ClassroomID); ok {
			currentClass = gin.H{
				"subject": cl.Subject,
				"period":  cl.Period,
				"time":    fmt.Sprintf("%02d:%02d–%02d:%02d", cl.StartMin/60, cl.StartMin%60, cl.EndMin/60, cl.EndMin%60),
				"teacher": teacherNames[cl.TeacherID],
			}
		}
		out = append(out, gin.H{
			"classroom_id":   cr.ClassroomID,
			"classroom_name": cr.ClassroomName,
			"building":        buildings[cr.BuildingID],
			"sensors": gin.H{
				"light":       sensors["light"],
				"temperature": sensors["temp"],
				"humidity":    sensors["humi"],
				"smoke":       sensors["smoke"],
			},
			"fresh":         fresh,
			"in_session":    inSession,
			"current_class": currentClass,
			"attendance": gin.H{
				"present": a.Present, "late": a.Late, "excused": a.Excused, "absent": a.Absent,
				"enrolled": a.Enrolled, "rate": a.Rate,
			},
			"danger": danger,
		})
	}
	c.JSON(http.StatusOK, out)
}

// scopedClassroomIDs returns the classroom IDs the caller may see.
// admin → all classrooms (isAll=true); teacher → only assigned classrooms; others → none.
func scopedClassroomIDs(c *gin.Context) (ids []uint, isAll bool) {
	role := c.GetString("role")
	if role == "teacher" {
		account := c.GetString("account_id")
		var teacher models.Teacher
		if err := db.DB.Where("account_id = ?", account).First(&teacher).Error; err != nil {
			return []uint{}, false
		}
		db.DB.Model(&models.ClassroomTeacher{}).Where("teacher_id = ?", teacher.TeacherID).Pluck("classroom_id", &ids)
		return ids, false
	}
	// admin and student may view all classrooms (students read-only via UI).
	if role == "admin" || role == "student" {
		db.DB.Model(&models.Classroom{}).Pluck("classroom_id", &ids)
		return ids, true
	}
	return []uint{}, false
}

func containsUint(s []uint, v uint) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// HandleMyClassrooms returns the classrooms in the caller's scope (admin=all, teacher=assigned).
func HandleMyClassrooms(c *gin.Context) {
	ids, _ := scopedClassroomIDs(c)
	rooms := []models.Classroom{}
	if len(ids) > 0 {
		db.DB.Where("classroom_id IN ?", ids).Order("classroom_id asc").Find(&rooms)
	}
	c.JSON(http.StatusOK, rooms)
}

// HandleMyAttendance returns the caller's own attendance history (for students).
func HandleMyAttendance(c *gin.Context) {
	account := c.GetString("account_id")
	var student models.Student
	if err := db.DB.Where("account_id = ?", account).First(&student).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"linked": false, "records": []any{}, "summary": gin.H{"total": 0, "present": 0, "late": 0}})
		return
	}
	type rec struct {
		Date          string `json:"date"`
		DetectionTime string `json:"detection_time"`
		Subject       string `json:"subject"`
		Status        string `json:"status"`
		ClassroomName string `json:"classroom_name"`
	}
	var records []rec
	db.DB.Table("attendances a").
		Select("a.date, a.detection_time, COALESCE(a.subject,'') as subject, a.attendance_status as status, COALESCE(classrooms.classroom_name,'') as classroom_name").
		Joins("LEFT JOIN classrooms ON classrooms.classroom_id = a.classroom_id").
		Where("a.student_id = ?", student.StudentID).
		Order("a.date desc, a.detection_time desc").
		Limit(300).Scan(&records)

	present, late, excused, absent := 0, 0, 0, 0
	for _, r := range records {
		switch r.Status {
		case "present":
			present++
		case "late":
			late++
		case "excused":
			excused++
		case "absent":
			absent++
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"linked":  true,
		"student": gin.H{"student_id": student.StudentID, "mssv": student.MSSV, "student_name": student.StudentName},
		"summary": gin.H{"total": len(records), "present": present, "late": late, "excused": excused, "absent": absent},
		"records": records,
	})
}

type classroomReport struct {
	ClassroomID   uint    `json:"classroom_id"`
	ClassroomName string  `json:"classroom_name"`
	Subject       string  `json:"subject"`
	Present       int     `json:"present"`
	Late          int     `json:"late"`
	Excused       int     `json:"excused"`
	Enrolled      int     `json:"enrolled"`
	Absent        int     `json:"absent"`
	Rate          float64 `json:"rate"`
}

type datePoint struct {
	Date    string `json:"date"`
	Present int    `json:"present"`
}

// computeByClassroom builds the per-classroom daily breakdown using the shared
// student-level roll-up, so dashboard / reports / overview always agree.
func computeByClassroom(ids []uint, dateStr, weekday string) (rows []classroomReport, totPresent, totLate, totExcused, totEnrolled int) {
	roll := dailyRollup(ids, dateStr, weekday)
	var classrooms []models.Classroom
	db.DB.Where("classroom_id IN ?", ids).Order("classroom_id asc").Find(&classrooms)
	rows = make([]classroomReport, 0, len(classrooms))
	for _, cr := range classrooms {
		rd := roll[cr.ClassroomID]
		if rd == nil {
			rd = &RoomDaily{}
		}
		rows = append(rows, classroomReport{
			ClassroomID: cr.ClassroomID, ClassroomName: cr.ClassroomName, Subject: cr.Subject,
			Present: rd.Present, Late: rd.Late, Excused: rd.Excused, Enrolled: rd.Enrolled, Absent: rd.Absent, Rate: rd.Rate,
		})
		totPresent += rd.Present
		totLate += rd.Late
		totExcused += rd.Excused
		totEnrolled += rd.Enrolled
	}
	return rows, totPresent, totLate, totExcused, totEnrolled
}

// HandleAttendanceReport returns attendance analytics scoped to the caller's role.
// The per-classroom breakdown + KPI totals are a SNAPSHOT of the range's end day
// (`to`) — so each class bar equals its real sĩ số that day (present+late+excused+
// absent = enrolled). The daily trend line spans the full [from, to] range.
// Query: ?from=YYYY-MM-DD &to=YYYY-MM-DD (default: last 7 days, snapshot = today).
func HandleAttendanceReport(c *gin.Context) {
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)

	ids, isAll := scopedClassroomIDs(c)
	if len(ids) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"scope": c.GetString("role"), "is_all": isAll,
			"from": now.Format("2006-01-02"), "to": now.Format("2006-01-02"),
			"snapshot_day": now.Format("2006-01-02"),
			"totals":       gin.H{"present": 0, "late": 0, "excused": 0, "enrolled": 0, "absent": 0, "rate": 0},
			"by_classroom": []classroomReport{}, "by_date": []datePoint{},
		})
		return
	}

	from := c.Query("from")
	to := c.Query("to")
	if to == "" {
		to = now.Format("2006-01-02")
	}
	if from == "" {
		from = now.AddDate(0, 0, -6).Format("2006-01-02")
	}
	toT, err := time.ParseInLocation("2006-01-02", to, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ngày không hợp lệ (YYYY-MM-DD)"})
		return
	}

	// Per-classroom breakdown + KPIs: single-day snapshot of `to` (student-level
	// roll-up), so present+late+excused+absent = enrolled for each class.
	byClassroom, totPresent, totLate, totExcused, totEnrolled := computeByClassroom(ids, to, toT.Weekday().String())

	// Daily trend: present/late SLOTS per date (count(*) — same "lượt" grain as the
	// KPI totals and bars, since the write-path dedup key is (student_id,class_id,date)).
	var trend []datePoint
	db.DB.Table("attendances").Select("date, count(*) as present").
		Where("classroom_id IN ? AND attendance_status IN ? AND date BETWEEN ? AND ?", ids, []string{"present", "late"}, from, to).
		Group("date").Order("date asc").Scan(&trend)

	totalAbsent := totEnrolled - totPresent - totLate - totExcused
	if totalAbsent < 0 {
		totalAbsent = 0
	}
	totalRate := 0.0
	if d := totEnrolled - totExcused; d > 0 {
		totalRate = float64(totPresent+totLate) / float64(d)
	}

	c.JSON(http.StatusOK, gin.H{
		"scope":        c.GetString("role"),
		"is_all":       isAll,
		"from":         from,
		"to":           to,
		"snapshot_day": to,
		"totals":       gin.H{"present": totPresent, "late": totLate, "excused": totExcused, "enrolled": totEnrolled, "absent": totalAbsent, "rate": totalRate},
		"by_classroom": byClassroom,
		"by_session":   computeBySession(ids, to, toT.Weekday().String()),
		"by_date":      trend,
	})
}

// SessionStat is one class-SESSION's attendance for a day (per room/subject/period),
// so a teacher/admin can see each môn học diễn ra ở phòng tách biệt — the detail the
// summed per-room view (computeByClassroom) intentionally folds together.
type SessionStat struct {
	ClassID       uint    `json:"class_id"`
	ClassroomID   uint    `json:"classroom_id"`
	ClassroomName string  `json:"classroom_name"`
	Subject       string  `json:"subject"`
	Period        int     `json:"period"`
	StartMin      int     `json:"start_min"` // minutes from midnight — clock-time axis + period ordering
	EndMin        int     `json:"end_min"`
	AllDay        bool    `json:"all_day"` // synthetic all-day class (demo) — excluded from period/time charts
	Present       int     `json:"present"`
	Late          int     `json:"late"`
	Excused       int     `json:"excused"`
	Absent        int     `json:"absent"`
	Enrolled      int     `json:"enrolled"`
	Rate          float64 `json:"rate"`
	Ended         bool    `json:"ended"`
}

// computeBySession returns per-class-session attendance for `date`, applying the
// SAME now-gate as dailyRollup (future sessions omitted; ongoing no-shows pending).
func computeBySession(ids []uint, date, weekday string) []SessionStat {
	if len(ids) == 0 {
		return []SessionStat{}
	}
	now := nowVN()
	isToday := date == now.Format("2006-01-02")
	m := minutesOf(now)

	var classes []models.Class
	db.DB.Where("day_of_week = ? AND classroom_id IN ?", weekday, ids).
		Order("classroom_id asc, period asc").Find(&classes)
	if len(classes) == 0 {
		return []SessionStat{}
	}
	classIDs := make([]uint, 0, len(classes))
	for _, cl := range classes {
		classIDs = append(classIDs, cl.ClassID)
	}

	type csRow struct {
		ClassID   uint
		StudentID uint
	}
	var rosterRows []csRow
	db.DB.Table("class_students").Select("class_id, student_id").
		Where("class_id IN ?", classIDs).Scan(&rosterRows)
	roster := map[uint][]uint{}
	for _, r := range rosterRows {
		roster[r.ClassID] = append(roster[r.ClassID], r.StudentID)
	}

	type arow struct {
		ClassID   uint
		StudentID uint
		Status    string
	}
	var ars []arow
	db.DB.Table("attendances").Select("class_id, student_id, attendance_status as status").
		Where("date = ? AND class_id IN ?", date, classIDs).Scan(&ars)
	status := map[uint]map[uint]string{}
	for _, a := range ars {
		if status[a.ClassID] == nil {
			status[a.ClassID] = map[uint]string{}
		}
		status[a.ClassID][a.StudentID] = a.Status
	}

	var leaveStudents []uint
	db.DB.Model(&models.LeaveRequest{}).Where("date = ? AND status = ?", date, "approved").Pluck("student_id", &leaveStudents)
	leaveSet := map[uint]bool{}
	for _, s := range leaveStudents {
		leaveSet[s] = true
	}

	var rooms []models.Classroom
	db.DB.Where("classroom_id IN ?", ids).Find(&rooms)
	roomName := map[uint]string{}
	for _, r := range rooms {
		roomName[r.ClassroomID] = r.ClassroomName
	}

	out := make([]SessionStat, 0, len(classes))
	for _, cl := range classes {
		ended := !isToday || m >= cl.EndMin // informational (status chip) only
		ss := SessionStat{
			ClassID: cl.ClassID, ClassroomID: cl.ClassroomID, ClassroomName: roomName[cl.ClassroomID],
			Subject: cl.Subject, Period: cl.Period, Ended: ended,
			StartMin: cl.StartMin, EndMin: cl.EndMin, AllDay: cl.EndMin-cl.StartMin >= 1380,
		}
		for _, sid := range roster[cl.ClassID] {
			ss.Enrolled++
			switch status[cl.ClassID][sid] {
			case models.StatusPresent:
				ss.Present++
			case models.StatusLate:
				ss.Late++
			case models.StatusExcused:
				ss.Excused++
			default:
				if leaveSet[sid] {
					ss.Excused++
				} else {
					ss.Absent++
				}
			}
		}
		if denom := ss.Enrolled - ss.Excused; denom > 0 {
			ss.Rate = float64(ss.Present+ss.Late) / float64(denom)
		}
		out = append(out, ss)
	}
	return out
}

// HandleAttendanceReportExport streams the attendance report (scoped to the
// caller) as CSV or XLSX. Query:
//   ?from=&to=   date range (default: ?date= or today). ?to defaults to ?from.
//   ?detail=1    per-STUDENT rows (Ngày/Phòng/Môn/MSSV/Họ tên/Trạng thái) instead
//                of the per-classroom daily summary.
//   ?format=xlsx true Excel workbook; otherwise CSV (UTF-8 BOM for Excel).
func HandleAttendanceReportExport(c *gin.Context) {
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)
	ids, _ := scopedClassroomIDs(c)

	from := c.Query("from")
	to := c.Query("to")
	if from == "" {
		from = c.Query("date")
	}
	if from == "" {
		from = now.Format("2006-01-02")
	}
	if to == "" {
		to = from
	}
	fromT, err1 := time.ParseInLocation("2006-01-02", from, loc)
	toT, err2 := time.ParseInLocation("2006-01-02", to, loc)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khoảng ngày không hợp lệ (YYYY-MM-DD)"})
		return
	}
	if toT.Before(fromT) {
		fromT, toT = toT, fromT
		from, to = to, from
	}

	detail := c.Query("detail") == "1" || c.Query("detail") == "true"
	format := c.Query("format")

	var headers []string
	var rows [][]string
	if detail {
		headers = []string{"Ngày", "Phòng", "Môn", "MSSV", "Họ tên", "Trạng thái"}
		rows = attendanceDetailRows(ids, from, to)
	} else {
		headers = []string{"Ngày", "Phòng", "Môn", "Sĩ số", "Có mặt", "Đi muộn", "Có phép", "Vắng", "Tỉ lệ (%)"}
		for d := fromT; !d.After(toT); d = d.AddDate(0, 0, 1) {
			ds := d.Format("2006-01-02")
			crows, _, _, _, _ := computeByClassroom(ids, ds, d.Weekday().String())
			for _, r := range crows {
				rows = append(rows, []string{
					ds, r.ClassroomName, r.Subject,
					strconv.Itoa(r.Enrolled), strconv.Itoa(r.Present), strconv.Itoa(r.Late),
					strconv.Itoa(r.Excused), strconv.Itoa(r.Absent), strconv.Itoa(int(r.Rate * 100)),
				})
			}
		}
	}

	base := "diem_danh_" + from
	if to != from {
		base += "_" + to
	}
	if detail {
		base += "_chitiet"
	}

	if format == "xlsx" {
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename="+base+".xlsx")
		if err := writeXLSX(c.Writer, "DiemDanh", headers, rows); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tạo được file Excel"})
		}
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+base+".csv")
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM so Excel renders Vietnamese
	w := csv.NewWriter(c.Writer)
	_ = w.Write(headers)
	for _, r := range rows {
		_ = w.Write(r)
	}
	w.Flush()
}

// attendanceDetailRows returns per-student attendance records in [from,to] for the
// given classrooms, with Vietnamese status labels.
func attendanceDetailRows(ids []uint, from, to string) [][]string {
	type row struct {
		Date          string
		ClassroomName string
		Subject       string
		MSSV          string
		StudentName   string
		Status        string
	}
	var rs []row
	db.DB.Table("attendances a").
		Select(`a.date, COALESCE(r.classroom_name,'') as classroom_name, COALESCE(a.subject,'') as subject,
			COALESCE(s.mssv,'') as mssv, COALESCE(s.student_name,'') as student_name, a.attendance_status as status`).
		Joins("LEFT JOIN classrooms r ON r.classroom_id = a.classroom_id").
		Joins("LEFT JOIN students s ON s.student_id = a.student_id").
		Where("a.classroom_id IN ? AND a.date BETWEEN ? AND ?", ids, from, to).
		Order("a.date asc, r.classroom_name asc, s.student_name asc").
		Scan(&rs)
	out := make([][]string, 0, len(rs))
	for _, x := range rs {
		out = append(out, []string{x.Date, x.ClassroomName, x.Subject, x.MSSV, x.StudentName, statusVN(x.Status)})
	}
	return out
}

func statusVN(s string) string {
	switch s {
	case models.StatusPresent:
		return "Có mặt"
	case models.StatusLate:
		return "Đi muộn"
	case models.StatusExcused:
		return "Có phép"
	case models.StatusAbsent:
		return "Vắng"
	}
	return s
}
