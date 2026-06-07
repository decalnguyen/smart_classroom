package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
)

// scopedClassroomIDs returns the classroom IDs the caller may see.
// admin → all classrooms (isAll=true); teacher → only assigned classrooms; others → none.
func scopedClassroomIDs(c *gin.Context) (ids []uint, isAll bool) {
	role := c.GetString("role")
	if role == "teacher" {
		account := c.GetString("account_id")
		var teacher models.Teacher
		if err := db.DB.Where("account_id = ?", account).First(&teacher).Error; err != nil {
			return []uint{}, false
		}
		db.DB.Model(&models.ClassroomTeacher{}).Where("teacher_id = ?", teacher.TeacherID).Pluck("classroom_id", &ids)
		return ids, false
	}
	// admin and student may view all classrooms (students read-only via UI).
	if role == "admin" || role == "student" {
		db.DB.Model(&models.Classroom{}).Pluck("classroom_id", &ids)
		return ids, true
	}
	return []uint{}, false
}

func containsUint(s []uint, v uint) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// HandleMyClassrooms returns the classrooms in the caller's scope (admin=all, teacher=assigned).
func HandleMyClassrooms(c *gin.Context) {
	ids, _ := scopedClassroomIDs(c)
	rooms := []models.Classroom{}
	if len(ids) > 0 {
		db.DB.Where("classroom_id IN ?", ids).Order("classroom_id asc").Find(&rooms)
	}
	c.JSON(http.StatusOK, rooms)
}

// HandleMyAttendance returns the caller's own attendance history (for students).
func HandleMyAttendance(c *gin.Context) {
	account := c.GetString("account_id")
	var student models.Student
	if err := db.DB.Where("account_id = ?", account).First(&student).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"linked": false, "records": []any{}, "summary": gin.H{"total": 0, "present": 0, "late": 0}})
		return
	}
	type rec struct {
		Date          string `json:"date"`
		DetectionTime string `json:"detection_time"`
		Subject       string `json:"subject"`
		Status        string `json:"status"`
		ClassroomName string `json:"classroom_name"`
	}
	var records []rec
	db.DB.Table("attendances a").
		Select("a.date, a.detection_time, COALESCE(a.subject,'') as subject, a.attendance_status as status, COALESCE(classrooms.classroom_name,'') as classroom_name").
		Joins("LEFT JOIN classrooms ON classrooms.classroom_id = a.classroom_id").
		Where("a.student_id = ?", student.StudentID).
		Order("a.date desc, a.detection_time desc").
		Limit(300).Scan(&records)

	present, late := 0, 0
	for _, r := range records {
		switch r.Status {
		case "present":
			present++
		case "late":
			late++
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"linked":  true,
		"student": gin.H{"student_id": student.StudentID, "mssv": student.MSSV, "student_name": student.StudentName},
		"summary": gin.H{"total": len(records), "present": present, "late": late},
		"records": records,
	})
}

type classroomReport struct {
	ClassroomID   uint    `json:"classroom_id"`
	ClassroomName string  `json:"classroom_name"`
	Subject       string  `json:"subject"`
	Present       int     `json:"present"`
	Late          int     `json:"late"`
	Enrolled      int     `json:"enrolled"`
	Absent        int     `json:"absent"`
	Rate          float64 `json:"rate"`
}

type datePoint struct {
	Date    string `json:"date"`
	Present int    `json:"present"`
}

// computeByClassroom builds the per-classroom present/late/absent breakdown for
// a given date across the given classroom IDs. Returns rows + totals.
func computeByClassroom(ids []uint, dateStr, weekday string) (rows []classroomReport, totPresent, totLate, totEnrolled int) {
	var classrooms []models.Classroom
	db.DB.Where("classroom_id IN ?", ids).Order("classroom_id asc").Find(&classrooms)

	var classes []models.Class
	db.DB.Where("classroom_id IN ? AND day_of_week = ?", ids, weekday).Find(&classes)
	classroomToClass := map[uint]uint{}
	classIDs := []uint{}
	for _, cl := range classes {
		if _, ok := classroomToClass[cl.ClassroomID]; !ok {
			classroomToClass[cl.ClassroomID] = cl.ClassID
			classIDs = append(classIDs, cl.ClassID)
		}
	}

	type cnt struct {
		ClassID uint
		C       int
	}
	enrolledMap, presentMap, lateMap := map[uint]int{}, map[uint]int{}, map[uint]int{}
	if len(classIDs) > 0 {
		var er []cnt
		db.DB.Table("class_students").Select("class_id, count(*) as c").
			Where("class_id IN ?", classIDs).Group("class_id").Scan(&er)
		for _, r := range er {
			enrolledMap[r.ClassID] = r.C
		}
		var sr []struct {
			ClassID uint
			Status  string
			C       int
		}
		db.DB.Table("attendances").Select("class_id, attendance_status as status, count(distinct student_id) as c").
			Where("class_id IN ? AND date = ?", classIDs, dateStr).
			Group("class_id, attendance_status").Scan(&sr)
		for _, r := range sr {
			if r.Status == "present" {
				presentMap[r.ClassID] = r.C
			} else if r.Status == "late" {
				lateMap[r.ClassID] = r.C
			}
		}
	}

	rows = make([]classroomReport, 0, len(classrooms))
	for _, cr := range classrooms {
		classID := classroomToClass[cr.ClassroomID]
		enrolled := enrolledMap[classID]
		present := presentMap[classID]
		late := lateMap[classID]
		if present+late > enrolled {
			late = enrolled - present
			if late < 0 {
				late = 0
				present = enrolled
			}
		}
		absent := enrolled - present - late
		rate := 0.0
		if enrolled > 0 {
			rate = float64(present+late) / float64(enrolled)
		}
		rows = append(rows, classroomReport{
			ClassroomID: cr.ClassroomID, ClassroomName: cr.ClassroomName, Subject: cr.Subject,
			Present: present, Late: late, Enrolled: enrolled, Absent: absent, Rate: rate,
		})
		totPresent += present
		totLate += late
		totEnrolled += enrolled
	}
	return rows, totPresent, totLate, totEnrolled
}

// HandleAttendanceReport returns attendance analytics for a date (per-classroom
// breakdown) and a date range (daily trend), scoped to the caller's role.
// Query: ?date=YYYY-MM-DD (default today), ?from=&to= (default last 7 days).
func HandleAttendanceReport(c *gin.Context) {
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)

	ids, isAll := scopedClassroomIDs(c)
	if len(ids) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"scope": c.GetString("role"), "is_all": isAll,
			"date": now.Format("2006-01-02"),
			"totals": gin.H{"present": 0, "enrolled": 0, "absent": 0, "rate": 0},
			"by_classroom": []classroomReport{}, "by_date": []datePoint{},
		})
		return
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = now.Format("2006-01-02")
	}
	dt, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date (use YYYY-MM-DD)"})
		return
	}
	weekday := dt.Weekday().String()

	from := c.Query("from")
	to := c.Query("to")
	if to == "" {
		to = now.Format("2006-01-02")
	}
	if from == "" {
		from = now.AddDate(0, 0, -6).Format("2006-01-02")
	}

	byClassroom, totPresent, totLate, totEnrolled := computeByClassroom(ids, dateStr, weekday)

	// Daily trend (present + late) across the scope.
	var trend []datePoint
	db.DB.Table("attendances").Select("date, count(*) as present").
		Where("classroom_id IN ? AND attendance_status IN ? AND date BETWEEN ? AND ?", ids, []string{"present", "late"}, from, to).
		Group("date").Order("date asc").Scan(&trend)

	totalAbsent := totEnrolled - totPresent - totLate
	totalRate := 0.0
	if totEnrolled > 0 {
		totalRate = float64(totPresent+totLate) / float64(totEnrolled)
	}

	c.JSON(http.StatusOK, gin.H{
		"scope":  c.GetString("role"),
		"is_all": isAll,
		"date":   dateStr,
		"from":   from,
		"to":     to,
		"totals": gin.H{"present": totPresent, "late": totLate, "enrolled": totEnrolled, "absent": totalAbsent, "rate": totalRate},
		"by_classroom": byClassroom,
		"by_date":      trend,
	})
}

// HandleAttendanceReportExport streams the per-classroom report for a date as CSV.
func HandleAttendanceReportExport(c *gin.Context) {
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)
	ids, _ := scopedClassroomIDs(c)
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = now.Format("2006-01-02")
	}
	dt, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date"})
		return
	}
	rows, _, _, _ := computeByClassroom(ids, dateStr, dt.Weekday().String())

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=attendance_"+dateStr+".csv")
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"Ngay", "Phong", "Mon", "Si so", "Co mat", "Di muon", "Vang", "Ti le (%)"})
	for _, r := range rows {
		_ = w.Write([]string{
			dateStr, r.ClassroomName, r.Subject,
			strconv.Itoa(r.Enrolled), strconv.Itoa(r.Present), strconv.Itoa(r.Late), strconv.Itoa(r.Absent),
			strconv.Itoa(int(r.Rate * 100)),
		})
	}
	w.Flush()
}
