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

	// Save data to the database
	if err := db.DB.Create(&data).Error; err != nil {
		log.Printf("Error saving to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
		return
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
