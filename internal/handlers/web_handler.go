package handlers

import (
	"net/http"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GET /notifications — returns the caller's own notifications plus any
// system-wide broadcasts (account_id = "ALL"), newest first.
func HandleGetNotifications(c *gin.Context) {
	accountID := c.GetString("account_id")
	if accountID == "" {
		accountID = c.Query("account_id")
	}
	var notifications []models.Notification
	if err := db.DB.
		Where("account_id = ? OR account_id = ?", accountID, "ALL").
		Order("created_at desc").
		Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}
	c.JSON(http.StatusOK, notifications)
}

// POST /notifications — create a notification. Defaults to the caller; an
// explicit account_id (query) lets admins/teachers target another user.
func HandleCreateNotification(c *gin.Context) {
	var notification models.Notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if target := c.Query("account_id"); target != "" {
		notification.AccountID = target
	} else if notification.AccountID == "" {
		notification.AccountID = c.GetString("account_id")
	}
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	if err := db.DB.Create(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}
	c.JSON(http.StatusOK, notification)
}

// PUT /notifications/:id — update a notification owned by the caller.
func HandleUpdateNotification(c *gin.Context) {
	accountID := c.GetString("account_id")
	id := c.Param("id")
	var notification models.Notification
	if err := db.DB.Where("id = ? AND (account_id = ? OR account_id = ?)", id, accountID, "ALL").
		First(&notification).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	var input models.Notification
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if input.Title != "" {
		notification.Title = input.Title
	}
	if input.Message != "" {
		notification.Message = input.Message
	}
	notification.IsRead = input.IsRead
	if err := db.DB.Save(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification"})
		return
	}
	c.JSON(http.StatusOK, notification)
}

// DELETE /notifications/:id — delete a notification owned by the caller.
func HandleDeleteNotification(c *gin.Context) {
	accountID := c.GetString("account_id")
	id := c.Param("id")
	if err := db.DB.Where("id = ? AND account_id = ?", id, accountID).
		Delete(&models.Notification{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}
