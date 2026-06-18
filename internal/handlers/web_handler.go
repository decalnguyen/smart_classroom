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
// system-wide broadcasts (account_id = "ALL"), newest first. For broadcasts, the
// caller's PER-USER state (read/dismissed) is applied so it never bleeds across users.
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
	// Per-user state for broadcast rows: apply read flag, drop dismissed ones.
	var states []models.NotificationState
	db.DB.Where("account_id = ?", accountID).Find(&states)
	stateByNotif := map[string]models.NotificationState{}
	for _, s := range states {
		stateByNotif[s.NotificationID] = s
	}
	out := make([]models.Notification, 0, len(notifications))
	for _, n := range notifications {
		if n.AccountID == "ALL" {
			st, ok := stateByNotif[n.ID]
			if ok {
				if st.Dismissed {
					continue // this user dismissed the broadcast
				}
				n.IsRead = st.Read
			} else {
				n.IsRead = false // unseen by this user
			}
		}
		out = append(out, n)
	}
	c.JSON(http.StatusOK, out)
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

// PUT /notifications/:id — mark read/update. For a broadcast ("ALL") row, the
// read state is stored PER USER (NotificationState) so it doesn't affect others;
// for a personal row the state lives on the row itself.
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
	if notification.AccountID == "ALL" {
		// Per-user read state for the broadcast (do NOT mutate the shared row).
		upsertNotifState(accountID, id, func(s *models.NotificationState) { s.Read = input.IsRead })
		notification.IsRead = input.IsRead
		c.JSON(http.StatusOK, notification)
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

// DELETE /notifications/:id — remove a personal notification, or per-user DISMISS
// a broadcast ("ALL") row so it disappears only for the caller (not everyone).
func HandleDeleteNotification(c *gin.Context) {
	accountID := c.GetString("account_id")
	id := c.Param("id")

	var notification models.Notification
	if err := db.DB.Where("id = ?", id).First(&notification).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"}) // already gone
		return
	}
	if notification.AccountID == "ALL" {
		upsertNotifState(accountID, id, func(s *models.NotificationState) { s.Dismissed = true })
		c.JSON(http.StatusOK, gin.H{"message": "Notification dismissed"})
		return
	}
	if notification.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không có quyền"})
		return
	}
	if err := db.DB.Where("id = ? AND account_id = ?", id, accountID).
		Delete(&models.Notification{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}

// upsertNotifState creates/updates a user's per-broadcast state via the unique
// (account_id, notification_id) index, applying mutate to set Read/Dismissed.
func upsertNotifState(accountID, notifID string, mutate func(*models.NotificationState)) {
	var st models.NotificationState
	db.DB.Where("account_id = ? AND notification_id = ?", accountID, notifID).First(&st)
	st.AccountID = accountID
	st.NotificationID = notifID
	mutate(&st)
	db.DB.Save(&st)
}
