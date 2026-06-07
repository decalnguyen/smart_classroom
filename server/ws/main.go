package main

import (
	"log"
	"os"

	"smart_classroom/internal/handlers"
	"smart_classroom/internal/rabbitmq"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	rabbitmq.Init()

	// Bind each queue to its message type on the topic exchange.
	if _, err := rabbitmq.DeclareQueue("sensor_data", "sensor.*"); err != nil {
		log.Fatalf("declare sensor_data: %v", err)
	}
	if _, err := rabbitmq.DeclareQueue("notification_data", "notify.*"); err != nil {
		log.Fatalf("declare notification_data: %v", err)
	}
	if _, err := rabbitmq.DeclareQueue("attendance_data", "attendance.*"); err != nil {
		log.Fatalf("declare attendance_data: %v", err)
	}

	// Consume each stream and fan it out to the matching WS clients.
	rabbitmq.ConsumeAndHandleSensor("sensor_data", handlers.HandleSensorWS)
	rabbitmq.ConsumeAndHandleNotification("notification_data", handlers.HandleNotificationsWS)
	rabbitmq.ConsumeAndHandleAttendance("attendance_data", handlers.HandleAttendanceWS)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:3000", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/ws/notifications", handlers.NotificationsWsHandler)
	r.GET("/ws/sensor", handlers.SensorWsHandler)
	r.GET("/ws/attendance", handlers.AttendanceWsHandler)

	port := os.Getenv("WS_PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("🟢 WebSocket server listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("WebSocket server failed: %v", err)
	}
}
