package rabbitmq

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

var Conn *amqp.Connection
var Channel *amqp.Channel

func Init() {
	var err error
	Conn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}
	//defer conn.Close()
	Channel, err = Conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %s", err)
	}
	//defer channel.Close()
	if err = Channel.ExchangeDeclare(
		"main_exchange",
		"fanout",
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
	if err := Channel.Publish(
		"main_exchange", // exchange
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
func DecalareQueue(queueName string) (amqp.Queue, error) {
	// Declare a queue with the specified name
	queue, err := Channel.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	if err != nil {
		log.Printf("Failed to declare queue %s: %s", queueName, err)
		return amqp.Queue{}, err
	}
	if err = Channel.QueueBind(
		queue.Name,      // queue name
		"",              // routing key
		"main_exchange", // exchange
		false,           // no-wait
		nil,             // arguments
	); err != nil {
		log.Printf("Failed to bind queue %s to exchange: %s", queue.Name, err)
		return amqp.Queue{}, err
	}
	log.Printf("Queue %s declared and bound to exchange", queue.Name)
	return queue, nil
}
func ConsumeAndHandleNotification(queueName string, handle func(msg []byte)) {
	//3. Consume messages from the notification queue
	go func() {
		msgs, err := Channel.Consume(
			queueName,
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
			log.Printf("Received notification message from MQ now Send to WS: %s", msg.Body)
			handle(msg.Body)
		}
	}()
}
func ConsumeAndHandleSensor(queueName string, handle func(msg []byte)) {
	//4. Consume messages from the sensor queue
	msgs, err := Channel.Consume(
		queueName,
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
	go func() {
		for msg := range msgs {
			log.Printf("Received sensor message from MQ now Send to WS: %s", msg.Body)
			handle(msg.Body)
		}
	}()
}
