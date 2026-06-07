package handlers

import (
	"fmt"
	"net/http"
	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
	"smart_classroom/internal/rabbitmq"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

// parseUintParam converts a path param to uint (0 on failure).
func parseUintParam(s string) uint {
	n, _ := strconv.ParseUint(s, 10, 64)
	return uint(n)
}

// periodTime maps a class to a display period slot (cosmetic, deterministic).
func periodTime(classID uint) string {
	slots := []string{"07:00 - 09:30", "09:45 - 12:00", "13:00 - 15:30", "15:45 - 18:00"}
	return slots[int(classID)%len(slots)]
}

func HandleGetBuildings(c *gin.Context) {
	var buildings []models.Building
	if err := db.DB.Find(&buildings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve buildings"})
		return
	}
	c.JSON(http.StatusOK, buildings)
}

func HandlePostBuilding(c *gin.Context) {
	var building models.Building
	if err := c.BindJSON(&building); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Where("building_id = ?", building.BuildingID).First(&models.Building{}).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Building already exists"})
		return
	} else if err := db.DB.Create(&building).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create building"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Building created"})
}

func HandlePutBuilding(c *gin.Context) {
	id := c.Param("id")
	var building models.Building
	if err := db.DB.Where("building_id = ?", id).First(&building).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Building not found"})
		return
	}
	if err := c.BindJSON(&building); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	building.BuildingID = parseUintParam(id) // prevent PK change via body
	if err := db.DB.Save(&building).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update building"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Building updated"})
}

func HandleDeleteBuilding(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("building_id = ?", id).Delete(&models.Building{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete building"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Building deleted"})
}

func HandleGetClassrooms(c *gin.Context) {
	var classrooms []models.Classroom
	if err := db.DB.Find(&classrooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve classrooms"})
		return
	}
	c.JSON(http.StatusOK, classrooms)
}

func HandlePostClassroom(c *gin.Context) {
	var classroom models.Classroom
	if err := c.BindJSON(&classroom); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Where("classroom_id = ?", classroom.ClassroomID).First(&models.Classroom{}).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Classroom already exists"})
		return
	} else if err := db.DB.Create(&classroom).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create classroom"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Classroom created"})
}

func HandlePutClassroom(c *gin.Context) {
	id := c.Param("id")
	var classroom models.Classroom
	if err := db.DB.Where("classroom_id = ?", id).First(&classroom).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Classroom not found"})
		return
	}
	if err := c.BindJSON(&classroom); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	classroom.ClassroomID = parseUintParam(id) // prevent PK change via body
	if err := db.DB.Save(&classroom).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update classroom"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Classroom updated"})
}

func HandleDeleteClassroom(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("classroom_id = ?", id).Delete(&models.Classroom{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete classroom"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Classroom deleted"})
}
func HandleGetClass(c *gin.Context) {
	var class models.Class
	now := time.Now()
	weekday := now.Weekday().String()
	classroomID := c.Param("id")
	if err := db.DB.Where("classroom_id = ? AND day_of_week = ? AND start_time <= ? AND end_time >= ?",
		classroomID, weekday, now, now).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No ongoing class found"})
		return
	}
	c.JSON(http.StatusOK, class)
}
func HandlePostClass(c *gin.Context) {
	var class models.Class
	if err := c.BindJSON(&class); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Where("class_id = ?", class.ClassID).First(&models.Class{}).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Class already exists"})
		return
	} else if err := db.DB.Create(&class).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create class"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Class created"})
}
func HandlePutClass(c *gin.Context) {
	id := c.Param("id")
	var class models.Class
	if err := db.DB.Where("class_id = ?", id).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}
	if err := c.BindJSON(&class); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	class.ClassID = parseUintParam(id) // prevent PK change via body
	if err := db.DB.Save(&class).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update class"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Class updated"})
}
func HandleDeleteClass(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("class_id = ?", id).Delete(&models.Class{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete class"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Class deleted"})
}
func HandleGetStudents(c *gin.Context) {
	var students []models.Student
	q := db.DB.Model(&models.Student{})

	// Optional search by name or MSSV/id.
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		q = q.Where("student_name ILIKE ? OR mssv ILIKE ? OR CAST(student_id AS TEXT) ILIKE ?", like, like, like)
	}

	var total int64
	q.Count(&total)

	// Optional pagination (limit/offset). Capped to avoid huge payloads.
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil && limit > 0 {
		if limit > 200 {
			limit = 200
		}
		offset, _ := strconv.Atoi(c.Query("offset"))
		if offset < 0 {
			offset = 0
		}
		q = q.Limit(limit).Offset(offset)
	} else {
		q = q.Limit(2000) // safety cap when no pagination requested
	}

	if err := q.Order("student_id asc").Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve students"})
		return
	}
	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	c.JSON(http.StatusOK, students)
}

func HandlePostStudent(c *gin.Context) {
	var student models.Student
	if err := c.BindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Where("student_id = ?", student.StudentID).First(&models.Student{}).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Student already exists"})
		return
	} else if err := db.DB.Create(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student created"})
}

func HandlePutStudent(c *gin.Context) {
	id := c.Param("id")
	var student models.Student
	if err := db.DB.Where("student_id = ?", id).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}
	if err := c.BindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	student.StudentID = parseUintParam(id) // prevent PK change via body
	if err := db.DB.Save(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student updated"})
}

func HandleDeleteStudent(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("student_id = ?", id).Delete(&models.Student{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student deleted"})
}
func HandleGetTeachers(c *gin.Context) {
	var teachers []models.Teacher
	if err := db.DB.Find(&teachers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve teachers"})
		return
	}
	c.JSON(http.StatusOK, teachers)
}

func HandlePostTeacher(c *gin.Context) {
	var teacher models.Teacher
	if err := c.BindJSON(&teacher); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Where("teacher_id = ?", teacher.TeacherID).First(&models.Teacher{}).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Teacher already exists"})
		return
	} else if err := db.DB.Create(&teacher).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create teacher"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Teacher created"})
}

func HandlePutTeacher(c *gin.Context) {
	id := c.Param("id")
	var teacher models.Teacher
	if err := db.DB.Where("teacher_id = ?", id).First(&teacher).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Teacher not found"})
		return
	}
	if err := c.BindJSON(&teacher); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	teacher.TeacherID = parseUintParam(id) // prevent PK change via body
	if err := db.DB.Save(&teacher).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update teacher"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Teacher updated"})
}

func HandleDeleteTeacher(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("teacher_id = ?", id).Delete(&models.Teacher{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete teacher"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Teacher deleted"})
}
func HandleGetSchedules(c *gin.Context) {
	// account_id is set by the auth middleware.
	accountID := c.GetString("account_id")
	if accountID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	weekly := map[string][]gin.H{
		"Monday": {}, "Tuesday": {}, "Wednesday": {}, "Thursday": {},
		"Friday": {}, "Saturday": {}, "Sunday": {},
	}

	// For a linked student, derive the timetable from real class enrollment.
	if c.GetString("role") == "student" {
		var student models.Student
		if err := db.DB.Where("account_id = ?", accountID).First(&student).Error; err == nil {
			var classes []models.Class
			db.DB.Joins("JOIN class_students cs ON cs.class_id = classes.class_id").
				Where("cs.student_id = ?", student.StudentID).Find(&classes)
			rooms := map[uint]string{}
			var crs []models.Classroom
			db.DB.Find(&crs)
			for _, r := range crs {
				rooms[r.ClassroomID] = r.ClassroomName
			}
			for _, cl := range classes {
				if _, ok := weekly[cl.DayOfWeek]; !ok {
					continue
				}
				weekly[cl.DayOfWeek] = append(weekly[cl.DayOfWeek], gin.H{
					"time":  periodTime(cl.ClassID),
					"title": cl.Subject,
					"room":  rooms[cl.ClassroomID],
					"desc":  "Lớp được phân công",
				})
			}
		}
	}

	// Merge any personal (free-text) schedule entries for this account.
	var schedules []models.Schedule
	db.DB.Where("account_id = ?", accountID).Find(&schedules)
	for _, s := range schedules {
		if _, ok := weekly[s.Day]; ok {
			weekly[s.Day] = append(weekly[s.Day], gin.H{"time": s.Time, "title": s.Title, "desc": s.Desc, "room": s.Room})
		}
	}

	c.JSON(http.StatusOK, weekly)
}

func HandlePostSchedule(c *gin.Context) {
	accountID := c.GetString("account_id")
	if accountID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var schedule models.Schedule
	if err := c.BindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Set the user ID for the schedule
	schedule.AccountID = accountID

	if err := db.DB.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule created"})
}

func HandlePutSchedule(c *gin.Context) {
	accountID := c.GetString("account_id")
	if accountID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var schedule models.Schedule

	// Find the schedule by ID, scoped to the owner (ownership check).
	if err := db.DB.Where("id = ? AND account_id = ?", id, accountID).First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	// Parse the updated data
	if err := c.BindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := db.DB.Save(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule updated"})
}
func HandleDeleteSchedule(c *gin.Context) {
	accountID := c.GetString("account_id")
	if accountID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	// Delete the schedule by ID, scoped to the owner (ownership check).
	if err := db.DB.Where("id = ? AND account_id = ?", id, accountID).Delete(&models.Schedule{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted"})
}

func HandleGetAttendance(c *gin.Context) {
	classroomID := c.Query("classroom_id")

	// Teachers may only view classrooms assigned to them.
	if c.GetString("role") == "teacher" {
		ids, _ := scopedClassroomIDs(c)
		if !containsUint(ids, parseUintParam(classroomID)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không được phân công lớp này"})
			return
		}
	}

	loc := time.FixedZone("UTC+7", 7*60*60) // or "Asia/Bangkok", etc.
	now := time.Now().In(loc)
	weekday := now.Weekday().String()
	// Find the class that is currently ongoing in the specified classroom.
	// NOTE: Pluck requires the model/table to be set explicitly.
	var classID uint
	db.DB.Model(&models.Class{}).
		Where("classroom_id = ? AND day_of_week = ? AND start_time <= ? AND end_time >= ?",
			classroomID, weekday, now, now).
		Limit(1).Pluck("class_id", &classID)
	if classID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No ongoing class in this classroom"})
		return
	}
	// Get the list of students enrolled in the class
	var enrolledStudents []models.Student
	if err := db.DB.
		Joins("JOIN class_students ON students.student_id = class_students.student_id").
		Where("class_students.class_id = ?", classID).
		Find(&enrolledStudents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get enrolled students"})
		return
	}
	// Map each student's attendance status today (present / late). Present wins
	// over late if both exist.
	type attRow struct {
		StudentID        uint
		AttendanceStatus string
	}
	var attRows []attRow
	db.DB.Model(&models.Attendance{}).
		Select("student_id, attendance_status").
		Where("class_id = ? AND date = ?", classID, now.Format("2006-01-02")).
		Scan(&attRows)
	statusMap := map[uint]string{}
	for _, r := range attRows {
		if statusMap[r.StudentID] == "present" {
			continue
		}
		statusMap[r.StudentID] = r.AttendanceStatus
	}

	// Privacy: only staff see contact details; students do not see peers' phone/email.
	includeContact := c.GetString("role") != "student"

	results := []gin.H{}
	for _, student := range enrolledStudents {
		status := statusMap[student.StudentID]
		if status == "" {
			status = "absent"
		}
		row := gin.H{
			"student_id":   student.StudentID,
			"mssv":         student.MSSV,
			"student_name": student.StudentName,
			"status":       status,
		}
		if includeContact {
			row["phone"] = student.Phone
			row["email"] = student.Email
		}
		results = append(results, row)
	}
	c.JSON(http.StatusOK, results)
}
func HandlePostAttendance(c *gin.Context) {
	var attendance models.Attendance

	if err := c.ShouldBindJSON(&attendance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	loc := time.FixedZone("UTC+7", 7*60*60) // Vietnam Time
	nowVN := time.Now().In(loc)             // Vietnam time
	nowUTC := nowVN.UTC()
	weekday := nowVN.Weekday().String()

	var class models.Class
	if err := db.DB.Where("classroom_id = ? AND day_of_week = ? AND start_time <= ? AND end_time >= ?",
		attendance.ClassroomID, weekday, nowUTC, nowUTC).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}
	var student models.Student
	if err := db.DB.Where("student_id = ?", attendance.StudentID).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}
	id_new := uuid.New().String()
	attendance = models.Attendance{
		ID:        &id_new,
		StudentID: attendance.StudentID,
		Student:   &student,

		ClassroomID:      attendance.ClassroomID,
		ClassID:          &class.ClassID,
		Class:            &class,
		Subject:          &class.Subject,
		Date:             nowUTC.Format("2006-01-02"),
		AttendanceStatus: attendance.AttendanceStatus,
		DetectionTime:    nowUTC.Format("15:04:05"),
		DeviceID:         attendance.DeviceID,
	}
	if err := db.DB.Create(&attendance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create attendance record"})
		return
	}
	notif := models.Notification{
		ID:        uuid.New().String(),
		Title:     "attendance",
		Message:   fmt.Sprintf("Attendance recorded for student %s in class %s", student.StudentName, class.Subject),
		AccountID: student.AccountID,
		IsRead:    false,
		CreatedAt: nowUTC,
	}
	if err := db.DB.Create(&notif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}
	rabbitmq.Publish("notify.data", notif)
	c.JSON(http.StatusOK, gin.H{"message": "Attendance record created"})
}
func HandlePutAttendance(c *gin.Context) {
	id := c.Param("id")
	var attendance models.Attendance

	// Find the attendance record by ID
	if err := db.DB.Where("id = ?", id).First(&attendance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attendance record not found"})
		return
	}

	// Parse the updated data
	if err := c.BindJSON(&attendance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := db.DB.Save(&attendance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update attendance record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Attendance record updated"})
}

func HandleDeleteAttendance(c *gin.Context) {
	id := c.Param("id")

	// Delete the attendance record by ID
	if err := db.DB.Where("id = ?", id).Delete(&models.Attendance{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attendance record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Attendance record deleted"})
}
