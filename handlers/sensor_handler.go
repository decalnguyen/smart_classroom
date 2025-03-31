package handlers

import (
	"log"
	"net/http"

	"smart_classroom/db"
	"smart_classroom/models"

	"github.com/gin-gonic/gin"
)

func HandlePostSensorData(c *gin.Context) {
	var data models.SenSorData

	// Parse JSON input
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := db.DB.Where("device_id = ?", data.DeviceID).First(&data).Error; err == nil {

		log.Printf("Device ID already exists: %s", data.DeviceID)
		c.JSON(http.StatusOK, gin.H{"message": "Data updated"})
		return
	} else {
		if err := db.DB.Create(&data).Error; err != nil {
			log.Printf("Error saving to database: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
			return
		}
	}

	log.Printf("Received sensor data: %+v", data)
	c.JSON(http.StatusOK, gin.H{"message": "Data received"})
}

func HandleGetSensorData(c *gin.Context) {
	var data []models.SenSorData

	// Retrieve all sensor data from the database
	if err := db.DB.Find(&data).Error; err != nil {
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
		existingData.Temperature = data.Temperature
		existingData.Humidity = data.Humidity
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
