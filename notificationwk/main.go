package main

import (
	"encoding/json"
	"log"

	"smart_classroom/models"

	"github.com/streadway/amqp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func main() {
	// DB connection
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

	// RabbitMQ connection
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatal("RabbitMQ error:", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Channel error:", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("attendance_notification", false, false, false, false, nil)
	if err != nil {
		log.Fatal("Queue declare error:", err)
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatal("Consume error:", err)
	}

	log.Println("Worker is consuming messages...")

	for d := range msgs {
		var notif models.Notification
		if err := json.Unmarshal(d.Body, &notif); err != nil {
			log.Println("Invalid message format:", err)
			continue
		}

		// Save to DB
		if err := DB.Create(&notif).Error; err != nil {
			log.Println("Failed to save notification to DB:", err)
			continue
		}

		// TODO: Push to WebSocket (using Redis pub/sub, gRPC, or direct call if in same process)
		log.Println("Saved & ready to push notification:", notif)
	}
}
