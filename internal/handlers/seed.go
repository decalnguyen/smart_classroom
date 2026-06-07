package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm/clause"
)

// insertIgnore creates records, ignoring rows that conflict on primary key
// (keeps the mock seed idempotent on a partially-populated database).
func insertIgnore(value interface{}, batch int) error {
	return db.DB.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(value, batch).Error
}

// SeedDefaults creates demo accounts (one per role) the first time the system
// boots. Existing users are never overwritten.
func SeedDefaults() {
	defaults := []struct{ Username, Password, Role string }{
		{"admin", "admin123", "admin"},
		{"teacher", "teacher123", "teacher"},
		{"student", "student123", "student"},
	}
	for _, d := range defaults {
		var existing models.User
		if err := db.DB.Where("username = ?", d.Username).First(&existing).Error; err == nil {
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(d.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Seed: hash %s: %v", d.Username, err)
			continue
		}
		user := models.User{AccountID: uuid.New().String(), Username: d.Username, Password: hash, Role: d.Role}
		if err := db.DB.Create(&user).Error; err != nil {
			log.Printf("Seed: create %s: %v", d.Username, err)
			continue
		}
		log.Printf("Seed: created default %s account (%s)", d.Role, d.Username)
	}
}

var (
	surnames = []string{"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Huỳnh", "Phan", "Vũ", "Võ", "Đặng", "Bùi", "Đỗ", "Hồ", "Ngô", "Dương", "Lý"}
	middles  = []string{"Văn", "Thị", "Hữu", "Đức", "Minh", "Thanh", "Quang", "Ngọc", "Gia", "Anh", "Bảo", "Hoài", "Khánh", "Tuấn"}
	givens   = []string{"An", "Bình", "Cường", "Dũng", "Hà", "Hải", "Hùng", "Khoa", "Lan", "Linh", "Mai", "Nam", "Ngân", "Phúc", "Quân", "Sơn", "Trang", "Tuấn", "Vy", "Yến", "Hương", "Đạt", "Long", "Thảo", "Nhi"}
	subjects = []string{"Lập trình", "Toán rời rạc", "Mạng máy tính", "Cơ sở dữ liệu", "Kiến trúc máy tính", "Hệ điều hành", "Trí tuệ nhân tạo", "IoT ứng dụng"}
)

func vietName(i int) string {
	return fmt.Sprintf("%s %s %s", surnames[i%len(surnames)], middles[(i/3)%len(middles)], givens[(i/7)%len(givens)])
}

// SeedMockData populates a realistic dataset (~10 classrooms, ~70 students each)
// the first time the system boots with an empty schema. Idempotent: it does
// nothing if classrooms already exist.
func SeedMockData() {
	var classroomCount int64
	db.DB.Model(&models.Classroom{}).Count(&classroomCount)
	if classroomCount > 0 {
		return
	}
	log.Println("Seed: generating mock data (10 classrooms x 70 students)...")
	rng := rand.New(rand.NewSource(42))
	wideStart := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	wideEnd := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	weekdays := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	// Buildings.
	buildings := []models.Building{
		{BuildingID: 1, BuildingName: "Tòa nhà A", Location: "Khu học tập phía Đông"},
		{BuildingID: 2, BuildingName: "Tòa nhà B", Location: "Khu học tập phía Tây"},
	}
	insertIgnore(&buildings, 10)

	// Subjects.
	subjectRows := make([]models.Subject, 0, len(subjects))
	for i, s := range subjects {
		subjectRows = append(subjectRows, models.Subject{SubjectID: uint(i + 1), SubjectName: s})
	}
	insertIgnore(&subjectRows, 20)

	// Teachers.
	teacherRows := make([]models.Teacher, 0, 12)
	for i := 0; i < 12; i++ {
		teacherRows = append(teacherRows, models.Teacher{
			TeacherID:   uint(i + 1),
			TeacherName: "GV. " + vietName(i*5+1),
			Subject:     subjects[i%len(subjects)],
		})
	}
	insertIgnore(&teacherRows, 20)

	// Classrooms (10): 5 per building.
	classrooms := make([]models.Classroom, 0, 10)
	for i := 0; i < 10; i++ {
		building := uint(1)
		name := fmt.Sprintf("A10%d", i+1)
		if i >= 5 {
			building = 2
			name = fmt.Sprintf("B20%d", i-4)
		}
		classrooms = append(classrooms, models.Classroom{
			ClassroomID:   uint(i + 1),
			ClassroomName: name,
			Subject:       subjects[i%len(subjects)],
			BuildingID:    building,
			StartTime:     wideStart,
			EndTime:       wideEnd,
		})
	}
	db.DB.Clauses(clause.OnConflict{DoNothing: true}).Omit("Classes").CreateInBatches(&classrooms, 20)

	// Sensors/devices per classroom.
	sensors := make([]models.Sensor, 0, 50)
	now := time.Now()
	for _, cr := range classrooms {
		for _, dt := range []string{"light", "temperature", "humidity", "smoke", "fan"} {
			sensors = append(sensors, models.Sensor{
				DeviceID:   fmt.Sprintf("%s-%s", cr.ClassroomName, dt),
				DeviceName: fmt.Sprintf("%s %s", cr.ClassroomName, dt),
				DeviceType: dt,
				Location:   cr.ClassroomName,
				Status:     "Active",
				Timestamp:  now,
			})
		}
	}
	insertIgnore(&sensors, 100)

	// Students: 70 per classroom, MSSV-style ids starting 22520001.
	students := make([]models.Student, 0, 700)
	classroomStudents := map[uint][]uint{} // classroomID -> studentIDs
	idx := 0
	for _, cr := range classrooms {
		for j := 0; j < 70; j++ {
			sid := uint(22520001 + idx)
			students = append(students, models.Student{
				StudentID:   sid,
				MSSV:        fmt.Sprintf("%d", sid),
				StudentName: vietName(idx),
				Age:         18 + rng.Intn(5),
				Phone:       fmt.Sprintf("09%08d", rng.Intn(100000000)),
				Email:       fmt.Sprintf("%d@student.uit.edu.vn", sid),
			})
			classroomStudents[cr.ClassroomID] = append(classroomStudents[cr.ClassroomID], sid)
			idx++
		}
	}
	if err := db.DB.Clauses(clause.OnConflict{DoNothing: true}).Omit("User", "Classes").CreateInBatches(students, 100).Error; err != nil {
		log.Printf("Seed: students: %v", err)
	}

	// Classes: one per classroom per weekday (wide time window so there is always
	// an ongoing class), with all 70 classroom students enrolled.
	enrollments := make([]models.ClassStudent, 0, 5000)
	for _, cr := range classrooms {
		for _, wd := range weekdays {
			class := models.Class{
				Subject:     cr.Subject,
				ClassroomID: cr.ClassroomID,
				DayOfWeek:   wd,
				StartTime:   wideStart,
				EndTime:     wideEnd,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := db.DB.Omit("Classroom", "Students").Create(&class).Error; err != nil {
				log.Printf("Seed: class: %v", err)
				continue
			}
			for _, sid := range classroomStudents[cr.ClassroomID] {
				enrollments = append(enrollments, models.ClassStudent{ClassID: class.ClassID, StudentID: sid})
			}
		}
	}
	if err := db.DB.Omit("Student", "Class").CreateInBatches(enrollments, 500).Error; err != nil {
		log.Printf("Seed: enrollments: %v", err)
	}

	seedSchedules()

	log.Printf("Seed: mock data done — %d classrooms, %d students, %d classes, %d enrollments",
		len(classrooms), len(students), len(classrooms)*len(weekdays), len(enrollments))
}

// SeedAccountLinks binds the demo teacher/student accounts to a Teacher/Student
// row so role-scoped views (own classes, own attendance) work. Idempotent.
func SeedAccountLinks() {
	if u := userByName("teacher"); u != nil {
		var t models.Teacher
		if db.DB.Where("teacher_id = ?", 1).First(&t).Error == nil && t.AccountID == "" {
			db.DB.Model(&models.Teacher{}).Where("teacher_id = ?", 1).Update("account_id", u.AccountID)
		}
	}
	if u := userByName("student"); u != nil {
		var s models.Student
		if db.DB.Order("student_id asc").First(&s).Error == nil {
			if s.AccountID == "" {
				db.DB.Model(&models.Student{}).Where("student_id = ?", s.StudentID).Update("account_id", u.AccountID)
				log.Printf("Seed: linked demo student account to %s (%d)", s.StudentName, s.StudentID)
			}
			seedStudentAttendance(s)
		}
	}
}

// seedStudentAttendance gives the demo student a short attendance history so the
// "My attendance" page isn't empty on first run. Idempotent.
func seedStudentAttendance(s models.Student) {
	var n int64
	db.DB.Model(&models.Attendance{}).Where("student_id = ?", s.StudentID).Count(&n)
	if n >= 3 { // already has a meaningful history
		return
	}
	var class models.Class
	if db.DB.Joins("JOIN class_students cs ON cs.class_id = classes.class_id").
		Where("cs.student_id = ?", s.StudentID).First(&class).Error != nil {
		return
	}
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)
	rows := []models.Attendance{}
	for i := 1; i <= 6; i++ {
		d := now.AddDate(0, 0, -i)
		status := "present"
		if i%4 == 0 {
			status = "late"
		}
		id := uuid.New().String()
		subj := class.Subject
		cid := class.ClassID
		rows = append(rows, models.Attendance{
			ID: &id, StudentID: s.StudentID, ClassroomID: class.ClassroomID,
			ClassID: &cid, Subject: &subj, Date: d.Format("2006-01-02"),
			AttendanceStatus: status, DetectionTime: "07:35:00", DeviceID: "seed",
		})
	}
	db.DB.Create(&rows)
	log.Printf("Seed: created %d attendance records for demo student", len(rows))
}

func userByName(name string) *models.User {
	var u models.User
	if err := db.DB.Where("username = ?", name).First(&u).Error; err != nil {
		return nil
	}
	return &u
}

// SeedTeacherAssignments links teachers to classrooms (ClassroomTeacher) and
// binds the demo "teacher" account to teacher #1. Idempotent. This is what lets
// a teacher see only their assigned classrooms while admins see everything.
func SeedTeacherAssignments() {
	var n int64
	db.DB.Model(&models.ClassroomTeacher{}).Count(&n)
	if n > 0 {
		return
	}
	// Bind the demo teacher account to teacher #1.
	var u models.User
	if err := db.DB.Where("username = ?", "teacher").First(&u).Error; err == nil {
		db.DB.Model(&models.Teacher{}).Where("teacher_id = ?", 1).Update("account_id", u.AccountID)
	}

	var classrooms []models.Classroom
	db.DB.Order("classroom_id asc").Find(&classrooms)
	rows := make([]models.ClassroomTeacher, 0, len(classrooms))
	for _, cr := range classrooms {
		tid := uint(1) // demo teacher teaches classrooms 1..3
		if cr.ClassroomID > 3 {
			tid = uint(2 + int(cr.ClassroomID-4)%11) // spread the rest over teachers 2..12
		}
		rows = append(rows, models.ClassroomTeacher{ClassroomID: cr.ClassroomID, TeacherID: tid})
	}
	if len(rows) > 0 {
		db.DB.Create(&rows)
		log.Printf("Seed: assigned %d classroom-teacher links (demo teacher → rooms 1-3)", len(rows))
	}
}

// seedSchedules gives the demo student & teacher accounts a sample weekly timetable.
func seedSchedules() {
	type acc struct{ Username, Role string }
	for _, a := range []acc{{"student", "student"}, {"teacher", "teacher"}} {
		var u models.User
		if err := db.DB.Where("username = ?", a.Username).First(&u).Error; err != nil {
			continue
		}
		rows := []models.Schedule{
			{AccountID: u.AccountID, Role: a.Role, Title: "Lập trình", Desc: "Phòng thực hành", Room: "A101", Day: "Monday", Time: "07:30"},
			{AccountID: u.AccountID, Role: a.Role, Title: "Cơ sở dữ liệu", Desc: "Lý thuyết", Room: "A102", Day: "Monday", Time: "09:30"},
			{AccountID: u.AccountID, Role: a.Role, Title: "Mạng máy tính", Desc: "Lý thuyet", Room: "B201", Day: "Wednesday", Time: "13:00"},
			{AccountID: u.AccountID, Role: a.Role, Title: "IoT ứng dụng", Desc: "Đồ án", Room: "B202", Day: "Friday", Time: "07:30"},
			{AccountID: u.AccountID, Role: a.Role, Title: "Trí tuệ nhân tạo", Desc: "Seminar", Room: "A103", Day: "Thursday", Time: "15:00"},
		}
		db.DB.Create(&rows)
	}
}
