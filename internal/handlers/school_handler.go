package handlers

import (
	"fmt"
	"net/http"
	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
	"strconv"

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
	writeAudit(c, "create", "building", uintStr(building.BuildingID), building.BuildingName)
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
	writeAudit(c, "update", "building", id, building.BuildingName)
	c.JSON(http.StatusOK, gin.H{"message": "Building updated"})
}

func HandleDeleteBuilding(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("building_id = ?", id).Delete(&models.Building{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete building"})
		return
	}
	writeAudit(c, "delete", "building", id, "")
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
	writeAudit(c, "create", "classroom", uintStr(classroom.ClassroomID), classroom.ClassroomName)
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
	writeAudit(c, "update", "classroom", id, classroom.ClassroomName)
	c.JSON(http.StatusOK, gin.H{"message": "Classroom updated"})
}

func HandleDeleteClassroom(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("classroom_id = ?", id).Delete(&models.Classroom{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete classroom"})
		return
	}
	writeAudit(c, "delete", "classroom", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "Classroom deleted"})
}
func HandleGetClass(c *gin.Context) {
	cl, ok := findOngoingClass(parseUintParam(c.Param("id")))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không có tiết học đang diễn ra"})
		return
	}
	c.JSON(http.StatusOK, cl)
}
func HandlePostClass(c *gin.Context) {
	var class models.Class
	if err := c.BindJSON(&class); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if msg := classConflict(class.ClassroomID, class.TeacherID, class.DayOfWeek, class.StartMin, class.EndMin, 0); msg != "" {
		c.JSON(http.StatusConflict, gin.H{"error": msg})
		return
	}
	if err := db.DB.Omit("Classroom", "Students").Create(&class).Error; err != nil {
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
	if msg := classConflict(class.ClassroomID, class.TeacherID, class.DayOfWeek, class.StartMin, class.EndMin, class.ClassID); msg != "" {
		c.JSON(http.StatusConflict, gin.H{"error": msg})
		return
	}
	if err := db.DB.Omit("Classroom", "Students").Save(&class).Error; err != nil {
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
	writeAudit(c, "create", "student", uintStr(student.StudentID), student.MSSV+" "+student.StudentName)
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
	writeAudit(c, "update", "student", id, student.MSSV+" "+student.StudentName)
	c.JSON(http.StatusOK, gin.H{"message": "Student updated"})
}

func HandleDeleteStudent(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("student_id = ?", id).Delete(&models.Student{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete student"})
		return
	}
	writeAudit(c, "delete", "student", id, "")
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
	writeAudit(c, "create", "teacher", uintStr(teacher.TeacherID), teacher.TeacherName)
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
	writeAudit(c, "update", "teacher", id, teacher.TeacherName)
	c.JSON(http.StatusOK, gin.H{"message": "Teacher updated"})
}

func HandleDeleteTeacher(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("teacher_id = ?", id).Delete(&models.Teacher{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete teacher"})
		return
	}
	writeAudit(c, "delete", "teacher", id, "")
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

	// For an admin, show the ROOM-USAGE timetable: every classroom's weekly
	// schedule (môn / tiết / phòng / GV) so they see how rooms are used. This is
	// the institution-wide view, not a personal one.
	if c.GetString("role") == "admin" {
		var classes []models.Class
		db.DB.Order("start_min asc").Find(&classes)
		rooms := map[uint]string{}
		var crs []models.Classroom
		db.DB.Find(&crs)
		for _, r := range crs {
			rooms[r.ClassroomID] = r.ClassroomName
		}
		teachers := map[uint]string{}
		var ts []models.Teacher
		db.DB.Find(&ts)
		for _, t := range ts {
			teachers[t.TeacherID] = t.TeacherName
		}
		for _, cl := range classes {
			if _, ok := weekly[cl.DayOfWeek]; !ok {
				continue
			}
			// Skip synthetic all-day classes (e.g. the demo class) — not a real period.
			if cl.EndMin-cl.StartMin >= 600 {
				continue
			}
			desc := fmt.Sprintf("Tiết %d", cl.Period)
			if tn := teachers[cl.TeacherID]; tn != "" {
				desc += " · " + tn
			}
			weekly[cl.DayOfWeek] = append(weekly[cl.DayOfWeek], gin.H{
				"time":     fmt.Sprintf("%02d:%02d - %02d:%02d", cl.StartMin/60, cl.StartMin%60, cl.EndMin/60, cl.EndMin%60),
				"title":    cl.Subject,
				"room":     rooms[cl.ClassroomID],
				"desc":     desc,
				"editable": false, // managed via the Quản trị (classes) page
			})
		}
		c.JSON(http.StatusOK, weekly)
		return
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
				if cl.EndMin-cl.StartMin >= 600 { // skip synthetic all-day demo class
					continue
				}
				weekly[cl.DayOfWeek] = append(weekly[cl.DayOfWeek], gin.H{
					"time":     fmt.Sprintf("%02d:%02d - %02d:%02d", cl.StartMin/60, cl.StartMin%60, cl.EndMin/60, cl.EndMin%60),
					"title":    cl.Subject,
					"room":     rooms[cl.ClassroomID],
					"desc":     fmt.Sprintf("Tiết %d", cl.Period),
					"editable": false, // class-derived: read-only (managed via timetable)
				})
			}
		}
	}

	// For a teacher, derive the weekly TEACHING timetable from the classes they teach.
	if c.GetString("role") == "teacher" {
		var teacher models.Teacher
		if err := db.DB.Where("account_id = ?", accountID).First(&teacher).Error; err == nil {
			var classes []models.Class
			db.DB.Where("teacher_id = ?", teacher.TeacherID).Find(&classes)
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
				if cl.EndMin-cl.StartMin >= 600 { // skip synthetic all-day demo class
					continue
				}
				weekly[cl.DayOfWeek] = append(weekly[cl.DayOfWeek], gin.H{
					"time":     fmt.Sprintf("%02d:%02d - %02d:%02d", cl.StartMin/60, cl.StartMin%60, cl.EndMin/60, cl.EndMin%60),
					"title":    cl.Subject,
					"room":     rooms[cl.ClassroomID],
					"desc":     fmt.Sprintf("Tiết %d", cl.Period),
					"editable": false, // teaching schedule is managed via the timetable
				})
			}
		}
	}

	// Merge any personal (free-text) schedule entries for this account.
	var schedules []models.Schedule
	db.DB.Where("account_id = ?", accountID).Find(&schedules)
	for _, s := range schedules {
		if _, ok := weekly[s.Day]; ok {
			weekly[s.Day] = append(weekly[s.Day], gin.H{
				"id": s.ID, "time": s.Time, "title": s.Title, "desc": s.Desc, "room": s.Room, "editable": true,
			})
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
	origID := schedule.ID

	// Parse the updated data
	if err := c.BindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	// Re-pin identity + owner so a malicious body can't move the row or reassign ownership.
	schedule.ID = origID
	schedule.AccountID = accountID

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

	// Current ongoing period for this classroom (respects holidays/makeups).
	class, ok := findOngoingClass(parseUintParam(classroomID))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không có tiết học đang diễn ra trong phòng này"})
		return
	}
	today := nowVN().Format("2006-01-02")

	// Distinct enrolled student IDs first, then load them — so a student appears
	// exactly once even if the join table has stray duplicate rows.
	var enrolledIDs []uint
	db.DB.Model(&models.ClassStudent{}).
		Where("class_id = ?", class.ClassID).
		Distinct("student_id").
		Pluck("student_id", &enrolledIDs)
	var enrolledStudents []models.Student
	if len(enrolledIDs) > 0 {
		if err := db.DB.Where("student_id IN ?", enrolledIDs).
			Order("student_name asc").
			Find(&enrolledStudents).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get enrolled students"})
			return
		}
	}

	// Status of each enrolled student in THIS period today (present/late).
	type attRow struct {
		ID               *string
		StudentID        uint
		AttendanceStatus string
	}
	var attRows []attRow
	db.DB.Model(&models.Attendance{}).Select("id, student_id, attendance_status").
		Where("class_id = ? AND date = ?", class.ClassID, today).Scan(&attRows)
	statusMap := map[uint]string{}
	idMap := map[uint]string{} // record id of the row whose status we keep -> enables edit/delete
	for _, r := range attRows {
		if statusMap[r.StudentID] == models.StatusPresent {
			continue
		}
		statusMap[r.StudentID] = r.AttendanceStatus
		if r.ID != nil {
			idMap[r.StudentID] = *r.ID
		}
	}
	// Approved leave today -> excused.
	var leaveIDs []uint
	db.DB.Model(&models.LeaveRequest{}).Where("date = ? AND status = ?", today, "approved").Pluck("student_id", &leaveIDs)
	excused := map[uint]bool{}
	for _, id := range leaveIDs {
		excused[id] = true
	}

	includeContact := c.GetString("role") != "student" // privacy

	results := []gin.H{}
	for _, student := range enrolledStudents {
		status := statusMap[student.StudentID]
		if status == "" {
			if excused[student.StudentID] {
				status = models.StatusExcused
			} else {
				status = models.StatusAbsent
			}
		}
		row := gin.H{"student_id": student.StudentID, "mssv": student.MSSV, "student_name": student.StudentName, "status": status}
		if aid, ok := idMap[student.StudentID]; ok {
			row["id"] = aid // present only when a real record exists -> UI shows edit/delete
		}
		if includeContact {
			row["phone"] = student.Phone
			row["email"] = student.Email
		}
		results = append(results, row)
	}
	c.JSON(http.StatusOK, gin.H{"class": gin.H{
		"period":    class.Period,
		"subject":   class.Subject,
		"start_min": class.StartMin,
		"end_min":   class.EndMin,
		"time":      fmt.Sprintf("%02d:%02d–%02d:%02d", class.StartMin/60, class.StartMin%60, class.EndMin/60, class.EndMin%60),
	}, "students": results})
}
func HandlePostAttendance(c *gin.Context) {
	var attendance models.Attendance

	if err := c.ShouldBindJSON(&attendance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	class, ok := findOngoingClass(attendance.ClassroomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không có tiết học đang diễn ra trong phòng này"})
		return
	}
	var student models.Student
	if err := db.DB.Where("student_id = ?", attendance.StudentID).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}
	// May only mark students enrolled in the class currently in session in this room.
	var enrolledCount int64
	db.DB.Model(&models.ClassStudent{}).
		Where("class_id = ? AND student_id = ?", class.ClassID, student.StudentID).
		Count(&enrolledCount)
	if enrolledCount == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Sinh viên %s không thuộc lớp %s đang học tại phòng này", student.StudentName, class.Subject)})
		return
	}
	now := nowVN()
	dateStr := now.Format("2006-01-02")
	status := attendance.AttendanceStatus
	if status == "" {
		status = models.StatusPresent
	}

	// Dedup: one row per (student, class, date) — update status (with audit) instead of duplicating.
	var dup models.Attendance
	if db.DB.Where("student_id = ? AND class_id = ? AND date = ?", student.StudentID, class.ClassID, dateStr).
		First(&dup).Error == nil {
		old := dup.AttendanceStatus
		db.DB.Model(&dup).Update("attendance_status", status)
		writeAudit(c, "update", "attendance", deref(dup.ID), fmt.Sprintf("SV %d (%s→%s) môn %s", student.StudentID, old, status, class.Subject))
		c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật điểm danh"})
		return
	}
	id := uuid.New().String()
	device := attendance.DeviceID
	if device == "" {
		device = "manual"
	}
	att := models.Attendance{
		ID: &id, StudentID: student.StudentID, ClassroomID: attendance.ClassroomID,
		ClassID: &class.ClassID, Subject: &class.Subject, Date: dateStr,
		AttendanceStatus: status, DetectionTime: now.Format("15:04:05"), DeviceID: device,
	}
	if err := db.DB.Create(&att).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create attendance record"})
		return
	}
	writeAudit(c, "create", "attendance", id, fmt.Sprintf("SV %d điểm danh '%s' môn %s", student.StudentID, status, class.Subject))
	c.JSON(http.StatusOK, gin.H{"message": "Đã ghi nhận điểm danh"})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func HandlePutAttendance(c *gin.Context) {
	id := c.Param("id")
	var attendance models.Attendance

	// Find the attendance record by ID
	if err := db.DB.Where("id = ?", id).First(&attendance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attendance record not found"})
		return
	}

	// A teacher may only edit records in classrooms assigned to them.
	if !canManageClassroom(c, attendance.ClassroomID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không được phân công lớp này"})
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

	writeAudit(c, "update", "attendance", id, "trạng thái -> "+attendance.AttendanceStatus)
	c.JSON(http.StatusOK, gin.H{"message": "Attendance record updated"})
}

func HandleDeleteAttendance(c *gin.Context) {
	id := c.Param("id")

	// Load first so we can scope the delete to a teacher's assigned classrooms.
	var attendance models.Attendance
	if err := db.DB.Where("id = ?", id).First(&attendance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attendance record not found"})
		return
	}
	if !canManageClassroom(c, attendance.ClassroomID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không được phân công lớp này"})
		return
	}

	if err := db.DB.Where("id = ?", id).Delete(&models.Attendance{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attendance record"})
		return
	}

	writeAudit(c, "delete", "attendance", id, "xoá bản ghi điểm danh")
	c.JSON(http.StatusOK, gin.H{"message": "Attendance record deleted"})
}

// canManageClassroom returns true unless the caller is a teacher who is not
// assigned to the given classroom (admins always pass).
func canManageClassroom(c *gin.Context, classroomID uint) bool {
	if c.GetString("role") != "teacher" {
		return true
	}
	ids, _ := scopedClassroomIDs(c)
	return containsUint(ids, classroomID)
}
