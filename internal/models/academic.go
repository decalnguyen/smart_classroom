package models

import "time"

// Attendance status values.
const (
	StatusPresent = "present"
	StatusLate    = "late"
	StatusExcused = "excused" // vắng có phép (approved leave)
	StatusAbsent  = "absent"  // vắng không phép
)

// Semester / academic term.
type Semester struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	StartDate string    `json:"start_date"` // YYYY-MM-DD
	EndDate   string    `json:"end_date"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Holiday — attendance is not processed on these dates.
type Holiday struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Date string `gorm:"index" json:"date"` // YYYY-MM-DD
	Name string `json:"name"`
}

// MakeupSession — an extra class session held on a specific date (buổi bù).
type MakeupSession struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	ClassID  uint   `json:"class_id"`
	Date     string `gorm:"index" json:"date"` // YYYY-MM-DD
	StartMin int    `json:"start_min"`
	EndMin   int    `json:"end_min"`
	Note     string `json:"note"`
}

// LeaveRequest — a student's request to be excused for a date.
type LeaveRequest struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	StudentID   uint       `gorm:"index" json:"student_id"`
	StudentName string     `json:"student_name"`
	AccountID   string     `gorm:"index" json:"account_id"` // requester account
	Date        string     `json:"date"`                    // YYYY-MM-DD requested off
	Reason      string     `json:"reason"`
	Status      string     `gorm:"index" json:"status"` // pending | approved | rejected
	ReviewedBy  string     `json:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// FaceReview — a low-confidence recognition awaiting human confirmation.
type FaceReview struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	StudentID     uint      `json:"student_id"`
	MSSV          string    `json:"mssv"`
	StudentName   string    `json:"student_name"`
	ClassroomID   uint      `json:"classroom_id"`
	ClassID       uint      `json:"class_id"`
	Subject       string    `json:"subject"`
	Confidence    float64   `json:"confidence"`
	Date          string    `json:"date"`
	DetectionTime string    `json:"detection_time"`
	DeviceID      string    `json:"device_id"`
	Status        string    `gorm:"index" json:"status"` // pending | confirmed | rejected
	CreatedAt     time.Time `json:"created_at"`
}

// AuditLog — immutable record of sensitive actions (who/what/when).
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ActorID    string    `gorm:"index" json:"actor_id"`
	ActorName  string    `json:"actor_name"`
	ActorRole  string    `json:"actor_role"`
	Action     string    `json:"action"`           // create | update | delete | approve | reject
	Entity     string    `gorm:"index" json:"entity"` // attendance | leave_request | ...
	EntityID   string    `json:"entity_id"`
	Detail     string    `json:"detail"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}
