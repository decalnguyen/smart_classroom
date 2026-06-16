package handlers

import (
	"net/http"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
)

// HandleCreateLeave — a student (or staff on their behalf) submits a leave request.
func HandleCreateLeave(c *gin.Context) {
	var req struct {
		StudentID uint   `json:"student_id"`
		Date      string `json:"date"`
		Reason    string `json:"reason"`
	}
	if err := c.BindJSON(&req); err != nil || req.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required"})
		return
	}
	account := c.GetString("account_id")

	var student models.Student
	if c.GetString("role") == "student" {
		if err := db.DB.Where("account_id = ?", account).First(&student).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tài khoản chưa liên kết hồ sơ học sinh"})
			return
		}
	} else {
		if err := db.DB.Where("student_id = ?", req.StudentID).First(&student).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy học sinh"})
			return
		}
	}

	lr := models.LeaveRequest{
		StudentID: student.StudentID, StudentName: student.StudentName, AccountID: account,
		Date: req.Date, Reason: req.Reason, Status: "pending", CreatedAt: nowVN(),
	}
	if err := db.DB.Create(&lr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tạo được đơn"})
		return
	}
	c.JSON(http.StatusOK, lr)
}

// HandleListLeaves — students see their own; staff see all.
func HandleListLeaves(c *gin.Context) {
	q := db.DB.Model(&models.LeaveRequest{}).Order("created_at desc")
	if c.GetString("role") == "student" {
		q = q.Where("account_id = ?", c.GetString("account_id"))
	} else if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	var rows []models.LeaveRequest
	q.Limit(500).Find(&rows)
	c.JSON(http.StatusOK, rows)
}

// HandleReviewLeave — staff approve/reject. Approved leaves become "excused" in
// the attendance roll-up (no attendance row needed).
func HandleReviewLeave(c *gin.Context) {
	id := parseUintParam(c.Param("id"))
	var req struct {
		Status string `json:"status"` // approved | rejected
	}
	if err := c.BindJSON(&req); err != nil || (req.Status != "approved" && req.Status != "rejected") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be approved or rejected"})
		return
	}
	var lr models.LeaveRequest
	if err := db.DB.First(&lr, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đơn"})
		return
	}
	now := nowVN()
	db.DB.Model(&lr).Updates(map[string]interface{}{
		"status": req.Status, "reviewed_by": c.GetString("account_id"), "reviewed_at": now,
	})
	writeAudit(c, req.Status, "leave_request", uintStr(lr.ID),
		"Đơn nghỉ của SV "+lr.StudentName+" ("+lr.Date+")")
	c.JSON(http.StatusOK, gin.H{"message": "Đã xử lý đơn", "status": req.Status})
}
