package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/handlers"
	"smart_classroom/internal/middleware"
	"smart_classroom/internal/rabbitmq"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func allowedOrigins() []string {
	origins := []string{"http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:3000", "http://127.0.0.1:5173"}
	if o := os.Getenv("FRONTEND_ORIGIN"); o != "" {
		origins = append(origins, o)
	}
	return origins
}

func main() {
	db.InitDB()
	handlers.SeedDefaults()
	handlers.SeedMockData()
	handlers.SeedTeacherAssignments()
	handlers.SeedAccountLinks()
	rabbitmq.Init()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// ---- Public (no auth) ----
	r.POST("/signup", handlers.SignUp)
	r.POST("/login", handlers.Login)
	r.POST("/logout", handlers.Logout)

	// Device ingestion (no user JWT): ESP32 sensors + AI camera face-scan events.
	r.POST("/sensor", handlers.HandlePostSensorData)
	r.POST("/attendance/scan", handlers.HandleAttendanceScan)

	// ---- Authenticated (any role) ----
	auth := r.Group("/")
	auth.Use(middleware.RequireRole())
	{
		auth.GET("/user", handlers.User)
		auth.GET("/stats/overview", handlers.HandleStatsOverview)
		auth.GET("/my/classrooms", handlers.HandleMyClassrooms)
		auth.GET("/my/attendance", handlers.HandleMyAttendance)

		// Reads available to every authenticated user.
		auth.GET("/sensor/:device_id", handlers.HandleGetSensorData)
		auth.GET("/sensorinf", handlers.HandleGetSensors)
		auth.GET("/buildings", handlers.HandleGetBuildings)
		auth.GET("/classrooms", handlers.HandleGetClassrooms)
		auth.GET("/classes/:id", handlers.HandleGetClass)
		auth.GET("/students", handlers.HandleGetStudents)
		auth.GET("/teachers", handlers.HandleGetTeachers)
		auth.GET("/attendance", handlers.HandleGetAttendance)
		auth.GET("/electricity", handlers.HandleGetElectricity)

		// Personal schedule (own data, any role).
		auth.GET("/schedules", handlers.HandleGetSchedules)
		auth.POST("/schedules", handlers.HandlePostSchedule)
		auth.PUT("/schedules/:id", handlers.HandlePutSchedule)
		auth.DELETE("/schedules/:id", handlers.HandleDeleteSchedule)

		// Personal notifications (own data, any role).
		auth.GET("/notifications", handlers.HandleGetNotifications)
		auth.PUT("/notifications/:id", handlers.HandleUpdateNotification)
		auth.DELETE("/notifications/:id", handlers.HandleDeleteNotification)
	}

	// ---- Teacher + Admin ----
	staff := r.Group("/")
	staff.Use(middleware.RequireRole("admin", "teacher"))
	{
		// Attendance management.
		staff.POST("/attendance", handlers.HandlePostAttendance)
		staff.PUT("/attendance/:id", handlers.HandlePutAttendance)
		staff.DELETE("/attendance/:id", handlers.HandleDeleteAttendance)

		// Class management.
		staff.POST("/classes", handlers.HandlePostClass)
		staff.PUT("/classes/:id", handlers.HandlePutClass)
		staff.DELETE("/classes/:id", handlers.HandleDeleteClass)

		// Device control + sensor reading edits.
		staff.PUT("/sensor/:device_id", handlers.HandlePutSensorData)
		staff.POST("/device/:device_type/:device_id/mode", handlers.HandlePostDeviceMode)

		// Electricity records.
		staff.POST("/electricity", handlers.HandlePostElectricity)
		staff.PUT("/electricity/:id", handlers.HandlePutElectricity)
		staff.DELETE("/electricity/:id", handlers.HandleDeleteElectricity)

		// Targeted notification creation.
		staff.POST("/notifications", handlers.HandleCreateNotification)

		// Attendance analytics (scoped: teacher = own classrooms, admin = all).
		staff.GET("/reports/attendance", handlers.HandleAttendanceReport)
		staff.GET("/reports/attendance/export", handlers.HandleAttendanceReportExport)
	}

	// ---- Admin only ----
	admin := r.Group("/")
	admin.Use(middleware.RequireRole("admin"))
	{
		admin.POST("/buildings", handlers.HandlePostBuilding)
		admin.PUT("/buildings/:id", handlers.HandlePutBuilding)
		admin.DELETE("/buildings/:id", handlers.HandleDeleteBuilding)

		admin.POST("/classrooms", handlers.HandlePostClassroom)
		admin.PUT("/classrooms/:id", handlers.HandlePutClassroom)
		admin.DELETE("/classrooms/:id", handlers.HandleDeleteClassroom)

		admin.POST("/students", handlers.HandlePostStudent)
		admin.PUT("/students/:id", handlers.HandlePutStudent)
		admin.DELETE("/students/:id", handlers.HandleDeleteStudent)

		admin.POST("/teachers", handlers.HandlePostTeacher)
		admin.PUT("/teachers/:id", handlers.HandlePutTeacher)
		admin.DELETE("/teachers/:id", handlers.HandleDeleteTeacher)

		// Sensor/device registry management.
		admin.POST("/sensorinf", handlers.HandlePostSensor)
		admin.PUT("/sensorinf/:device_id", handlers.HandlePutSensor)
		admin.DELETE("/sensorinf/:device_id", handlers.HandleDeleteSensor)

		// Teacher ↔ classroom assignment management.
		admin.GET("/classroom-teachers", handlers.HandleGetClassroomTeachers)
		admin.POST("/classroom-teachers", handlers.HandlePostClassroomTeacher)
		admin.DELETE("/classroom-teachers", handlers.HandleDeleteClassroomTeacher)
	}

	// Background sensor liveness checker.
	handlers.SensorChecker()

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8081"
	}

	srv := &http.Server{Addr: ":" + port, Handler: r}
	go func() {
		log.Printf("🟢 HTTP API server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down HTTP API server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Forced shutdown: %v", err)
	}
}
