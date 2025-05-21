package handlers

import (
	"net/http"
	"smart_classroom/db"
	"smart_classroom/models"

	"github.com/gin-gonic/gin"
)

// GET /notifications
func HandleGetNotifications(c *gin.Context) {
	accountID := c.Query("account_id")
	println("accountID param:", accountID) // Thêm dòng này để debug
	var notifications []models.Notification
	if err := db.DB.Where("account_id = ?", accountID).Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}
	println("notifications:", notifications) // Thêm dòng này để debug
	c.JSON(http.StatusOK, notifications)
}
func HandleCreateNotification(c *gin.Context) {
	accountID := c.Query("account_id")
	var notification models.Notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	notification.AccountID = accountID
	if err := db.DB.Create(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}
	c.JSON(http.StatusOK, notification)
}

// PUT /notifications/:account_id/:id
func HandleUpdateNotification(c *gin.Context) {
	accountID := c.Param("account_id")
	id := c.Param("id")
	var notification models.Notification
	if err := db.DB.Where("account_id = ? AND id = ?", accountID, id).First(&notification).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	var input models.Notification
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	notification.Title = input.Title
	notification.Message = input.Message
	notification.IsRead = input.IsRead
	if err := db.DB.Save(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification"})
		return
	}
	c.JSON(http.StatusOK, notification)
}

// DELETE /notifications/:account_id/:id
func HandleDeleteNotification(c *gin.Context) {
	accountID := c.Param("account_id")
	id := c.Param("id")
	if err := db.DB.Where("account_id = ? AND id = ?", accountID, id).Delete(&models.Notification{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}
