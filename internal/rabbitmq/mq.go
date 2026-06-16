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
		"topic", // routing keys: sensor.*, notify.*, attendance.*, classroom.*
		true,    // durable
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Fatalf("Failed to declare exchange: %s", err)
	}

	// Dead-letter exchange + queue: messages a consumer nacks (poison/failed)
	// land here instead of being lost.
	if err = Channel.ExchangeDeclare("dlx", "fanout", true, false, false, false, nil); err != nil {
		log.Fatalf("Failed to declare DLX: %s", err)
	}
	if _, err = Channel.QueueDeclare("dead_letters", true, false, false, false, nil); err == nil {
		_ = Channel.QueueBind("dead_letters", "", "dlx", false, nil)
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
		amqp.Table{"x-dead-letter-exchange": "dlx"}, // failed messages -> DLX
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
		false, // manual ack — don't lose messages on crash
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to consume from %s queue: %s", queueName, err)
	}
	log.Printf("Started consuming from %s queue (manual ack)", queueName)
	go func() {
		for msg := range msgs {
			func(m amqp.Delivery) {
				// A handler panic must not lose the message: nack -> dead-letter.
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[%s] handler panic: %v -> dead-letter", label, r)
						_ = m.Nack(false, false)
					}
				}()
				handle(m.Body)
				_ = m.Ack(false)
			}(msg)
		}
	}()
}

// ConsumeKeyed declares+binds a queue and consumes with the routing key passed
// to the handler (used for MQTT device topics classroom.{room}.{kind}.{leaf}).
func ConsumeKeyed(queueName, bindingKey string, handle func(routingKey string, body []byte)) {
	if _, err := DeclareQueue(queueName, bindingKey); err != nil {
		return
	}
	msgs, err := Channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to consume from %s: %s", queueName, err)
	}
	log.Printf("Started consuming %s (key %s)", queueName, bindingKey)
	go func() {
		for msg := range msgs {
			func(m amqp.Delivery) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[%s] panic: %v -> dead-letter", queueName, r)
						_ = m.Nack(false, false)
					}
				}()
				handle(m.RoutingKey, m.Body)
				_ = m.Ack(false)
			}(msg)
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
