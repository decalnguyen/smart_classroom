package handlers

import (
	"fmt"
	"math/rand"
	"net/http"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
	"smart_classroom/internal/rabbitmq"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AttendanceEvent is the realtime payload broadcast on a successful face scan.
type AttendanceEvent struct {
	StudentID   uint   `json:"student_id"`
	MSSV        string `json:"mssv"`
	StudentName string `json:"student_name"`
	ClassroomID uint   `json:"classroom_id"`
	ClassID     uint   `json:"class_id"`
	Subject     string `json:"subject"`
	Status      string  `json:"attendance_status"`
	Time        string  `json:"detection_time"`
	Date        string  `json:"date"`
	DeviceID    string  `json:"device_id"`
	Confidence  float64 `json:"confidence,omitempty"`
}

// HandleAttendanceScan simulates the edge AI camera reporting a recognized face.
// Public device endpoint (like /sensor): no user JWT required.
//
// Body: { classroom_id (required), student_id (optional), device_id (optional) }.
// If student_id is omitted, the server picks a random enrolled student of the
// ongoing class who is not yet present (mimicking a fresh recognition). On
// success it persists attendance and broadcasts an AttendanceEvent over the
// realtime attendance channel so the web updates live (name, MSSV, time, status).
func HandleAttendanceScan(c *gin.Context) {
	var req struct {
		ClassroomID uint      `json:"classroom_id"`
		StudentID   uint      `json:"student_id"`
		DeviceID    string    `json:"device_id"`
		Status      string    `json:"status"`     // present | late (default present)
		Embedding   []float64 `json:"embedding"`  // optional ArcFace 512-d from the edge
		EventID     string    `json:"event_id"`   // optional edge-side idempotency key
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if req.ClassroomID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classroom_id is required"})
		return
	}
	if req.DeviceID == "" {
		req.DeviceID = fmt.Sprintf("cam-%d", req.ClassroomID)
	}

	now := nowVN()

	// Find the ongoing period for this classroom (respects holidays/makeups).
	class, ok := findOngoingClass(req.ClassroomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không có tiết học đang diễn ra trong phòng này"})
		return
	}
	// Late policy: present within grace of the period start, else late.
	status := checkinStatus(class)
	if req.Status == models.StatusLate {
		status = models.StatusLate
	}

	// Edge recognition path: when the camera sends a 512-d embedding, match it
	// against the classroom gallery (pgvector cosine) and apply confidence gating:
	//   sim >= T_high → accept (auto-mark); T_low <= sim < T_high → human review;
	//   sim < T_low   → unknown face (ignored).
	confidence := 1.0
	if len(req.Embedding) == 512 {
		sid, mssv, name, sim, matched := recognizeByEmbedding(req.ClassroomID, req.Embedding)
		if !matched {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chưa có khuôn mặt ghi danh trong phòng này"})
			return
		}
		confidence = sim
		switch {
		case sim < tLow():
			c.JSON(http.StatusOK, gin.H{"message": "Khuôn mặt không xác định", "confidence": sim})
			return
		case sim < tHigh():
			db.DB.Create(&models.FaceReview{
				StudentID: sid, MSSV: mssv, StudentName: name,
				ClassroomID: req.ClassroomID, ClassID: class.ClassID, Subject: class.Subject,
				Confidence: sim, Date: now.Format("2006-01-02"),
				DetectionTime: now.Format("15:04:05"), DeviceID: req.DeviceID, Status: "pending",
			})
			c.JSON(http.StatusOK, gin.H{"message": "Độ tin cậy thấp — chờ duyệt", "confidence": sim, "student_name": name})
			return
		default:
			req.StudentID = sid // high confidence → treat as a positive recognition
		}
	}

	// Resolve the recognized student.
	var student models.Student
	if req.StudentID != 0 {
		if err := db.DB.Where("student_id = ?", req.StudentID).First(&student).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
			return
		}
	} else {
		var enrolled []models.Student
		if err := db.DB.
			Joins("JOIN class_students ON students.student_id = class_students.student_id").
			Where("class_students.class_id = ?", class.ClassID).
			Find(&enrolled).Error; err != nil || len(enrolled) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No students enrolled in the ongoing class"})
			return
		}
		var presentIDs []uint
		db.DB.Model(&models.Attendance{}).
			Where("class_id = ? AND date = ? AND attendance_status IN ?", class.ClassID, now.Format("2006-01-02"), []string{"present", "late"}).
			Pluck("student_id", &presentIDs)
		present := map[uint]bool{}
		for _, id := range presentIDs {
			present[id] = true
		}
		candidates := make([]models.Student, 0, len(enrolled))
		for _, s := range enrolled {
			if !present[s.StudentID] {
				candidates = append(candidates, s)
			}
		}
		// Cap attendance at ~90% so the numbers stay realistic (not everyone
		// shows up). Beyond the cap, just re-broadcast an already-present student
		// to keep the live recognition feed alive — without creating a new row.
		target := int(0.9 * float64(len(enrolled)))
		if len(present) >= target || len(candidates) == 0 {
			if len(presentIDs) > 0 {
				var st models.Student
				if db.DB.Where("student_id = ?", presentIDs[rand.Intn(len(presentIDs))]).First(&st).Error == nil {
					rabbitmq.Publish("attendance.event", AttendanceEvent{
						StudentID: st.StudentID, MSSV: st.MSSV, StudentName: st.StudentName,
						ClassroomID: req.ClassroomID, ClassID: class.ClassID, Subject: class.Subject,
						Status: "present", Time: now.Format("15:04:05"), Date: now.Format("2006-01-02"), DeviceID: req.DeviceID,
					})
				}
			}
			c.JSON(http.StatusOK, gin.H{"message": "Lớp đã đạt tỉ lệ điểm danh mục tiêu"})
			return
		}
		student = candidates[rand.Intn(len(candidates))]
	}

	dateStr := now.Format("2006-01-02")

	// Dedup: one attendance row per (student, class, date). Update status if it
	// already exists instead of inserting a duplicate (prevents rate > 100%).
	var existing models.Attendance
	if db.DB.Where("student_id = ? AND class_id = ? AND date = ?", student.StudentID, class.ClassID, dateStr).
		First(&existing).Error == nil {
		if existing.AttendanceStatus != status {
			db.DB.Model(&existing).Update("attendance_status", status)
		}
		c.JSON(http.StatusOK, gin.H{"message": "Học sinh đã được điểm danh trước đó", "student_id": student.StudentID})
		return
	}

	id := uuid.New().String()
	att := models.Attendance{
		ID:               &id,
		StudentID:        student.StudentID,
		ClassroomID:      req.ClassroomID,
		ClassID:          &class.ClassID,
		Subject:          &class.Subject,
		Date:             dateStr,
		AttendanceStatus: status,
		DetectionTime:    now.Format("15:04:05"),
		DeviceID:         req.DeviceID,
	}
	if err := db.DB.Create(&att).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record attendance"})
		return
	}

	// Broadcast the recognition to the realtime attendance channel.
	event := AttendanceEvent{
		StudentID:   student.StudentID,
		MSSV:        student.MSSV,
		StudentName: student.StudentName,
		ClassroomID: req.ClassroomID,
		ClassID:     class.ClassID,
		Subject:     class.Subject,
		Status:      status,
		Time:        att.DetectionTime,
		Date:        att.Date,
		DeviceID:    req.DeviceID,
		Confidence:  confidence,
	}
	rabbitmq.Publish("attendance.event", event)

	// Per-student notification (only if the student has a linked account).
	if student.AccountID != "" {
		notif := models.Notification{
			ID:        uuid.New().String(),
			AccountID: student.AccountID,
			Title:     "attendance",
			Message:   fmt.Sprintf("Bạn đã được điểm danh môn %s lúc %s", class.Subject, att.DetectionTime),
			IsRead:    false,
			CreatedAt: now,
		}
		db.DB.Create(&notif)
		rabbitmq.Publish("notify.data", notif)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Face recognized", "event": event})
}
