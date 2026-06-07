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
	var alertsToday, sessionsToday, presentToday, enrolledToday int64

	db.DB.Model(&models.Classroom{}).Count(&classrooms)
	db.DB.Model(&models.Student{}).Count(&students)
	db.DB.Model(&models.Teacher{}).Count(&teachers)
	db.DB.Model(&models.Sensor{}).Count(&sensorsTotal)
	db.DB.Model(&models.Sensor{}).Where("status = ?", "Active").Count(&sensorsActive)
	db.DB.Model(&models.Notification{}).Where("title = ? AND created_at >= ?", "alert", startOfDay).Count(&alertsToday)
	db.DB.Model(&models.Class{}).Where("day_of_week = ?", weekday).Count(&sessionsToday)
	db.DB.Model(&models.Attendance{}).Where("date = ? AND attendance_status = ?", today, "present").Count(&presentToday)
	db.DB.Table("class_students").
		Joins("JOIN classes ON classes.class_id = class_students.class_id").
		Where("classes.day_of_week = ?", weekday).
		Count(&enrolledToday)

	rate := 0.0
	if enrolledToday > 0 {
		rate = float64(presentToday) / float64(enrolledToday)
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
			"present_today":  presentToday,
			"enrolled_today": enrolledToday,
			"rate":           rate,
		},
	})
}
