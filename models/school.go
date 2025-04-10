package models

import "time"

type Building struct {
	BuildingID   uint   `gorm:"primaryKey" json:"building_id"`
	BuildingName string `json:"building_name"`
	Location     string `json:"location"`
}

type Classroom struct {
	ClassroomID   uint   `gorm:"primaryKey" json:"classroom_id"`
	ClassroomName string `json:"classroom_name"`
	BuildingID    uint   `json:"building_id"`
	Capacity      int    `json:"capacity"`
}

type Student struct {
	StudentID   uint   `gorm:"primaryKey" json:"student_id"`
	StudentName string `json:"student_name"`
	ClassroomID uint   `json:"classroom_id"`
	FaceID      string `json:"face_id"`
	Photo       string `json:"photo"`
}

type Subject struct {
	SubjectID   uint   `gorm:"primaryKey" json:"subject_id"`
	SubjectName string `json:"subject_name"`
}

type Teacher struct {
	TeacherID   uint   `gorm:"primaryKey" json:"teacher_id"`
	TeacherName string `json:"teacher_name"`
	Subject     string `json:"subject"`
}

type ClassroomTeacher struct {
	ClassroomID uint `json:"classroom_id"`
	TeacherID   uint `json:"teacher_id"`
	SubjectID   uint `json:"subject_id"`
}

type Attendance struct {
	ID               string `gorm:"primaryKey" json:"id"`
	StudentID        string `json:"student_id"`
	ClassroomID      uint   `json:"classroom_id"`
	SubjectID        uint   `json:"subject_id"`
	Date             string `json:"date"`
	AttendanceStatus string `json:"attendance_status"`
	DetectionTime    string `json:"detection_time"`
	DeviceID         string `json:"device_id"`
}

type Schedule struct {
	UserID    string    `gorm:"primaryKey" json:"user_id"`
	Role      string    `json:"role"`  // e.g., "student" or "teacher"
	Title     string    `json:"title"` // e.g., "Math Class"
	Date      time.Time `json:"date"`  // e.g., "2025-04-10"
	Time      string    `json:"time"`  // e.g., "10:00 AM"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
