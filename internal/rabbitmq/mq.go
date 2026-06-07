package rabbitmq

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/streadway/amqp"
)

var Conn *amqp.Connection
var Channel *amqp.Channel

const ExchangeName = "main_exchange"

func url() string {
	if v := os.Getenv("RABBITMQ_URL"); v != "" {
		return v
	}
	return "amqp://guest:guest@rabbitmq:5672/"
}

// Init connects to RabbitMQ (retrying while the broker boots) and declares a
// durable topic exchange. A topic exchange lets sensor and notification
// consumers each receive only the message types they bind to.
func Init() {
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		Conn, err = amqp.Dial(url())
		if err == nil {
			break
		}
		log.Printf("Waiting for RabbitMQ (attempt %d/30): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ after retries: %s", err)
	}

	Channel, err = Conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %s", err)
	}

	if err = Channel.ExchangeDeclare(
		ExchangeName,
		"topic", // routing keys: sensor.*, notify.*
		true,    // durable
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Fatalf("Failed to declare exchange: %s", err)
	}
}

// Publish marshals msg to JSON and publishes it with the given routing key.
func Publish(routingKey string, msg interface{}) {
	body, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %s", err)
		return
	}
	if Channel == nil {
		log.Printf("RabbitMQ channel not initialized; dropping message for %s", routingKey)
		return
	}
	if err := Channel.Publish(
		ExchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		log.Printf("Failed to publish message: %s", err)
		return
	}
	log.Printf("Published message with routing key %s: %s", routingKey, body)
}

// DeclareQueue declares a durable queue and binds it to the exchange with the
// given routing-key pattern (e.g. "sensor.*" or "notify.*").
func DeclareQueue(queueName, bindingKey string) (amqp.Queue, error) {
	queue, err := Channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Printf("Failed to declare queue %s: %s", queueName, err)
		return amqp.Queue{}, err
	}
	if err = Channel.QueueBind(
		queue.Name,
		bindingKey,
		ExchangeName,
		false,
		nil,
	); err != nil {
		log.Printf("Failed to bind queue %s (key %s): %s", queueName, bindingKey, err)
		return amqp.Queue{}, err
	}
	log.Printf("Queue %s declared and bound with key %s", queue.Name, bindingKey)
	return queue, nil
}

func consume(queueName string, handle func(msg []byte), label string) {
	msgs, err := Channel.Consume(
		queueName,
		"",
		true,  // auto-acknowledge
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to consume from %s queue: %s", queueName, err)
	}
	log.Printf("Started consuming from %s queue", queueName)
	go func() {
		for msg := range msgs {
			log.Printf("[%s] received from MQ -> forwarding to WS: %s", label, msg.Body)
			handle(msg.Body)
		}
	}()
}

func ConsumeAndHandleNotification(queueName string, handle func(msg []byte)) {
	consume(queueName, handle, "notification")
}

func ConsumeAndHandleSensor(queueName string, handle func(msg []byte)) {
	consume(queueName, handle, "sensor")
}

func ConsumeAndHandleAttendance(queueName string, handle func(msg []byte)) {
	consume(queueName, handle, "attendance")
}
