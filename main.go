package main

import (
	"log"
	"net/http"
	"smart_classroom/db"
	"smart_classroom/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	db.InitDB()
	r := gin.Default()
	r.Use(cors.Default())
	r.POST("/signup", handlers.SignUp)
	r.POST("/login", handlers.Login)

	r.POST("/sensor", handlers.HandlePostSensorData)
	r.GET("/sensor", handlers.HandleGetSensorData)
	r.PUT("/sensor/:device_id", handlers.HandlePutSensorData)

	r.GET("/sensorinf", handlers.HandleGetSensors)
	r.POST("/sensorinf", handlers.HandlePostSensor)
	r.PUT("/sensorinf/:device_id", handlers.HandlePutSensor)
	r.DELETE("/sensorinf/:device_id", handlers.HandleDeleteSensor)

	port := ":8081"
	r.Run(port)
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
