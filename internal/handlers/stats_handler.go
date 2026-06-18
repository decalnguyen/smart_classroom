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

	// Scope the KPIs to the caller: admin = whole school; teacher = rooms they
	// teach; student = rooms of enrolled classes. So a GV/SV dashboard reflects
	// only their own context, not the entire institution.
	roomSet, _, isAll := actorRoomScope(c)
	var scopeIDs []uint
	names := roomNames(roomSet)
	if !isAll {
		if len(roomSet) > 0 {
			db.DB.Model(&models.Classroom{}).Where("classroom_name IN ?", names).Pluck("classroom_id", &scopeIDs)
		}
	}

	if isAll {
		db.DB.Model(&models.Classroom{}).Count(&classrooms)
		db.DB.Model(&models.Student{}).Count(&students)
		db.DB.Model(&models.Teacher{}).Count(&teachers)
		db.DB.Model(&models.Sensor{}).Count(&sensorsTotal)
		db.DB.Model(&models.Sensor{}).Where("lower(status) = ?", "active").Count(&sensorsActive)
		db.DB.Model(&models.Class{}).Where("day_of_week = ?", weekday).Count(&sessionsToday)
	} else if len(scopeIDs) > 0 {
		classrooms = int64(len(scopeIDs))
		db.DB.Table("class_students cs").Joins("JOIN classes c ON c.class_id = cs.class_id").
			Where("c.classroom_id IN ?", scopeIDs).Distinct("cs.student_id").Count(&students)
		db.DB.Model(&models.Class{}).Where("classroom_id IN ?", scopeIDs).Distinct("teacher_id").Count(&teachers)
		db.DB.Model(&models.Sensor{}).Where("location IN ?", names).Count(&sensorsTotal)
		db.DB.Model(&models.Sensor{}).Where("location IN ? AND lower(status) = ?", names, "active").Count(&sensorsActive)
		db.DB.Model(&models.Class{}).Where("day_of_week = ? AND classroom_id IN ?", weekday, scopeIDs).Count(&sessionsToday)
	}
	// Safety alerts are system-wide (everyone should see them).
	db.DB.Model(&models.Notification{}).Where("title = ? AND created_at >= ?", "alert", startOfDay).Count(&alertsToday)

	// Attendance via the SHARED daily roll-up (student-level) so the dashboard
	// always agrees with the reports page. "Có mặt"=present, "Đi muộn"=late,
	// "Có phép"=excused, "Tỉ lệ tham gia"=(present+late)/(enrolled-excused).
	var rollIDs []uint
	if isAll {
		db.DB.Model(&models.Classroom{}).Pluck("classroom_id", &rollIDs)
	} else {
		rollIDs = scopeIDs
	}
	roll := dailyRollup(rollIDs, today, weekday)
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
