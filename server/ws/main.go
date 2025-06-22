package main

import (
	"log"
	"smart_classroom/internal/handlers"
	"smart_classroom/internal/rabbitmq"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Send message to all sensor clients

func main() {

	rabbitmq.Init()
	rabbitmq.DecalareQueue("sensor_data")
	rabbitmq.DecalareQueue("notification_data")
	rabbitmq.ConsumeAndHandleSensor("sensor_data", handlers.HandleSensorWS)
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.GET("/ws/notifications", handlers.NotificationsWsHandler)
	r.GET("/ws/sensor", handlers.SensorWsHandler)
	log.Println("🟢 WebSocket server listening on :8082")
	r.Run(":8082")
}
