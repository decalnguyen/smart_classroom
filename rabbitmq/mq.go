package rabbitmq

import (
	"encoding/json"
	"log"
	"smart_classroom/ws"

	"github.com/streadway/amqp"
)

var conn *amqp.Connection
var channel *amqp.Channel

func Init() {
	var err error
	if conn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/"); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}

	if channel, err = conn.Channel(); err != nil {
		log.Fatalf("Failed to open a channel: %s", err)
	}
	if err = channel.ExchangeDeclare(
		"main_exchange",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Fatalf("Failed to declare exchange: %s", err)
	}
}
func Publish(routingKey string, msg interface{}) {
	body, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %s", err)
		return
	}
	// Publish the message to the exchange with the specified routing key
	log.Printf("Publishing message with routing key: %s, body: %s", routingKey, body)
	if err := channel.Publish(
		"main_exchange", // exchange
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		log.Printf("Failed to publish message: %s", err)
	}
}
func ConsumeAndHandleMessage() {
	//1. Declare a queue for notifications
	notiQueue, _ := channel.QueueDeclare("notification_queue", false, true, true, false, nil)
	channel.QueueBind(notiQueue.Name, "notify.*", "main_exchange", false, nil)
	//2. Declare a queue for sensor notifications
	sensorQueue, _ := channel.QueueDeclare("sensor_queue", false, true, true, false, nil)
	channel.QueueBind(sensorQueue.Name, "sensor.*", "main_exchange", false, nil)

	//3. Consume messages from the notification queue
	go func() {
		msgs, err := channel.Consume(
			notiQueue.Name,
			"",
			true,  // auto-acknowledge
			false, // exclusive
			false, // no-local
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			log.Fatalf("Failed to consume messages from notification queue: %s", err)
		} else {
			log.Println("Started consuming messages from notification queue")
		}
		for msg := range msgs {
			ws.HandleNotificationsWS(msg.Body)
		}
	}()
	//4. Consume messages from the sensor queue
	go func() {
		msgs, err := channel.Consume(
			sensorQueue.Name,
			"",
			true,  // auto-acknowledge
			false, // exclusive
			false, // no-local
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			log.Fatalf("Failed to consume messages from sensor queue: %s", err)
		} else {
			log.Println("Started consuming messages from sensor queue")
		}
		for msg := range msgs {
			ws.HandleSensorNotificationsWS(msg.Body)
		}
	}()
}
