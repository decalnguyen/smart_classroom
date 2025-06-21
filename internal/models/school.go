package models

import "time"

type Building struct {
	BuildingID   uint   `gorm:"primaryKey" json:"building_id"`
	BuildingName string `json:"building_name"`
	Location     string `json:"location"`
}

type Classroom struct {
	ClassroomID   uint      `gorm:"primaryKey" json:"classroom_id"`
	ClassroomName string    `json:"classroom_name"`
	Subject       string    `json:"subject"`
	BuildingID    uint      `json:"building_id"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Classes       []Class   `gorm:"foreignKey:ClassroomID"`
}
type Class struct {
	ClassID     uint      `gorm:"primaryKey" json:"class_id"`
	Subject     string    `json:"subject"`
	ClassroomID uint      `json:"classroom_id"`
	DayOfWeek   string    `json:"day_of_week"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Classroom   Classroom
	Students    []Student `gorm:"many2many:class_students;" json:"students"`
}
type ClassStudent struct {
	ID        uint `gorm:"primaryKey"`
	ClassID   uint
	StudentID uint
	Student   Student
	Class     Class
}
type Student struct {
	StudentID   uint    `gorm:"primaryKey" json:"student_id"`
	StudentName string  `json:"student_name"`
	Age         int     `json:"age"`
	Phone       string  `json:"phone"`
	Email       string  `json:"email"`
	AccountID   string  `json:"account_id"`
	User        *User   `gorm:"foreignKey:AccountID;references:AccountID"`
	Classes     []Class `gorm:"many2many:class_students;" json:"classes"`
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
	ID        *string `gorm:"primaryKey" json:"id"`
	StudentID uint    `json:"student_id" gorm:"foreignKey:StudentID"`
	Student   *Student

	ClassroomID uint  `json:"classroom_id"`
	ClassID     *uint `json:"class_id" gorm:"foreignKey:ClassID"`
	Class       *Class

	Subject          *string `json:"subject"`
	Date             string  `json:"date"`
	AttendanceStatus string  `json:"attendance_status"`
	DetectionTime    string  `json:"detection_time"`
	DeviceID         string  `json:"device_id"`
}

type Schedule struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AccountID string    `json:"account_id"`
	Role      string    `json:"role"`  // e.g., "student" or "teacher"
	Title     string    `json:"title"` // e.g., "Math Class"
	Desc      string    `json:"desc"`
	Room      string    `json:"room"`
	Day       string    `json:"day"`  // e.g., "Monday"
	Time      string    `json:"time"` // e.g., "10:00 AM"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
