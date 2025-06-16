package models

import "time"

type User struct {
	AccountID string    `gorm:"primaryKey" json:"account_id"`
	Username  string    `gorm:"unique" json:"username"`
	Password  []byte    `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
type UserProfile struct {
	AccountID  string     `gorm:"primaryKey" json:"account_id"` // Foreign key to User.AccountID
	FirstName  string     `json:"first_name"`
	LastName   string     `json:"last_name"`
	Email      string     `json:"email"`
	Phone      string     `json:"phone"`
	Address    string     `json:"address"`
	ProfilePic string     `json:"profile_pic"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"` // Soft delete
}
type Notification struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	AccountID string    `json:"account_id"` // Foreign key to User.AccountID
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}
