package handlers

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"fmt"
	"smart_classroom/db"
	"smart_classroom/models"
	"smart_classroom/rabbitmq"

	"github.com/gin-gonic/gin"
)

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

	// Fetch all sensors from the database
	if err := db.DB.Where("status = ?", "Activate").Find(&sensors).Error; err != nil {
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
	loc := time.FixedZone("UTC+7", 7*60*60) // Vietnam Time
	nowVN := time.Now().In(loc)
	data.Timestamp = nowVN

	if err := db.DB.Create(&data).Error; err != nil {
		if err := db.DB.Model(&models.Sensor{}).Where("device_id = ?", data.DeviceID).
			Updates(map[string]interface{}{
				"timestamp": time.Now(),
				"status":    "Active",
			}).Error; err != nil {
			log.Printf("Error updating sensor timestamp: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update timestamp"})
		}
		log.Printf("Error saving to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
		return
	}
	// Publish the sensor data to RabbitMQ
	rabbitmq.Publish("sensor.data", data)

	log.Printf("Received sensor data: %+v", data)
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
	if err := db.DB.Where("device_id = ?", deviceID).First(&existingData).Error; err != nil {
		log.Printf("Device ID not found: %s", data.DeviceID)
		return
	} else {
		existingData.Value = data.Value
		if err := db.DB.Save(&data).Error; err != nil {
			log.Printf("Error updating database: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update data"})
			return
		}
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
		log.Printf("Device ID already exists: %s", sensor.DeviceID)
		c.JSON(http.StatusOK, gin.H{"message": "Sensor already exists"})
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
		log.Printf("Device ID not found: %s", sensor.DeviceID)
		return
	} else {
		existingSensor.DeviceName = sensor.DeviceName
		existingSensor.DeviceType = sensor.DeviceType
		existingSensor.Location = sensor.Location
		existingSensor.Status = sensor.Status
		if err := db.DB.Save(&sensor).Error; err != nil {
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

	// Find the electricity record by ID
	if err := db.DB.First(&electricity, "electricity_id = ?", id).Error; err != nil {
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

	// Delete the electricity record by ID
	if err := db.DB.Delete(&models.Electricity{}, "electricity_id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete electricity record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Electricity record deleted"})
}
func HandlePostDeviceMode(c *gin.Context) {
	deviceID := c.Param("device_id")
	deviceType := c.Param("device_type")

	var req struct {
		Mode string `json:"mode"`
	}

	espURL := fmt.Sprintf("http://%s/%s", deviceType, deviceID)
	payload := fmt.Sprintf(`{"mode": %d}`, req.Mode)

	// Parse JSON input
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	// Update the dimmer value in the database

	resp, err := http.Post(espURL, "application/json", bytes.NewBuffer([]byte(payload)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send command to ESP32"})
		return
	}
	defer resp.Body.Close()

	c.JSON(http.StatusOK, gin.H{"message": "Command sent", "device": deviceType, "mode": req.Mode})
}
