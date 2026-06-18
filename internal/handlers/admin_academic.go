package handlers

import (
	"fmt"
	"net/http"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---------- 1.4 Audit log ----------

func HandleGetAudit(c *gin.Context) {
	q := db.DB.Model(&models.AuditLog{}).Order("created_at desc")
	if e := c.Query("entity"); e != "" {
		q = q.Where("entity = ?", e)
	}
	var rows []models.AuditLog
	q.Limit(500).Find(&rows)
	c.JSON(http.StatusOK, rows)
}

// ---------- 3.1 Semester ----------

func HandleGetSemesters(c *gin.Context) {
	var rows []models.Semester
	db.DB.Order("id desc").Find(&rows)
	c.JSON(http.StatusOK, rows)
}

// ---------- 3.3 Holidays ----------

func HandleGetHolidays(c *gin.Context) {
	var rows []models.Holiday
	db.DB.Order("date asc").Find(&rows)
	c.JSON(http.StatusOK, rows)
}

func HandleCreateHoliday(c *gin.Context) {
	var h models.Holiday
	if err := c.BindJSON(&h); err != nil || h.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}
	if err := db.DB.Create(&h).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tạo được ngày lễ"})
		return
	}
	writeAudit(c, "create", "holiday", uintStr(h.ID), h.Date+" "+h.Name)
	c.JSON(http.StatusOK, h)
}

func HandleDeleteHoliday(c *gin.Context) {
	id := parseUintParam(c.Param("id"))
	res := db.DB.Delete(&models.Holiday{}, id)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy"})
		return
	}
	writeAudit(c, "delete", "holiday", uintStr(id), "")
	c.JSON(http.StatusOK, gin.H{"message": "Đã xoá ngày lễ"})
}

// ---------- 3.3 Makeup session (buổi bù) ----------

func HandleCreateMakeup(c *gin.Context) {
	var m models.MakeupSession
	if err := c.BindJSON(&m); err != nil || m.ClassID == 0 || m.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "class_id và date là bắt buộc"})
		return
	}
	if err := db.DB.Create(&m).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tạo được buổi bù"})
		return
	}
	writeAudit(c, "create", "makeup", uintStr(m.ID), fmt.Sprintf("Lớp %d ngày %s", m.ClassID, m.Date))
	c.JSON(http.StatusOK, m)
}

func HandleGetMakeups(c *gin.Context) {
	var rows []models.MakeupSession
	db.DB.Order("date asc, start_min asc").Find(&rows)
	c.JSON(http.StatusOK, rows)
}

func HandleDeleteMakeup(c *gin.Context) {
	id := parseUintParam(c.Param("id"))
	res := db.DB.Delete(&models.MakeupSession{}, id)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy"})
		return
	}
	writeAudit(c, "delete", "makeup", uintStr(id), "")
	c.JSON(http.StatusOK, gin.H{"message": "Đã xoá buổi bù"})
}

// HandleListClasses lists every class/period with its room + live enrolled count,
// for the admin class-enrollment UI. (GET /classes/:id is classroom/time scoped and
// returns no roster, so it can't be reused here.)
func HandleListClasses(c *gin.Context) {
	type classRow struct {
		ClassID       uint   `json:"class_id"`
		Subject       string `json:"subject"`
		ClassroomID   uint   `json:"classroom_id"`
		ClassroomName string `json:"classroom_name"`
		DayOfWeek     string `json:"day_of_week"`
		StartMin      int    `json:"start_min"`
		EndMin        int    `json:"end_min"`
		Capacity      int    `json:"capacity"`
		Enrolled      int64  `json:"enrolled"`
	}
	var rows []classRow
	db.DB.Table("classes").
		Select(`classes.class_id, classes.subject, classes.classroom_id, classrooms.classroom_name,
			classes.day_of_week, classes.start_min, classes.end_min, classrooms.capacity,
			(SELECT count(*) FROM class_students cs WHERE cs.class_id = classes.class_id) AS enrolled`).
		Joins("LEFT JOIN classrooms ON classrooms.classroom_id = classes.classroom_id").
		Order("classrooms.classroom_name asc, classes.start_min asc").
		Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// HandleGetClassRoster returns the students enrolled in a class (for the roster UI).
func HandleGetClassRoster(c *gin.Context) {
	classID := parseUintParam(c.Param("id"))
	type sRow struct {
		StudentID   uint   `json:"student_id"`
		MSSV        string `json:"mssv"`
		StudentName string `json:"student_name"`
	}
	var rows []sRow
	db.DB.Table("students").
		Select("students.student_id, students.mssv, students.student_name").
		Joins("JOIN class_students ON class_students.student_id = students.student_id").
		Where("class_students.class_id = ?", classID).
		Order("students.student_name asc").
		Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// ---------- 3.5 Enrollment with capacity check ----------

func HandleEnrollStudent(c *gin.Context) {
	classID := parseUintParam(c.Param("id"))
	var req struct {
		StudentID uint `json:"student_id"`
	}
	if err := c.BindJSON(&req); err != nil || req.StudentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student_id is required"})
		return
	}
	var class models.Class
	if err := db.DB.First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy lớp"})
		return
	}
	var room models.Classroom
	db.DB.First(&room, class.ClassroomID)

	var enrolled int64
	db.DB.Model(&models.ClassStudent{}).Where("class_id = ?", classID).Count(&enrolled)
	if room.Capacity > 0 && enrolled >= int64(room.Capacity) {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Lớp đã đầy (sĩ số tối đa %d)", room.Capacity)})
		return
	}
	var exists int64
	db.DB.Model(&models.ClassStudent{}).Where("class_id = ? AND student_id = ?", classID, req.StudentID).Count(&exists)
	if exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Học sinh đã ghi danh lớp này"})
		return
	}
	db.DB.Create(&models.ClassStudent{ClassID: classID, StudentID: req.StudentID})
	writeAudit(c, "create", "enrollment", uintStr(classID), fmt.Sprintf("Ghi danh SV %d", req.StudentID))
	c.JSON(http.StatusOK, gin.H{"message": "Đã ghi danh", "enrolled": enrolled + 1, "capacity": room.Capacity})
}

func HandleUnenrollStudent(c *gin.Context) {
	classID := parseUintParam(c.Param("id"))
	studentID := parseUintParam(c.Param("student_id"))
	res := db.DB.Where("class_id = ? AND student_id = ?", classID, studentID).Delete(&models.ClassStudent{})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy ghi danh"})
		return
	}
	writeAudit(c, "delete", "enrollment", uintStr(classID), fmt.Sprintf("Huỷ ghi danh SV %d", studentID))
	c.JSON(http.StatusOK, gin.H{"message": "Đã huỷ ghi danh"})
}

// ---------- 3.2 Timetable conflict detection ----------
// Returns a non-empty message if the proposed class overlaps an existing one
// in the same room or for the same teacher on the same weekday.
func classConflict(classroomID, teacherID uint, day string, startMin, endMin int, excludeID uint) string {
	var n int64
	db.DB.Model(&models.Class{}).
		Where("classroom_id = ? AND day_of_week = ? AND class_id <> ? AND start_min < ? AND end_min > ?",
			classroomID, day, excludeID, endMin, startMin).Count(&n)
	if n > 0 {
		return "Phòng học đã có lớp khác trùng khung giờ này"
	}
	if teacherID > 0 {
		db.DB.Model(&models.Class{}).
			Where("teacher_id = ? AND day_of_week = ? AND class_id <> ? AND start_min < ? AND end_min > ?",
				teacherID, day, excludeID, endMin, startMin).Count(&n)
		if n > 0 {
			return "Giáo viên đã có lớp khác trùng khung giờ này"
		}
	}
	return ""
}

// ---------- 1.2 Auto-absent: freeze no-shows after a period ends ----------

func AutoAbsentChecker() {
	go func() {
		for {
			autoCloseEndedPeriods()
			time.Sleep(2 * time.Minute)
		}
	}()
}

func autoCloseEndedPeriods() {
	now := nowVN()
	today := now.Format("2006-01-02")
	if isHoliday(today) {
		return
	}
	m := minutesOf(now)
	var classes []models.Class
	db.DB.Where("day_of_week = ? AND end_min <= ?", now.Weekday().String(), m).Find(&classes)

	// Also close ended MAKEUP sessions (buổi bù) — these are date-based, not
	// day_of_week, so the weekday query above misses them. findOngoingClass treats
	// a live makeup as the ongoing class, so its no-shows must be frozen too.
	var makeups []models.MakeupSession
	db.DB.Where("date = ? AND end_min <= ?", today, m).Find(&makeups)
	seen := map[uint]bool{}
	for _, cl := range classes {
		seen[cl.ClassID] = true
	}
	for _, mk := range makeups {
		if seen[mk.ClassID] {
			continue
		}
		var cl models.Class
		if err := db.DB.First(&cl, mk.ClassID).Error; err == nil {
			classes = append(classes, cl)
			seen[cl.ClassID] = true
		}
	}

	if len(classes) == 0 {
		return
	}
	var leaveIDs []uint
	db.DB.Model(&models.LeaveRequest{}).Where("date = ? AND status = ?", today, "approved").Pluck("student_id", &leaveIDs)
	excused := map[uint]bool{}
	for _, id := range leaveIDs {
		excused[id] = true
	}

	total := 0
	for _, cl := range classes {
		var enrolled []uint
		db.DB.Table("class_students").Where("class_id = ?", cl.ClassID).Pluck("student_id", &enrolled)
		var have []uint
		db.DB.Model(&models.Attendance{}).Where("class_id = ? AND date = ?", cl.ClassID, today).Pluck("student_id", &have)
		haveSet := map[uint]bool{}
		for _, id := range have {
			haveSet[id] = true
		}
		rows := make([]models.Attendance, 0)
		for _, sid := range enrolled {
			if haveSet[sid] || excused[sid] {
				continue
			}
			id := uuid.New().String()
			cid := cl.ClassID
			subj := cl.Subject
			rows = append(rows, models.Attendance{
				ID: &id, StudentID: sid, ClassroomID: cl.ClassroomID, ClassID: &cid, Subject: &subj,
				Date: today, AttendanceStatus: models.StatusAbsent, DetectionTime: "", DeviceID: "auto",
			})
		}
		if len(rows) > 0 {
			db.DB.CreateInBatches(rows, 500)
			total += len(rows)
		}
	}
	// (intentionally quiet unless something was closed)
	_ = total
}
