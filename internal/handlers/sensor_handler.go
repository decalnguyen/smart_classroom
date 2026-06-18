package handlers

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"fmt"
	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
	"smart_classroom/internal/rabbitmq"

	"github.com/gin-gonic/gin"
)

// deviceIdent restricts device type/id to a safe charset to avoid SSRF/URL injection.
var deviceIdent = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,128}$`)

func SensorChecker() {
	go func() {
		for {
			CheckSensorStatus()
			time.Sleep(10 * time.Second)
		}
	}()
}

// SensorRetentionChecker prunes raw sensor readings older than SENSOR_RETENTION_DAYS
// (default 7) so the time-series table stays bounded. Production should use a
// TimescaleDB hypertable + continuous aggregates; this is the minimal policy.
func SensorRetentionChecker() {
	days := envInt("SENSOR_RETENTION_DAYS", 7)
	if days <= 0 {
		return
	}
	go func() {
		for {
			cutoff := nowVN().AddDate(0, 0, -days)
			res := db.DB.Where("timestamp < ?", cutoff).Delete(&models.SenSorData{})
			if res.RowsAffected > 0 {
				log.Printf("Retention: pruned %d sensor rows older than %d days", res.RowsAffected, days)
			}
			time.Sleep(time.Hour)
		}
	}()
}
func CheckSensorStatus() {
	// Inactivity auto-downgrade is opt-in (SENSOR_INACTIVE_MINUTES). Disabled by
	// default so registered devices remain "Active" unless explicitly enabled.
	mins := 0
	if v := os.Getenv("SENSOR_INACTIVE_MINUTES"); v != "" {
		mins, _ = strconv.Atoi(v)
	}
	if mins <= 0 {
		return
	}
	threshold := time.Duration(mins) * time.Minute

	var sensors []models.Sensor
	if err := db.DB.Where("lower(status) = ?", "active").Find(&sensors).Error; err != nil {
		log.Printf("Error fetching sensors: %v", err)
		return
	}

	for _, sensor := range sensors {
		if time.Since(sensor.Timestamp) > threshold {
			// Update the sensor's status to "inactive" in the Sensor table
			if err := db.DB.Model(&sensor).Where("lower(status) = ?", "active").Update("status", "inactive").Error; err != nil {
				log.Printf("Error updating sensor status for device_id %s: %v", sensor.DeviceID, err)
			} else {
				log.Printf("Sensor %s marked as inactive in Sensor table", sensor.DeviceID)

				// Update the corresponding sensor's status in the SenSorData table
				if err := db.DB.Model(&models.SenSorData{}).Where("device_id = ?", sensor.DeviceID).Update("status", "inactive").Error; err != nil {
					log.Printf("Error updating sensor data status for device_id %s: %v", sensor.DeviceID, err)
				} else {
					log.Printf("Sensor %s marked as inactive in SenSorData table", sensor.DeviceID)
				}
			}
		}
	}
}
func HandlePostSensorData(c *gin.Context) {
	var data models.SenSorData

	// Parse JSON input
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if data.DeviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}
	loc := time.FixedZone("UTC+7", 7*60*60) // Vietnam Time
	data.Timestamp = time.Now().In(loc)
	if data.Status == "" {
		data.Status = "active"
	}
	// Canonicalize so HTTP-ingested rows use the same short device_type as MQTT
	// (e.g. the simulator's HTTP fallback sends "temperature" -> stored as "temp").
	data.DeviceType = canonicalType(data.DeviceType)

	if err := db.DB.Create(&data).Error; err != nil {
		log.Printf("Error saving sensor data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
		return
	}

	// Heartbeat: mark the owning sensor as active (no-op if it isn't registered).
	db.DB.Model(&models.Sensor{}).Where("device_id = ?", data.DeviceID).
		Updates(map[string]interface{}{"timestamp": data.Timestamp, "status": "active"})

	// Publish to the realtime sensor channel only AFTER a successful DB write.
	rabbitmq.Publish("sensor.data", data)

	// Safety: evaluate danger thresholds and raise an alert if breached.
	go EvaluateAndAlert(data)

	c.JSON(http.StatusOK, gin.H{"message": "Data received"})
}

func HandleGetSensorData(c *gin.Context) {
	deviceID := c.Param("device_id")
	startTime := c.Query("start")
	endTime := c.Query("end")

	// Room + time-window scope: a teacher/student may only read a device in a room
	// they teach/study in, and only the readings that fall within their class
	// periods for that room ("xem theo khung giờ lịch dạy/học").
	rooms, windows, isAll := actorRoomScope(c)
	room := roomOf(deviceID)
	if !isAll && !rooms[room] {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn chỉ xem được phòng trong lịch dạy/học của mình"})
		return
	}

	var data []models.SenSorData
	if err := db.DB.Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, startTime, endTime).Order("timestamp asc").Find(&data).Error; err != nil {
		log.Printf("Error retrieving data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve data"})
		return
	}

	if !isAll {
		wins := windows[room]
		kept := make([]models.SenSorData, 0, len(data))
		for _, d := range data {
			if inAnyWindow(wins, d.Timestamp) {
				kept = append(kept, d)
			}
		}
		data = kept
	}
	c.JSON(http.StatusOK, data)
}
func HandlePutSensorData(c *gin.Context) {
	deviceID := c.Param("device_id")
	var data models.SenSorData

	// Parse JSON input
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	var existingData models.SenSorData
	if err := db.DB.Where("device_id = ?", deviceID).Order("timestamp desc").First(&existingData).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sensor data not found"})
		return
	}
	existingData.Value = data.Value
	if data.Status != "" {
		existingData.Status = data.Status
	}
	if err := db.DB.Save(&existingData).Error; err != nil {
		log.Printf("Error updating database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update data"})
		return
	}

	log.Printf("Updated sensor data: %+v", existingData)
	c.JSON(http.StatusOK, gin.H{"message": "Data updated"})
}
func HandleGetSensors(c *gin.Context) {
	var sensors []models.Sensor

	// Retrieve all sensors from the database
	if err := db.DB.Find(&sensors).Error; err != nil {
		log.Printf("Error retrieving sensors: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sensors"})
		return
	}

	// Scope to the caller's rooms: teacher = rooms they teach, student = enrolled
	// rooms, admin = all (so GV/SV only see devices of their own classrooms).
	if rooms, _, isAll := actorRoomScope(c); !isAll {
		kept := make([]models.Sensor, 0, len(sensors))
		for _, s := range sensors {
			if rooms[s.Location] {
				kept = append(kept, s)
			}
		}
		sensors = kept
	}

	c.JSON(http.StatusOK, sensors)
}
func HandlePostSensor(c *gin.Context) {
	var sensor models.Sensor

	if err := c.BindJSON(&sensor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	sensor.Timestamp = time.Now()
	if err := db.DB.Where("device_id = ?", sensor.DeviceID).First(&sensor).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Sensor already exists"})
		return
	} else if err := db.DB.Create(&sensor).Error; err != nil {
		log.Printf("Error saving sensor: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save sensor"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Sensor created"})
}
func HandlePutSensor(c *gin.Context) {
	deviceID := c.Param("device_id")
	var sensor models.Sensor

	// Parse JSON input
	if err := c.BindJSON(&sensor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var existingSensor models.Sensor
	if err := db.DB.Where("device_id = ?", deviceID).First(&existingSensor).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	} else {
		existingSensor.DeviceName = sensor.DeviceName
		existingSensor.DeviceType = sensor.DeviceType
		existingSensor.Location = sensor.Location
		existingSensor.Status = sensor.Status
		if err := db.DB.Save(&existingSensor).Error; err != nil {
			log.Printf("Error updating database: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sensor"})
			return
		}
	}

	log.Printf("Updated sensor: %+v", existingSensor)
	c.JSON(http.StatusOK, gin.H{"message": "Sensor updated"})
}
func HandleDeleteSensor(c *gin.Context) {
	deviceID := c.Param("device_id")

	// Delete sensor from the database
	if err := db.DB.Where("device_id = ?", deviceID).Delete(&models.Sensor{}).Error; err != nil {
		log.Printf("Error deleting sensor: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete sensor"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sensor deleted"})
}
func HandleGetElectricity(c *gin.Context) {
	id := c.Query("id")
	deviceType := c.Query("type")
	var results []struct {
		DeviceType string  `json:"device_type"`
		Value      float64 `json:"value"`
	}
	if err := db.DB.Table("electricities").
		Select("device_type, value").
		Where("device_id = ? AND device_type = ?", id, deviceType).
		Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve electricity records"})
		return
	}

	c.JSON(http.StatusOK, results)
}
func HandlePostElectricity(c *gin.Context) {
	var electricity models.Electricity
	if err := c.BindJSON(&electricity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	electricity.Timestamp = time.Now()
	if err := db.DB.Create(&electricity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create electricity record"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Electricity record created", "electricity": electricity})
}
func HandlePutElectricity(c *gin.Context) {
	id := c.Param("id")
	var electricity models.Electricity

	// Find the electricity record by its device id
	if err := db.DB.First(&electricity, "device_id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Electricity record not found"})
		return
	}

	// Parse the updated data
	if err := c.ShouldBindJSON(&electricity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := db.DB.Save(&electricity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update electricity record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Electricity record updated", "electricity": electricity})
}
func HandleDeleteElectricity(c *gin.Context) {
	id := c.Param("id")

	// Delete the electricity record by its device id
	if err := db.DB.Delete(&models.Electricity{}, "device_id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete electricity record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Electricity record deleted"})
}
func HandlePostDeviceMode(c *gin.Context) {
	deviceID := c.Param("device_id")
	deviceType := c.Param("device_type")

	var req struct {
		Mode int `json:"mode"`
	}

	// Validate identifiers to prevent URL/SSRF injection.
	if !deviceIdent.MatchString(deviceType) || !deviceIdent.MatchString(deviceID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device identifier"})
		return
	}

	// Scope: a teacher may only control devices in rooms they teach (admin = all).
	// The room is parsed from the device_id prefix, which is client-supplied, so it
	// must be checked against the caller's room scope — mirroring the read paths.
	if rooms, _, isAll := actorRoomScope(c); !isAll && !rooms[roomOf(deviceID)] {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn chỉ điều khiển được thiết bị trong phòng theo lịch dạy của mình"})
		return
	}

	// Parse JSON input first, then build the command payload.
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	// Actuator level: 0 = off, 1..3 = on/levels (fan speed; others on=1).
	if req.Mode < 0 || req.Mode > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be 0..3"})
		return
	}

	// Primary path: publish to the room's MQTT command topic; the device
	// subscribes (outbound connection → works behind classroom NAT).
	action := "off"
	if req.Mode > 0 {
		action = "on"
	}
	PublishDeviceCommand(roomOf(deviceID), deviceType, action, req.Mode, "manual")

	// Legacy best-effort direct HTTP (only works if the device is an HTTP server
	// reachable from here); bounded timeout so it can't block the goroutine.
	espURL := fmt.Sprintf("http://%s/%s", deviceType, deviceID)
	deviceHTTP := &http.Client{Timeout: 3 * time.Second}
	if resp, err := deviceHTTP.Post(espURL, "application/json", bytes.NewBufferString(fmt.Sprintf(`{"mode": %d}`, req.Mode))); err == nil {
		resp.Body.Close()
	}
	c.JSON(http.StatusOK, gin.H{"message": "Command sent", "device": deviceType, "mode": req.Mode, "via": "mqtt"})
}
