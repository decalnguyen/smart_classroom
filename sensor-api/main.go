package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type SenSorData struct {
	DeviceID    string  `json:"device_id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

var db *gorm.DB

func initDB() {
	dsn := "host=localhost user=nhattoan password=test123 dbname=sensordata port=5432 sslmode=disable "
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping the database:", err)
	}

	log.Println("Database connection initialized successfully")
	db.AutoMigrate(&SenSorData{})
}

func sensorData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var data SenSorData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received sensor data: %+v", data)
	w.WriteHeader(http.StatusOK)

	fmt.Fprint(w, `{"message": "Data received"}`)
}

func main() {
	initDB()
	r := gin.Default()
	r.POST("/sensor", func(c *gin.Context) {
		var data SenSorData
		err := c.BindJSON(&data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.Create(&data).Error; err != nil {
			log.Printf("Error saving to database: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
			return
		}

		log.Printf("Received sensor data: %+v", data)
		c.JSON(http.StatusOK, gin.H{"message": "Data received"})
	})
	r.GET("/sensor", func(c *gin.Context) {
		var data SenSorData
		db.Find(&data)
		c.JSON(http.StatusOK, data)
	})
	http.HandleFunc("/sensor", sensorData)

	port := ":8081"
	r.Run(port)
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
