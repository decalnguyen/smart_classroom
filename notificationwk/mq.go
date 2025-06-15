package rabbitmq

import (
	"encoding/json"
	"log"
	"smart_classroom/handlers"

	"github.com/streadway/amqp"
)

var conn *amqp.Connection
var channel *amqp.Channel

func Init() {
	var err error
	if conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/"); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}

	if channel, err = conn.Channel(); err != nil {
		log.Fatalf("Failed to open a channel: %s", err)
	}
	if err = channel.ExchangeDeclare(
		"notification_exchange",
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
func PublishNotification(msg interface{}) {
	body, _ := json.Marshal(msg)
	if err := channel.Publish(
		"notification_exchange", // exchange
		"",
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
func ConsumeAndHandleNotifications() {
	queue, _ := channel.QueueDeclare("notification_queue", false, true, true, false, nil)
	channel.QueueBind(queue.Name, "", "notification_exchange", false, nil)
	msgs, _ := channel.Consume(queue.Name, "", true, false, false, false, nil)
	go func() {
		for msg := range msgs {
			handlers.HandleNotificationsWS(msg.Body)
		}
	}()

}
