package handlers

import (
	"net/http"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
)

// HandleGetClassroomTeachers lists teacher↔classroom assignments (enriched with names).
func HandleGetClassroomTeachers(c *gin.Context) {
	type row struct {
		ClassroomID   uint   `json:"classroom_id"`
		ClassroomName string `json:"classroom_name"`
		TeacherID     uint   `json:"teacher_id"`
		TeacherName   string `json:"teacher_name"`
	}
	var rows []row
	db.DB.Table("classroom_teachers ct").
		Select("ct.classroom_id, classrooms.classroom_name, ct.teacher_id, teachers.teacher_name").
		Joins("JOIN classrooms ON classrooms.classroom_id = ct.classroom_id").
		Joins("JOIN teachers ON teachers.teacher_id = ct.teacher_id").
		Order("ct.classroom_id asc").
		Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// HandlePostClassroomTeacher assigns a teacher to a classroom (idempotent).
func HandlePostClassroomTeacher(c *gin.Context) {
	var req struct {
		ClassroomID uint `json:"classroom_id"`
		TeacherID   uint `json:"teacher_id"`
	}
	if err := c.BindJSON(&req); err != nil || req.ClassroomID == 0 || req.TeacherID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classroom_id and teacher_id are required"})
		return
	}
	var count int64
	db.DB.Model(&models.ClassroomTeacher{}).
		Where("classroom_id = ? AND teacher_id = ?", req.ClassroomID, req.TeacherID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Phân công đã tồn tại"})
		return
	}
	if err := db.DB.Create(&models.ClassroomTeacher{ClassroomID: req.ClassroomID, TeacherID: req.TeacherID}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã phân công"})
}

// HandleDeleteClassroomTeacher removes an assignment (?classroom_id=&teacher_id=).
func HandleDeleteClassroomTeacher(c *gin.Context) {
	classroomID := parseUintParam(c.Query("classroom_id"))
	teacherID := parseUintParam(c.Query("teacher_id"))
	if classroomID == 0 || teacherID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "classroom_id and teacher_id are required"})
		return
	}
	res := db.DB.Where("classroom_id = ? AND teacher_id = ?", classroomID, teacherID).Delete(&models.ClassroomTeacher{})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phân công"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã gỡ phân công"})
}
