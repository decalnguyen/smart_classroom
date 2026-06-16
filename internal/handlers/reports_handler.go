package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
)

// HandleClassroomsOverview returns a per-classroom snapshot (latest environment
// readings + today's attendance + safety state), scoped to the caller's role.
// Powers the "all classrooms" dashboard overview.
func HandleClassroomsOverview(c *gin.Context) {
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)
	today := now.Format("2006-01-02")
	weekday := now.Weekday().String()

	ids, _ := scopedClassroomIDs(c)
	if len(ids) == 0 {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	var classrooms []models.Classroom
	db.DB.Where("classroom_id IN ?", ids).Order("classroom_id asc").Find(&classrooms)

	buildings := map[uint]string{}
	var bs []models.Building
	db.DB.Find(&bs)
	for _, b := range bs {
		buildings[b.BuildingID] = b.BuildingName
	}

	// Today's attendance per classroom (reuses the report helper).
	att := map[uint]classroomReport{}
	rows, _, _, _, _ := computeByClassroom(ids, today, weekday)
	for _, r := range rows {
		att[r.ClassroomID] = r
	}

	// Latest reading per device in the last 30 minutes.
	type lr struct {
		DeviceID   string
		DeviceType string
		Value      float64
	}
	var lrs []lr
	db.DB.Raw(`SELECT DISTINCT ON (device_id) device_id, device_type, value
	           FROM sen_sor_data WHERE timestamp > ? ORDER BY device_id, timestamp DESC`,
		now.Add(-30*time.Minute)).Scan(&lrs)

	smokeThr := threshold("SMOKE_THRESHOLD", 300)
	tempThr := threshold("TEMP_THRESHOLD", 50)

	out := make([]gin.H, 0, len(classrooms))
	for _, cr := range classrooms {
		prefix := cr.ClassroomName + "-"
		sensors := map[string]float64{}
		for _, x := range lrs {
			if strings.HasPrefix(x.DeviceID, prefix) {
				sensors[x.DeviceType] = x.Value
			}
		}
		a := att[cr.ClassroomID]
		danger := sensors["smoke"] >= smokeThr || sensors["temperature"] >= tempThr
		out = append(out, gin.H{
			"classroom_id":   cr.ClassroomID,
			"classroom_name": cr.ClassroomName,
			"building":        buildings[cr.BuildingID],
			"sensors": gin.H{
				"light":       sensors["light"],
				"temperature": sensors["temperature"],
				"humidity":    sensors["humidity"],
				"smoke":       sensors["smoke"],
			},
			"attendance": gin.H{
				"present": a.Present, "late": a.Late, "excused": a.Excused, "absent": a.Absent,
				"enrolled": a.Enrolled, "rate": a.Rate,
			},
			"danger": danger,
		})
	}
	c.JSON(http.StatusOK, out)
}

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
	Excused       int     `json:"excused"`
	Enrolled      int     `json:"enrolled"`
	Absent        int     `json:"absent"`
	Rate          float64 `json:"rate"`
}

type datePoint struct {
	Date    string `json:"date"`
	Present int    `json:"present"`
}

// computeByClassroom builds the per-classroom daily breakdown using the shared
// student-level roll-up, so dashboard / reports / overview always agree.
func computeByClassroom(ids []uint, dateStr, weekday string) (rows []classroomReport, totPresent, totLate, totExcused, totEnrolled int) {
	roll := dailyRollup(ids, dateStr, weekday)
	var classrooms []models.Classroom
	db.DB.Where("classroom_id IN ?", ids).Order("classroom_id asc").Find(&classrooms)
	rows = make([]classroomReport, 0, len(classrooms))
	for _, cr := range classrooms {
		rd := roll[cr.ClassroomID]
		if rd == nil {
			rd = &RoomDaily{}
		}
		rows = append(rows, classroomReport{
			ClassroomID: cr.ClassroomID, ClassroomName: cr.ClassroomName, Subject: cr.Subject,
			Present: rd.Present, Late: rd.Late, Excused: rd.Excused, Enrolled: rd.Enrolled, Absent: rd.Absent, Rate: rd.Rate,
		})
		totPresent += rd.Present
		totLate += rd.Late
		totExcused += rd.Excused
		totEnrolled += rd.Enrolled
	}
	return rows, totPresent, totLate, totExcused, totEnrolled
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

	byClassroom, totPresent, totLate, totExcused, totEnrolled := computeByClassroom(ids, dateStr, weekday)

	// Daily trend: distinct students who attended (present/late) per date.
	var trend []datePoint
	db.DB.Table("attendances").Select("date, count(distinct student_id) as present").
		Where("classroom_id IN ? AND attendance_status IN ? AND date BETWEEN ? AND ?", ids, []string{"present", "late"}, from, to).
		Group("date").Order("date asc").Scan(&trend)

	totalAbsent := totEnrolled - totPresent - totLate - totExcused
	totalRate := 0.0
	if d := totEnrolled - totExcused; d > 0 {
		totalRate = float64(totPresent+totLate) / float64(d)
	}

	c.JSON(http.StatusOK, gin.H{
		"scope":  c.GetString("role"),
		"is_all": isAll,
		"date":   dateStr,
		"from":   from,
		"to":     to,
		"totals": gin.H{"present": totPresent, "late": totLate, "excused": totExcused, "enrolled": totEnrolled, "absent": totalAbsent, "rate": totalRate},
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
	rows, _, _, _, _ := computeByClassroom(ids, dateStr, dt.Weekday().String())

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=attendance_"+dateStr+".csv")
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"Ngay", "Phong", "Mon", "Si so", "Co mat", "Di muon", "Co phep", "Vang", "Ti le (%)"})
	for _, r := range rows {
		_ = w.Write([]string{
			dateStr, r.ClassroomName, r.Subject,
			strconv.Itoa(r.Enrolled), strconv.Itoa(r.Present), strconv.Itoa(r.Late), strconv.Itoa(r.Excused), strconv.Itoa(r.Absent),
			strconv.Itoa(int(r.Rate * 100)),
		})
	}
	w.Flush()
}
