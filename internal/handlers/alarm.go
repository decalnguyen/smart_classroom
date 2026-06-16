package handlers

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
	"smart_classroom/internal/rabbitmq"

	"github.com/google/uuid"
)

// Danger thresholds (env-configurable). Defaults are tuned for the demo.
func threshold(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// alertCooldown prevents alert spam: at most one alert per device every window.
var (
	alertMu       sync.Mutex
	lastAlertAt   = map[string]time.Time{}
	alertWindow   = 30 * time.Second
)

// EvaluateAndAlert inspects an incoming sensor reading and, if it breaches a
// danger threshold, raises a system-wide alert: it persists a broadcast
// notification, publishes it to the notification queue (so the WS server pushes
// it to every connected client in real time), and issues a buzzer command.
//
// This implements thesis requirement #5 (server detects danger -> trigger
// buzzer -> notify all users).
func EvaluateAndAlert(data models.SenSorData) {
	dtype := strings.ToLower(data.DeviceType)

	var (
		breached bool
		message  string
		limit    float64
	)

	switch {
	case strings.Contains(dtype, "smoke") || strings.Contains(dtype, "mq2") || strings.Contains(dtype, "gas"):
		limit = threshold("SMOKE_THRESHOLD", 300)
		if data.Value >= limit {
			breached = true
			message = fmt.Sprintf("🔥 Phát hiện khói/khí gas vượt ngưỡng tại %s: %.0f (ngưỡng %.0f)", data.DeviceID, data.Value, limit)
		}
	case strings.Contains(dtype, "temp"):
		limit = threshold("TEMP_THRESHOLD", 50)
		if data.Value >= limit {
			breached = true
			message = fmt.Sprintf("🌡️ Nhiệt độ vượt ngưỡng nguy hiểm tại %s: %.1f°C (ngưỡng %.0f°C)", data.DeviceID, data.Value, limit)
		}
	}

	if !breached {
		return
	}

	// Cooldown per device so a continuous breach doesn't flood clients.
	alertMu.Lock()
	if last, ok := lastAlertAt[data.DeviceID]; ok && time.Since(last) < alertWindow {
		alertMu.Unlock()
		return
	}
	lastAlertAt[data.DeviceID] = time.Now()
	alertMu.Unlock()

	loc := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(loc)

	notif := models.Notification{
		ID:        uuid.New().String(),
		AccountID: "ALL", // broadcast to every user
		Title:     "alert",
		Message:   message,
		IsRead:    false,
		CreatedAt: now,
	}

	// Persist (best-effort) so the alert appears in history too.
	if db.DB != nil {
		if err := db.DB.Create(&notif).Error; err != nil {
			log.Printf("Failed to persist alert notification: %v", err)
		}
	}

	// Push to all connected clients in real time via the notification queue.
	rabbitmq.Publish("notify.data", notif)

	// Issue the buzzer/alarm command toward the device layer.
	triggerBuzzer(data.DeviceID, message)
}

// triggerBuzzer issues the alarm command to the room's ESP32 buzzer via the MQTT
// command channel (/{room}/buzzer/cmd). The device subscribes and actuates.
// (Hardware fail-safe note: a life-safety buzzer should ALSO have a local trip
// on the ESP32 if the smoke pin crosses threshold, independent of the network —
// see docs/ARCHITECTURE.md.)
func triggerBuzzer(deviceID, reason string) {
	PublishDeviceCommand(roomOf(deviceID), "buzzer", "on", 1, reason)
	log.Printf("🚨 ALARM: buzzer command -> /%s/buzzer/cmd (%s)", roomOf(deviceID), reason)
}
