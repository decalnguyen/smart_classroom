package models

import "time"

type User struct {
	AccountID string    `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"unique" json:"username"`
	Password  []byte    `json:"-"`
	Role      string    `json:"role"`       // e.g., "admin", "teacher", "student"
	StudentID uint      `json:"student_id"` // Foreign key to Student
	Student   *Student  `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
