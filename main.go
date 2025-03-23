package main

import (
	"log"
	"net/http"
	"smart_classroom/db"
	"smart_classroom/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	db.InitDB()
	r := gin.Default()
	r.POST("/signup", handlers.SignUp)
	r.POST("/login", handlers.Login)

	r.POST("/sensor", handlers.HandlePostSensorData)
	r.GET("/sensor", handlers.HandleGetSensorData)

	port := ":8081"
	r.Run(port)
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
