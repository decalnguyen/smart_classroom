package models

import "time"

type User struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"unique" json:"username"`
	Password  []byte    `json:"-"`
	Role      string    `json:"role"` // e.g., "admin", "teacher", "student"
	CreatedAt time.Time `json:"created_at"`
}
