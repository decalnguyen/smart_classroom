package models

import "time"

type User struct {
	AccountID string    `gorm:"primaryKey" json:"account_id"`
	Username  string    `gorm:"unique" json:"username"`
	Password  []byte    `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
