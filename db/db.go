package db

import (
	"log"
	"smart_classroom/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := "host=postgres user=nhattoan password=test123 dbname=sensordata port=5432 sslmode=disable "
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping the database:", err)
	}
	if err := DB.AutoMigrate(&models.User{}); err != nil {
		log.Fatal("Failed to migrate User:", err)
	}
	modelsToMigrate := []interface{}{
		&models.SenSorData{},
		//&models.User{},
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
	}
	log.Println("Database connection initialized successfully")
	if err := DB.AutoMigrate(modelsToMigrate...); err != nil {
		log.Fatal("Failed to migrate database models:", err)
	}

}
