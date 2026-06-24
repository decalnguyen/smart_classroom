package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
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

	// Standard daily periods (minutes from midnight): period#, start, end.
	periods = []struct{ Period, Start, End int }{
		{1, 7*60, 8*60 + 30},       // 07:00–08:30
		{2, 8*60 + 40, 10*60 + 10}, // 08:40–10:10
		{3, 10*60 + 20, 11*60 + 50},// 10:20–11:50
		{4, 13 * 60, 14*60 + 30},   // 13:00–14:30
		{5, 14*60 + 40, 16*60 + 10},// 14:40–16:10
	}
)

func vietName(i int) string {
	return fmt.Sprintf("%s %s %s", surnames[i%len(surnames)], middles[(i/3)%len(middles)], givens[(i/7)%len(givens)])
}

// SeedRealStudents upserts real students (with physical face gallery on the edge)
// and enrolls them in every class of their assigned classroom. Idempotent.
func SeedRealStudents() {
	type entry struct {
		StudentID uint
		MSSV      string
		Name      string
		Room      string // classroom_name (camera demo room)
	}
	// Students whose faces are in the edge gallery (NhanDangMSSV/models/id_map.json:
	// MSSV 22520000..22520018, 22520707, 22521491). Each is enrolled EXCLUSIVELY in
	// its camera room's all-day demo class, so the room's report shows exactly these
	// students (not "lượt" summed across mock tiết).
	real := []entry{
		{22521491, "22521491", "Nguyễn Ngô Nhật Toàn", "A101"},
		// 19 newly face-enrolled students → A101 (2 named, the rest mocked).
		{22520000, "22520000", "Trần Văn Hùng", "A101"},
		{22520001, "22520001", "Nguyễn Văn An", "A101"},
		{22520002, "22520002", "Lê Thị Hồng", "A101"},
		{22520003, "22520003", "Phạm Quốc Bảo", "A101"},
		{22520004, "22520004", "Nguyễn Minh Ánh", "A101"},
		{22520005, "22520005", "Võ Thị Lan", "A101"},
		{22520006, "22520006", "Đặng Hoàng Long", "A101"},
		{22520007, "22520007", "Bùi Thị Mai", "A101"},
		{22520008, "22520008", "Hồ Văn Nam", "A101"},
		{22520009, "22520009", "Ngô Thị Thu", "A101"},
		{22520010, "22520010", "Dương Quang Huy", "A101"},
		{22520011, "22520011", "Trịnh Thị Ngọc", "A101"},
		{22520012, "22520012", "Cao Quang Minh", "A101"},
		{22520013, "22520013", "Đỗ Văn Phúc", "A101"},
		{22520014, "22520014", "Lý Thị Hương", "A101"},
		{22520015, "22520015", "Vũ Đình Khôi", "A101"},
		{22520016, "22520016", "Phan Thị Kim", "A101"},
		{22520017, "22520017", "Trương Văn Tài", "A101"},
		{22520018, "22520018", "Mai Thị Yến", "A101"},
		// Moved to its own camera room A102.
		{22520707, "22520707", "Nguyễn Trường Anh Kiện", "A102"},
	}

	// Each camera room gets an ALL-DAY demo class EVERY weekday so the camera always
	// finds an ongoing class — even outside the regular 07:00–16:00 periods. Created
	// before the enroll loop so the face-enrolled students join them. (A101: 9001..9007,
	// A102: 9101..9107.)
	demoRooms := []struct {
		Name string
		Base uint
	}{
		{"A101", 9001},
		{"A102", 9101},
	}
	wide0 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	wide1 := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for _, dr := range demoRooms {
		var cr models.Classroom
		if db.DB.Where("classroom_name = ?", dr.Name).First(&cr).Error != nil {
			continue
		}
		demos := make([]models.Class, 0, len(days))
		for i, dow := range days {
			demos = append(demos, models.Class{
				ClassID: dr.Base + uint(i), Subject: "Demo - Trí tuệ nhân tạo",
				ClassroomID: cr.ClassroomID, SemesterID: 1, TeacherID: 1,
				Period: 9, DayOfWeek: dow, StartMin: 0, EndMin: 1439,
				StartTime: wide0, EndTime: wide1, CreatedAt: nowVN(), UpdatedAt: nowVN(),
			})
		}
		db.DB.Clauses(clause.OnConflict{DoNothing: true}).Omit("Classroom", "Students").CreateInBatches(demos, 10)
	}

	for _, e := range real {
		st := models.Student{
			StudentID:   e.StudentID,
			MSSV:        e.MSSV,
			StudentName: e.Name,
			Email:       e.MSSV + "@student.uit.edu.vn",
		}
		if err := db.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "student_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"student_name", "email"}),
		}).Omit("User", "Classes").Create(&st).Error; err != nil {
			log.Printf("SeedRealStudents: upsert %s: %v", e.MSSV, err)
			continue
		}

		var cr models.Classroom
		if err := db.DB.Where("classroom_name = ?", e.Room).First(&cr).Error; err != nil {
			log.Printf("SeedRealStudents: classroom %s not found", e.Room)
			continue
		}
		// Enroll ONLY in the room's all-day DEMO classes (period 9, span >= 600 min) —
		// not mock regular tiết NOR the short evening "Tối" class — so the room's report
		// shows these students once per day, never summed "lượt" or phantom-absent.
		var classes []models.Class
		db.DB.Where("classroom_id = ? AND period = ? AND end_min - start_min >= ?", cr.ClassroomID, 9, 600).Find(&classes)
		enrollments := make([]models.ClassStudent, 0, len(classes))
		for _, cl := range classes {
			enrollments = append(enrollments, models.ClassStudent{ClassID: cl.ClassID, StudentID: e.StudentID})
		}
		if len(enrollments) > 0 {
			if err := db.DB.Clauses(clause.OnConflict{DoNothing: true}).
				Omit("Student", "Class").CreateInBatches(enrollments, 100).Error; err != nil {
				log.Printf("SeedRealStudents: enroll %s: %v", e.MSSV, err)
			}
		}
		// Exclusive: a face-enrolled student belongs ONLY to its camera room's all-day
		// demo class. Remove every other enrollment — other rooms (handles moves, e.g.
		// Kiện A101→A102) AND same-room non-demo classes (e.g. the short "Tối" class) —
		// so reports stay clean (no "lượt", no phantom absences).
		db.DB.Exec(`DELETE FROM class_students WHERE student_id = ? AND class_id NOT IN
			(SELECT class_id FROM classes WHERE classroom_id = ? AND period = 9 AND end_min - start_min >= 600)`,
			e.StudentID, cr.ClassroomID)
		log.Printf("SeedRealStudents: %s (%s) → %s (%d demo classes)", e.Name, e.MSSV, e.Room, len(classes))
	}

	// Build ID sets: all face-enrolled students, plus those assigned to A101.
	allRealIDs := make([]uint, 0, len(real))
	roomReal := map[string][]uint{}
	for _, e := range real {
		allRealIDs = append(allRealIDs, e.StudentID)
		roomReal[e.Room] = append(roomReal[e.Room], e.StudentID)
	}

	// Camera demo: room A101 must contain ONLY its assigned face-enrolled students.
	// Strip mock enrollments + their attendance from A101 (A102 keeps its regular
	// mock classes — only Kiện was added there). Then delete orphaned mock students.
	a101IDs := roomReal["A101"]
	var a101 models.Classroom
	if err := db.DB.Where("classroom_name = ?", "A101").First(&a101).Error; err == nil && len(a101IDs) > 0 {
		var classIDs []uint
		db.DB.Model(&models.Class{}).Where("classroom_id = ?", a101.ClassroomID).Pluck("class_id", &classIDs)
		if len(classIDs) > 0 {
			db.DB.Where("class_id IN ? AND student_id NOT IN ?", classIDs, a101IDs).Delete(&models.ClassStudent{})
		}
		db.DB.Where("classroom_id = ? AND student_id NOT IN ?", a101.ClassroomID, a101IDs).Delete(&models.Attendance{})
	}
	// Delete now-orphaned mock students (protecting ALL real students) + dangling leaves.
	db.DB.Exec(`DELETE FROM students WHERE (account_id = '' OR account_id IS NULL)
		AND student_id NOT IN (?) AND student_id NOT IN (SELECT DISTINCT student_id FROM class_students)`, allRealIDs)
	db.DB.Exec(`DELETE FROM leave_requests lr WHERE NOT EXISTS (SELECT 1 FROM students s WHERE s.student_id = lr.student_id)`)
	log.Printf("SeedRealStudents: A101=%d, A102=%d face-enrolled students", len(a101IDs), len(roomReal["A102"]))
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
			name = fmt.Sprintf("B10%d", i-4)
		}
		classrooms = append(classrooms, models.Classroom{
			ClassroomID:   uint(i + 1),
			ClassroomName: name,
			Subject:       subjects[i%len(subjects)],
			BuildingID:    building,
			Capacity:      80,
			StartTime:     wideStart,
			EndTime:       wideEnd,
		})
	}
	db.DB.Clauses(clause.OnConflict{DoNothing: true}).Omit("Classes").CreateInBatches(&classrooms, 20)

	// Sensors/devices per classroom. device_type uses the CANONICAL short codes so
	// registry device_ids ("A101-temp") match telemetry ids exactly — otherwise the
	// exact-id heartbeat never refreshes temp/humi rows (see docs/DATA_MODEL.md).
	regTypes := []struct{ code, name string }{
		{"light", "Ánh sáng"}, {"temp", "Nhiệt độ"}, {"humi", "Độ ẩm"}, {"smoke", "Khói/Gas"}, {"fan", "Quạt"},
	}
	sensors := make([]models.Sensor, 0, 50)
	now := time.Now()
	for _, cr := range classrooms {
		for _, dt := range regTypes {
			sensors = append(sensors, models.Sensor{
				DeviceID:   fmt.Sprintf("%s-%s", cr.ClassroomName, dt.code),
				DeviceName: fmt.Sprintf("%s — %s", cr.ClassroomName, dt.name),
				DeviceType: dt.code,
				Location:   cr.ClassroomName,
				Status:     "active",
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

	// Active semester.
	db.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.Semester{
		ID: 1, Name: "Học kỳ 2 (2025–2026)", StartDate: "2026-01-05", EndDate: "2026-05-31", IsActive: true, CreatedAt: now,
	})
	// Fixed national holidays (attendance not processed on these dates).
	db.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&[]models.Holiday{
		{Date: "2026-01-01", Name: "Tết Dương lịch"},
		{Date: "2026-04-30", Name: "Giải phóng miền Nam"},
		{Date: "2026-05-01", Name: "Quốc tế Lao động"},
		{Date: "2026-09-02", Name: "Quốc khánh"},
	})

	// Classes: real periods (tiết) per classroom per weekday; enroll all 70 students.
	enrollments := make([]models.ClassStudent, 0, 25000)
	classCount := 0
	for ci, cr := range classrooms {
		for _, wd := range weekdays {
			for pi, p := range periods {
				class := models.Class{
					Subject:     subjects[(pi+ci)%len(subjects)],
					ClassroomID: cr.ClassroomID,
					SemesterID:  1,
					TeacherID:   uint((ci+pi)%12 + 1),
					Period:      p.Period,
					DayOfWeek:   wd,
					StartMin:    p.Start,
					EndMin:      p.End,
					StartTime:   wideStart,
					EndTime:     wideEnd,
					CreatedAt:   now,
					UpdatedAt:   now,
				}
				if err := db.DB.Omit("Classroom", "Students").Create(&class).Error; err != nil {
					log.Printf("Seed: class: %v", err)
					continue
				}
				classCount++
				for _, sid := range classroomStudents[cr.ClassroomID] {
					enrollments = append(enrollments, models.ClassStudent{ClassID: class.ClassID, StudentID: sid})
				}
			}
		}
	}
	if err := db.DB.Clauses(clause.OnConflict{DoNothing: true}).
		Omit("Student", "Class").CreateInBatches(enrollments, 500).Error; err != nil {
		log.Printf("Seed: enrollments: %v", err)
	}

	seedSchedules()

	log.Printf("Seed: mock data done — %d classrooms, %d students, %d classes, %d enrollments",
		len(classrooms), len(students), classCount, len(enrollments))
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

// SeedTodayAttendance creates a believable, student-level attendance snapshot for
// today's periods: ~80% present (all periods), ~8% late (period 1), ~5% excused
// (approved leave), ~7% absent. Daily roll-up stays consistent. Idempotent.
func SeedTodayAttendance() {
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)
	today := now.Format("2006-01-02")
	weekday := now.Weekday().String()

	if isHoliday(today) {
		return
	}
	var existing int64
	db.DB.Model(&models.Attendance{}).Where("date = ?", today).Count(&existing)
	if existing > 0 {
		return
	}
	var classes []models.Class
	db.DB.Where("day_of_week = ?", weekday).Order("classroom_id asc, start_min asc").Find(&classes)
	if len(classes) == 0 {
		return
	}
	classByID := map[uint]models.Class{}
	classIDs := make([]uint, 0, len(classes))
	for _, c := range classes {
		classByID[c.ClassID] = c
		classIDs = append(classIDs, c.ClassID)
	}

	// Each student's classes TODAY (sorted by start_min). Seed is student-centric so
	// a student gets ONE daily status applied across ALL of their own enrolled
	// classes — correct even when classes in a room have different rosters (e.g. the
	// all-day demo class), unlike the old "one roster per room" assumption.
	type csRow struct {
		ClassID   uint
		StudentID uint
	}
	var enr []csRow
	db.DB.Table("class_students").Select("class_id, student_id").Where("class_id IN ?", classIDs).Scan(&enr)
	studentClasses := map[uint][]models.Class{}
	studentMeta := map[uint]models.Student{}
	for _, e := range enr {
		studentClasses[e.StudentID] = append(studentClasses[e.StudentID], classByID[e.ClassID])
	}
	// Stable iteration order (deterministic seed) + load student profiles for leaves.
	studentIDs := make([]uint, 0, len(studentClasses))
	for sid := range studentClasses {
		studentIDs = append(studentIDs, sid)
	}
	sort.Slice(studentIDs, func(i, j int) bool { return studentIDs[i] < studentIDs[j] })
	var studs []models.Student
	db.DB.Where("student_id IN ?", studentIDs).Find(&studs)
	for _, s := range studs {
		studentMeta[s.StudentID] = s
	}

	rng := rand.New(rand.NewSource(int64(now.YearDay()) + 1000))
	rows := make([]models.Attendance, 0, 8192)
	leaves := make([]models.LeaveRequest, 0, 256)
	mkRow := func(sid uint, cl models.Class, status, tm string) {
		id := uuid.New().String()
		cid := cl.ClassID
		subj := cl.Subject
		rows = append(rows, models.Attendance{
			ID: &id, StudentID: sid, ClassroomID: cl.ClassroomID,
			ClassID: &cid, Subject: &subj, Date: today,
			AttendanceStatus: status, DetectionTime: tm, DeviceID: fmt.Sprintf("cam-%d", cl.ClassroomID),
		})
	}
	// clock formats minutes-since-midnight (+second) as HH:MM:SS without minute overflow.
	clock := func(totalMin, sec int) string {
		return fmt.Sprintf("%02d:%02d:%02d", (totalMin/60)%24, totalMin%60, sec%60)
	}

	for _, sid := range studentIDs {
		cls := studentClasses[sid]
		sort.Slice(cls, func(i, j int) bool { return cls[i].StartMin < cls[j].StartMin })
		// Draw the student's bucket ONCE for the whole day, then apply it
		// consistently to every period (80% present / 8% late / 5% excused / 7% absent).
		r := rng.Float64()
		switch {
		case r < 0.80: // present — attend every period of the day
			for _, cl := range cls {
				mkRow(sid, cl, models.StatusPresent, clock(cl.StartMin+rng.Intn(4), rng.Intn(60)))
			}
		case r < 0.88: // late to the first period, present for the rest
			for i, cl := range cls {
				if i == 0 {
					mkRow(sid, cl, models.StatusLate, clock(cl.StartMin+10+rng.Intn(20), 0))
				} else {
					mkRow(sid, cl, models.StatusPresent, clock(cl.StartMin+rng.Intn(4), rng.Intn(60)))
				}
			}
		case r < 0.93: // excused — approved leave (covers all periods, no attendance rows)
			s := studentMeta[sid]
			leaves = append(leaves, models.LeaveRequest{
				StudentID: sid, StudentName: s.StudentName, AccountID: s.AccountID,
				Date: today, Reason: "Nghỉ phép (có đơn)", Status: "approved", ReviewedBy: "system", CreatedAt: now,
			})
		default: // absent — explicit 'absent' row for every period
			for _, cl := range cls {
				id := uuid.New().String()
				cid := cl.ClassID
				subj := cl.Subject
				rows = append(rows, models.Attendance{
					ID: &id, StudentID: sid, ClassroomID: cl.ClassroomID,
					ClassID: &cid, Subject: &subj, Date: today,
					AttendanceStatus: models.StatusAbsent, DetectionTime: "", DeviceID: "seed",
				})
			}
		}
	}
	if len(rows) > 0 {
		db.DB.CreateInBatches(rows, 500)
	}
	if len(leaves) > 0 {
		db.DB.CreateInBatches(leaves, 200)
	}
	log.Printf("Seed: today attendance — %d records, %d approved leaves", len(rows), len(leaves))
}

// SeedLeaveRequests creates a realistic backlog of leave requests with MIXED
// statuses (chờ duyệt / đã duyệt / từ chối) across recent + upcoming dates, so the
// "Đơn xin nghỉ" page and the approve/reject workflow are demonstrable (not all one
// status). Idempotent: skips once any pending leave exists. Today-dated approved
// leaves are created separately by SeedTodayAttendance (attendance excused source).
func SeedLeaveRequests() {
	var pending int64
	db.DB.Model(&models.LeaveRequest{}).Where("status = ?", "pending").Count(&pending)
	if pending > 0 {
		return // already have pending leaves (seeded or real submissions)
	}
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)
	var students []models.Student
	db.DB.Order("student_id").Limit(60).Find(&students)
	if len(students) == 0 {
		return
	}
	reasons := []string{"Ốm, có giấy khám bệnh", "Việc gia đình", "Khám sức khỏe định kỳ", "Tham gia cuộc thi", "Lý do cá nhân"}
	rng := rand.New(rand.NewSource(int64(now.YearDay()) + 777))
	leaves := make([]models.LeaveRequest, 0, len(students))
	for i, s := range students {
		offset := rng.Intn(11) - 5 // -5..+5 days; avoid today (handled elsewhere)
		if offset == 0 {
			offset = 3
		}
		date := now.AddDate(0, 0, offset).Format("2006-01-02")
		r := rng.Float64()
		status, reviewedBy := "pending", ""
		var reviewedAt *time.Time
		if offset < 0 {
			// Past requests are already decided (mostly approved, some rejected).
			if r < 0.75 {
				status, reviewedBy = "approved", "teacher"
			} else {
				status, reviewedBy = "rejected", "teacher"
			}
			t := now.AddDate(0, 0, offset-1)
			reviewedAt = &t
		} else if r < 0.4 {
			// Some upcoming requests pre-approved; the rest remain pending.
			status, reviewedBy = "approved", "teacher"
			t := now
			reviewedAt = &t
		}
		leaves = append(leaves, models.LeaveRequest{
			StudentID: s.StudentID, StudentName: s.StudentName, AccountID: s.AccountID,
			Date: date, Reason: reasons[i%len(reasons)], Status: status,
			ReviewedBy: reviewedBy, ReviewedAt: reviewedAt, CreatedAt: now.AddDate(0, 0, -rng.Intn(4)),
		})
	}
	if err := db.DB.CreateInBatches(leaves, 100).Error; err != nil {
		log.Printf("SeedLeaveRequests: %v", err)
		return
	}
	log.Printf("SeedLeaveRequests: %d leave requests (mixed statuses)", len(leaves))
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
			{AccountID: u.AccountID, Role: a.Role, Title: "Mạng máy tính", Desc: "Lý thuyet", Room: "B101", Day: "Wednesday", Time: "13:00"},
			{AccountID: u.AccountID, Role: a.Role, Title: "IoT ứng dụng", Desc: "Đồ án", Room: "B102", Day: "Friday", Time: "07:30"},
			{AccountID: u.AccountID, Role: a.Role, Title: "Trí tuệ nhân tạo", Desc: "Seminar", Room: "A103", Day: "Thursday", Time: "15:00"},
		}
		db.DB.Create(&rows)
	}
}
