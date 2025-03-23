package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/golang-jwt/jwt/v5"
)

type SenSorData struct {
	DeviceID    string  `json:"device_id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
    Username string `gorm:"unique" json:"username"`
    Password string `json:"password"`
}

var db *gorm.DB
var r *gin.Default()

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
	db.AutoMigrate(&SenSorData{}, &User{})
}

func HandlePostSensorData(c *gin.Context) {
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
	r.POST("/sensor", HandlePostSensorData)
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
