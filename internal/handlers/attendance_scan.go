package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

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
	Status      string `json:"attendance_status"`
	Time        string `json:"detection_time"`
	Date        string `json:"date"`
	DeviceID    string `json:"device_id"`
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
		ClassroomID uint   `json:"classroom_id"`
		StudentID   uint   `json:"student_id"`
		DeviceID    string `json:"device_id"`
		Status      string `json:"status"` // present | late (default present)
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if req.ClassroomID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classroom_id is required"})
		return
	}
	status := "present"
	if req.Status == "late" {
		status = "late"
	}
	if req.DeviceID == "" {
		req.DeviceID = fmt.Sprintf("cam-%d", req.ClassroomID)
	}

	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)
	weekday := now.Weekday().String()

	// Find the ongoing class in this classroom.
	var class models.Class
	if err := db.DB.Where("classroom_id = ? AND day_of_week = ? AND start_time <= ? AND end_time >= ?",
		req.ClassroomID, weekday, now, now).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No ongoing class in this classroom"})
		return
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
		if len(candidates) == 0 {
			candidates = enrolled // everyone already present: re-scan a random one
		}
		student = candidates[rand.Intn(len(candidates))]
	}

	id := uuid.New().String()
	att := models.Attendance{
		ID:               &id,
		StudentID:        student.StudentID,
		ClassroomID:      req.ClassroomID,
		ClassID:          &class.ClassID,
		Subject:          &class.Subject,
		Date:             now.Format("2006-01-02"),
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
