package handlers

import (
	"bytes"
	"log"
	"net/http"
	"regexp"
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
func CheckSensorStatus() {
	var sensors []models.Sensor

	// Fetch all active sensors from the database
	if err := db.DB.Where("status = ?", "Active").Find(&sensors).Error; err != nil {
		log.Printf("Error fetching sensors: %v", err)
		return
	}

	// Check each sensor's last activity
	for _, sensor := range sensors {
		if time.Since(sensor.Timestamp) > 5*time.Minute { // Threshold: 5 minutes
			// Update the sensor's status to "inactive" in the Sensor table
			if err := db.DB.Model(&sensor).Where("status = ?", "Active").Update("status", "Inactive").Error; err != nil {
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

	if err := db.DB.Create(&data).Error; err != nil {
		log.Printf("Error saving sensor data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
		return
	}

	// Heartbeat: mark the owning sensor as active (no-op if it isn't registered).
	db.DB.Model(&models.Sensor{}).Where("device_id = ?", data.DeviceID).
		Updates(map[string]interface{}{"timestamp": data.Timestamp, "status": "Active"})

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
	var data []models.SenSorData

	// Retrieve all sensor data from the database
	if err := db.DB.Where("device_id = ? AND timestamp BETWEEN ? AND ?", deviceID, startTime, endTime).Order("timestamp asc").Find(&data).Error; err != nil {
		log.Printf("Error retrieving data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve data"})
		return
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

	// Parse JSON input first, then build the command payload.
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	espURL := fmt.Sprintf("http://%s/%s", deviceType, deviceID)
	payload := fmt.Sprintf(`{"mode": %d}`, req.Mode)

	// Broadcast the command so other consumers / the UI can react regardless of
	// whether physical hardware is reachable.
	rabbitmq.Publish("command.device", gin.H{
		"device_type": deviceType,
		"device_id":   deviceID,
		"mode":        req.Mode,
	})

	resp, err := http.Post(espURL, "application/json", bytes.NewBufferString(payload))
	if err != nil {
		// No physical device in the demo: accept and queue the command.
		log.Printf("Device %s/%s unreachable (queued command): %v", deviceType, deviceID, err)
		c.JSON(http.StatusOK, gin.H{"message": "Command queued (device offline)", "device": deviceType, "mode": req.Mode})
		return
	}
	defer resp.Body.Close()

	c.JSON(http.StatusOK, gin.H{"message": "Command sent", "device": deviceType, "mode": req.Mode})
}
