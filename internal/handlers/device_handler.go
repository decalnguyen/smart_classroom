package handlers

import (
	"fmt"
	"net/http"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

// HandleDeviceHeartbeat — ESP32/Jetson report liveness + firmware/model version.
func HandleDeviceHeartbeat(c *gin.Context) {
	var req struct {
		DeviceID     string `json:"device_id"`
		Kind         string `json:"kind"`
		ModelVersion string `json:"model_version"`
		Status       string `json:"status"`
		EventID      string `json:"event_id"` // optional
		Ts           string `json:"ts"`        // event time (RFC3339/epoch) for anti-replay
	}
	if err := c.BindJSON(&req); err != nil || req.DeviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}
	// Anti-replay: heartbeats must carry a fresh ts (event_id optional here).
	switch verifyDeviceEvent(req.EventID, req.Ts, false) {
	case eventBadTS:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu/sai 'ts' (chống phát lại)"})
		return
	case eventStale:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Heartbeat quá hạn (lệch thời gian)"})
		return
	}
	db.DB.Model(&models.DeviceCredential{}).Where("device_id = ?", req.DeviceID).Update("last_seen", time.Now())
	// Mark the matching sensor row active (heartbeat).
	db.DB.Model(&models.Sensor{}).Where("device_id = ?", req.DeviceID).
		Updates(map[string]interface{}{"timestamp": time.Now(), "status": "active"})
	c.JSON(http.StatusOK, gin.H{"message": "ok", "device_id": req.DeviceID, "server_time": nowVN().Format(time.RFC3339)})
}

// SeedDeviceCredentials issues per-device tokens (cameras + room sensor hubs).
// Idempotent. The shared DEVICE_API_KEY env is the fleet-wide fallback.
func SeedDeviceCredentials() {
	var n int64
	db.DB.Model(&models.DeviceCredential{}).Count(&n)
	if n > 0 {
		return
	}
	var classrooms []models.Classroom
	db.DB.Order("classroom_id asc").Find(&classrooms)
	creds := make([]models.DeviceCredential, 0, len(classrooms)*2)
	for _, cr := range classrooms {
		creds = append(creds,
			models.DeviceCredential{DeviceID: fmt.Sprintf("cam-%d", cr.ClassroomID), Token: fmt.Sprintf("camtok-%d", cr.ClassroomID), Kind: "camera", ClassroomID: cr.ClassroomID, Active: true},
			models.DeviceCredential{DeviceID: fmt.Sprintf("hub-%s", cr.ClassroomName), Token: fmt.Sprintf("hubtok-%d", cr.ClassroomID), Kind: "sensor", ClassroomID: cr.ClassroomID, Active: true},
		)
	}
	if len(creds) > 0 {
		db.DB.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(creds, 100)
	}
}
