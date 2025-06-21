package main

import (
	"smart_classroom/internal/handlers"
	"smart_classroom/internal/rabbitmq"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	rabbitmq.Init()
	rabbitmq.ConsumeAndHandleMessage()
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.GET("/ws/notifications", handlers.NotificationsWsHandler)
	r.GET("/ws/sensor", handlers.SensorWsHandler)
}
