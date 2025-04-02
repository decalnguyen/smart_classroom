package models

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
	SubjectID   uint   `json:"subject_id"`
}

type ClassroomTeacher struct {
	ClassroomID uint `json:"classroom_id"`
	TeacherID   uint `json:"teacher_id"`
	SubjectID   uint `json:"subject_id"`
}

type Attendance struct {
	ID               int    `gorm:"primaryKey" json:"id"`
	StudentID        uint   `json:"student_id"`
	ClassroomID      uint   `json:"classroom_id"`
	SubjectID        uint   `json:"subject_id"`
	Date             string `json:"date"`
	AttendanceStatus string `json:"attendance_status"`
	DetectionTime    string `json:"detection_time"`
	DeviceID         string `json:"device_id"`
}
