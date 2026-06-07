package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"smart_classroom/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// env returns the environment variable value or a fallback default.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// dsn builds the PostgreSQL connection string from environment variables,
// falling back to local-dev defaults so the stack still boots out of the box.
func dsn() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		env("DB_HOST", "postgres"),
		env("DB_USER", "nhattoan"),
		env("DB_PASSWORD", "test123"),
		env("DB_NAME", "sensordata"),
		env("DB_PORT", "5432"),
		env("DB_SSLMODE", "disable"),
	)
}

func InitDB() {
	var err error

	// Postgres may still be starting up when this service boots, so retry the
	// connection for a while before giving up.
	for attempt := 1; attempt <= 30; attempt++ {
		DB, err = gorm.Open(postgres.Open(dsn()), &gorm.Config{})
		if err == nil {
			sqlDB, dbErr := DB.DB()
			if dbErr == nil && sqlDB.Ping() == nil {
				break
			}
			err = dbErr
		}
		log.Printf("Waiting for database (attempt %d/30): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Failed to connect to database after retries:", err)
	}

	if err := DB.AutoMigrate(&models.User{}); err != nil {
		log.Fatal("Failed to migrate User:", err)
	}
	modelsToMigrate := []interface{}{
		&models.SenSorData{},
		&models.UserProfile{},
		&models.Face{},
		&models.Notification{},
		&models.Sensor{},
		&models.Building{},
		&models.Classroom{},
		&models.Student{},
		&models.Subject{},
		&models.Teacher{},
		&models.Attendance{},
		&models.ClassroomTeacher{},
		&models.Schedule{},
		&models.Electricity{},
		&models.Class{},
		&models.ClassStudent{},
	}
	if err := DB.AutoMigrate(modelsToMigrate...); err != nil {
		log.Fatal("Failed to migrate database models:", err)
	}
	log.Println("Database connection initialized and migrated successfully")
}
