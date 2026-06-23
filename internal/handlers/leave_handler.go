package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
	"smart_classroom/internal/rabbitmq"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HandleCreateLeave — a student (or staff on their behalf) submits a leave request.
// After creation, notifies teachers assigned to classrooms where the student is enrolled.
func HandleCreateLeave(c *gin.Context) {
	var req struct {
		StudentID uint   `json:"student_id"`
		Date      string `json:"date"`
		Reason    string `json:"reason"`
	}
	if err := c.BindJSON(&req); err != nil || req.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}
	account := c.GetString("account_id")

	var student models.Student
	if c.GetString("role") == "student" {
		if err := db.DB.Where("account_id = ?", account).First(&student).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tài khoản chưa liên kết hồ sơ học sinh"})
			return
		}
	} else {
		if err := db.DB.Where("student_id = ?", req.StudentID).First(&student).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy học sinh"})
			return
		}
	}

	reason := req.Reason
	if reason == "" {
		reason = "Không có lý do"
	}
	lr := models.LeaveRequest{
		StudentID: student.StudentID, StudentName: student.StudentName, AccountID: account,
		Date: req.Date, Reason: reason, Status: "pending", CreatedAt: nowVN(),
	}
	if err := db.DB.Create(&lr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tạo được đơn"})
		return
	}

	// Notify teachers assigned to classrooms where this student is enrolled.
	go notifyTeachersForLeave(student, lr)

	c.JSON(http.StatusOK, lr)
}

// notifyTeachersForLeave finds teachers assigned to the student's classrooms and
// pushes a "leave" notification so their UI shows a popup immediately.
func notifyTeachersForLeave(student models.Student, lr models.LeaveRequest) {
	// Classrooms the student is enrolled in (via class_students → classes).
	var classroomIDs []uint
	db.DB.Table("class_students cs").
		Joins("JOIN classes c ON c.class_id = cs.class_id").
		Where("cs.student_id = ?", student.StudentID).
		Distinct("c.classroom_id").
		Pluck("c.classroom_id", &classroomIDs)
	if len(classroomIDs) == 0 {
		return
	}
	// Teachers assigned to those classrooms.
	var teachers []models.Teacher
	db.DB.Joins("JOIN classroom_teachers ct ON ct.teacher_id = teachers.teacher_id").
		Where("ct.classroom_id IN ?", classroomIDs).
		Distinct("teachers.teacher_id, teachers.teacher_name, teachers.account_id").
		Find(&teachers)

	msg := fmt.Sprintf("Học sinh %s xin phép nghỉ ngày %s — lý do: %s", student.StudentName, lr.Date, lr.Reason)
	now := nowVN()
	for _, t := range teachers {
		if t.AccountID == "" {
			continue
		}
		notif := models.Notification{
			ID:        uuid.New().String(),
			AccountID: t.AccountID,
			Title:     "leave",
			Message:   msg,
			IsRead:    false,
			CreatedAt: now,
		}
		db.DB.Create(&notif)
		rabbitmq.Publish("notify.data", notif)
	}
}

// HandleListLeaves — students see their own; teacher sees leaves for students in
// their assigned classrooms; admin sees all. Each leave is enriched with the
// CLASS(ES) the student misses that day (phòng / môn / tiết / giờ), scoped to the
// viewer (teacher → only their classes), so the page can group by lớp/phòng/giờ.
func HandleListLeaves(c *gin.Context) {
	role := c.GetString("role")
	q := db.DB.Model(&models.LeaveRequest{}).Order("created_at desc")

	var scopeClassroomIDs []uint // teacher: limit affected-classes to their rooms
	teacherScoped := false

	switch role {
	case "student":
		q = q.Where("account_id = ?", c.GetString("account_id"))
	case "teacher":
		var teacher models.Teacher
		if err := db.DB.Where("account_id = ?", c.GetString("account_id")).First(&teacher).Error; err == nil {
			db.DB.Model(&models.ClassroomTeacher{}).Where("teacher_id = ?", teacher.TeacherID).Pluck("classroom_id", &scopeClassroomIDs)
			teacherScoped = true
			if len(scopeClassroomIDs) == 0 {
				c.JSON(http.StatusOK, []gin.H{})
				return
			}
			var studentIDs []uint
			db.DB.Table("class_students cs").
				Joins("JOIN classes c ON c.class_id = cs.class_id").
				Where("c.classroom_id IN ?", scopeClassroomIDs).
				Distinct("cs.student_id").Pluck("cs.student_id", &studentIDs)
			if len(studentIDs) == 0 {
				c.JSON(http.StatusOK, []gin.H{})
				return
			}
			q = q.Where("student_id IN ?", studentIDs)
		}
		if s := c.Query("status"); s != "" {
			q = q.Where("status = ?", s)
		}
	default:
		if s := c.Query("status"); s != "" {
			q = q.Where("status = ?", s)
		}
	}

	var rows []models.LeaveRequest
	q.Limit(500).Find(&rows)

	// Affected classes per student, per weekday (room/subject/period/time).
	studentIDs := make([]uint, 0, len(rows))
	seen := map[uint]bool{}
	for _, r := range rows {
		if !seen[r.StudentID] {
			seen[r.StudentID] = true
			studentIDs = append(studentIDs, r.StudentID)
		}
	}
	type ci struct {
		StudentID uint
		DayOfWeek string
		Classroom string
		Subject   string
		Period    int
		StartMin  int
		EndMin    int
	}
	byStudentDay := map[uint]map[string][]gin.H{}
	if len(studentIDs) > 0 {
		cq := db.DB.Table("class_students cs").
			Select("cs.student_id, c.day_of_week, cr.classroom_name as classroom, c.subject, c.period, c.start_min, c.end_min").
			Joins("JOIN classes c ON c.class_id = cs.class_id").
			Joins("JOIN classrooms cr ON cr.classroom_id = c.classroom_id").
			Where("cs.student_id IN ?", studentIDs)
		if teacherScoped {
			cq = cq.Where("c.classroom_id IN ?", scopeClassroomIDs)
		}
		var cis []ci
		cq.Scan(&cis)
		for _, x := range cis {
			if byStudentDay[x.StudentID] == nil {
				byStudentDay[x.StudentID] = map[string][]gin.H{}
			}
			byStudentDay[x.StudentID][x.DayOfWeek] = append(byStudentDay[x.StudentID][x.DayOfWeek], gin.H{
				"classroom": x.Classroom, "subject": x.Subject, "period": x.Period,
				"start_min": x.StartMin,
				"time":      fmt.Sprintf("%02d:%02d–%02d:%02d", x.StartMin/60, x.StartMin%60, x.EndMin/60, x.EndMin%60),
			})
		}
	}

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		weekday := ""
		if t, err := time.Parse("2006-01-02", r.Date); err == nil {
			weekday = t.Weekday().String()
		}
		classes := byStudentDay[r.StudentID][weekday]
		sort.Slice(classes, func(i, j int) bool {
			return classes[i]["start_min"].(int) < classes[j]["start_min"].(int)
		})
		out = append(out, gin.H{
			"id": r.ID, "student_id": r.StudentID, "student_name": r.StudentName,
			"account_id": r.AccountID, "date": r.Date, "reason": r.Reason,
			"status": r.Status, "reviewed_by": r.ReviewedBy, "created_at": r.CreatedAt,
			"classes": classes,
		})
	}
	c.JSON(http.StatusOK, out)
}

// HandleReviewLeave — staff approve/reject. Approved leaves become "excused" in
// the attendance roll-up (no attendance row needed).
func HandleReviewLeave(c *gin.Context) {
	id := parseUintParam(c.Param("id"))
	var req struct {
		Status string `json:"status"` // approved | rejected
	}
	if err := c.BindJSON(&req); err != nil || (req.Status != "approved" && req.Status != "rejected") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be approved or rejected"})
		return
	}
	var lr models.LeaveRequest
	if err := db.DB.First(&lr, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đơn"})
		return
	}
	// A teacher may only review leaves of students enrolled in their assigned rooms.
	if c.GetString("role") == "teacher" {
		var teacher models.Teacher
		var scopeIDs []uint
		if db.DB.Where("account_id = ?", c.GetString("account_id")).First(&teacher).Error == nil {
			db.DB.Model(&models.ClassroomTeacher{}).Where("teacher_id = ?", teacher.TeacherID).Pluck("classroom_id", &scopeIDs)
		}
		var n int64
		if len(scopeIDs) > 0 {
			db.DB.Table("class_students cs").Joins("JOIN classes c ON c.class_id = cs.class_id").
				Where("c.classroom_id IN ? AND cs.student_id = ?", scopeIDs, lr.StudentID).Count(&n)
		}
		if n == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không được phân công học sinh này"})
			return
		}
	}
	now := nowVN()
	db.DB.Model(&lr).Updates(map[string]interface{}{
		"status": req.Status, "reviewed_by": c.GetString("account_id"), "reviewed_at": now,
	})
	writeAudit(c, req.Status, "leave_request", uintStr(lr.ID),
		"Đơn nghỉ của SV "+lr.StudentName+" ("+lr.Date+")")
	c.JSON(http.StatusOK, gin.H{"message": "Đã xử lý đơn", "status": req.Status})
}
