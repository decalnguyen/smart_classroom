package main

import (
	"log"
	"smart_classroom/internal/db"
	"smart_classroom/internal/handlers"
	"smart_classroom/internal/rabbitmq"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	db.InitDB()
	rabbitmq.Init()
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	//r.Use(middleware.ClassroomNetworkMiddleware())
	r.POST("/signup", handlers.SignUp)
	r.POST("/login", handlers.Login)
	r.POST("/logout", handlers.Logout)
	r.GET("/user", handlers.User)

	r.POST("/sensor", handlers.HandlePostSensorData)
	r.GET("/sensor/:device_id", handlers.HandleGetSensorData)
	r.PUT("/sensor/:device_id", handlers.HandlePutSensorData)

	r.GET("/sensorinf", handlers.HandleGetSensors)
	r.POST("/sensorinf", handlers.HandlePostSensor)
	r.PUT("/sensorinf/:device_id", handlers.HandlePutSensor)
	r.DELETE("/sensorinf/:device_id", handlers.HandleDeleteSensor)

	r.GET("/buildings", handlers.HandleGetBuildings)
	r.POST("/buildings", handlers.HandlePostBuilding)
	r.PUT("/buildings/:id", handlers.HandlePutBuilding)
	r.DELETE("/buildings/:id", handlers.HandleDeleteBuilding)

	// Classroom APIs
	r.GET("/classrooms", handlers.HandleGetClassrooms)
	r.POST("/classrooms", handlers.HandlePostClassroom)
	r.PUT("/classrooms/:id", handlers.HandlePutClassroom)
	r.DELETE("/classrooms/:id", handlers.HandleDeleteClassroom)

	// Class APIs
	r.GET("/classes/:id", handlers.HandleGetClass)
	r.POST("/classes", handlers.HandlePostClass)
	r.PUT("/classes/:id", handlers.HandlePutClass)
	r.DELETE("/classes/:id", handlers.HandleDeleteClass)
	// Student APIs
	r.GET("/students", handlers.HandleGetStudents)
	r.POST("/students", handlers.HandlePostStudent)
	r.PUT("/students/:id", handlers.HandlePutStudent)
	r.DELETE("/students/:id", handlers.HandleDeleteStudent)

	// Teacher APIs
	r.GET("/teachers", handlers.HandleGetTeachers)
	r.POST("/teachers", handlers.HandlePostTeacher)
	r.PUT("/teachers/:id", handlers.HandlePutTeacher)
	r.DELETE("/teachers/:id", handlers.HandleDeleteTeacher)

	// Schedule routes
	r.GET("/schedules", handlers.HandleGetSchedules)
	r.POST("/schedules", handlers.HandlePostSchedule)
	r.PUT("/schedules/:id", handlers.HandlePutSchedule)
	r.DELETE("/schedules/:id", handlers.HandleDeleteSchedule)

	//Attandace routes
	r.GET("/attendance", handlers.HandleGetAttendance)
	r.POST("/attendance", handlers.HandlePostAttendance)
	r.PUT("/attendance/:id", handlers.HandlePutAttendance)
	r.DELETE("/attendance/:id", handlers.HandleDeleteAttendance)
	//
	//Electricity routes
	r.GET("/electricity", handlers.HandleGetElectricity)
	r.POST("/electricity", handlers.HandlePostElectricity)
	r.PUT("/electricity/:id", handlers.HandlePutElectricity)
	r.DELETE("/electricity/:id", handlers.HandleDeleteElectricity)

	//Notification routes
	r.GET("/notifications", handlers.HandleGetNotifications)
	r.POST("/notifications", handlers.HandleCreateNotification)
	r.PUT("/notifications/:id", handlers.HandleUpdateNotification)
	r.DELETE("/notifications/:id", handlers.HandleDeleteNotification)

	r.POST("/device/:device_type/:device_id/mode", handlers.HandlePostDeviceMode)

	handlers.SensorChecker()
	port := ":8081"
	r.Run(port)
	log.Printf("Starting server on port %s", port)

}
