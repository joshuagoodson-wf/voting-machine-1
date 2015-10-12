package main

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/streadway/amqp"
	"log"
)

type Team struct {
	Team string
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func main() {
	// Establish RabbitMQ connection
	rabbitConnection, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer rabbitConnection.Close()

	// Establish Redis connection
	redisConnection, err := redis.Dial("tcp", ":6379")
	failOnError(err, "Failed to connect to Redis")
	defer redisConnection.Close()

	// Open a channel
	channel, err := rabbitConnection.Channel()
	failOnError(err, "Failed to open a channel")
	defer channel.Close()

	// Declare the queue
	queue, err := channel.QueueDeclare(
		"votes", // Queue name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// Consume messages off the queue
	msgs, err := channel.Consume(
		queue.Name, // queue name
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	failOnError(err, "Failed to register a consumer")

	// Create channel for consuming messages
	consume := make(chan bool)

	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %s", msg.Body)
			var team Team
			err := json.Unmarshal(msg.Body, &team)
			failOnError(err, "Failed to unmarshal JSON body of message")

			// Increment vote count for team
			redisConnection.Do("INCR", fmt.Sprintf("team%s", team.Team))
			log.Printf("Incremented vote count for team %s", team.Team)

			msg.Ack(false)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-consume
}
