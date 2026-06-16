package handlers

import (
	"net/http"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
)

// HandleStatsOverview returns aggregate KPIs for the dashboard.
func HandleStatsOverview(c *gin.Context) {
	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)
	weekday := now.Weekday().String()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	today := now.Format("2006-01-02")

	var classrooms, students, teachers, sensorsTotal, sensorsActive int64
	var alertsToday, sessionsToday int64

	db.DB.Model(&models.Classroom{}).Count(&classrooms)
	db.DB.Model(&models.Student{}).Count(&students)
	db.DB.Model(&models.Teacher{}).Count(&teachers)
	db.DB.Model(&models.Sensor{}).Count(&sensorsTotal)
	db.DB.Model(&models.Sensor{}).Where("status = ?", "Active").Count(&sensorsActive)
	db.DB.Model(&models.Notification{}).Where("title = ? AND created_at >= ?", "alert", startOfDay).Count(&alertsToday)
	db.DB.Model(&models.Class{}).Where("day_of_week = ?", weekday).Count(&sessionsToday)

	// Attendance via the SHARED daily roll-up (student-level) so the dashboard
	// always agrees with the reports page. "Có mặt"=present, "Đi muộn"=late,
	// "Có phép"=excused, "Tỉ lệ tham gia"=(present+late)/(enrolled-excused).
	var allIDs []uint
	db.DB.Model(&models.Classroom{}).Pluck("classroom_id", &allIDs)
	roll := dailyRollup(allIDs, today, weekday)
	present, late, excused, absent, enrolled := 0, 0, 0, 0, 0
	for _, rd := range roll {
		present += rd.Present
		late += rd.Late
		excused += rd.Excused
		absent += rd.Absent
		enrolled += rd.Enrolled
	}
	rate := 0.0
	if d := enrolled - excused; d > 0 {
		rate = float64(present+late) / float64(d)
	}

	c.JSON(http.StatusOK, gin.H{
		"classrooms":     classrooms,
		"students":       students,
		"teachers":       teachers,
		"sensors_total":  sensorsTotal,
		"sensors_active": sensorsActive,
		"alerts_today":   alertsToday,
		"sessions_today": sessionsToday,
		"attendance": gin.H{
			"present_today":  present,
			"late_today":     late,
			"excused_today":  excused,
			"attended_today": present + late,
			"absent_today":   absent,
			"enrolled_today": enrolled,
			"rate":           rate,
		},
	})
}
