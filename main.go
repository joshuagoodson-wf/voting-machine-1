package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/garyburd/redigo/redis"
	"github.com/rakyll/globalconf"
	"github.com/streadway/amqp"
)

// Define flags
var (
	configPath               = flag.String("config", "config.yml", "Path to a configuration file")
	redisConnectionString    = flag.String("redis", "redis://127.0.0.1:6379", "Redis connection string")
	rabbitmqConnectionString = flag.String("rabbitmq", "amqp://guest:guest@localhost:5672/", "AMMQ connection string")
	defaultQueue             = flag.String("queue", "votes", "Ephemeral AMQP queue name")
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
	// Configure options from environment variables
	conf, err := globalconf.NewWithOptions(&globalconf.Options{
		EnvPrefix: "VOTING_",
	})
	failOnError(err, "Failed to parse options")
	conf.ParseAll()

	// Establish RabbitMQ connection
	log.Print("Connecting to AMQP...")
	rabbitConnection, err := amqp.Dial(*rabbitmqConnectionString)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer rabbitConnection.Close()

	// Establish Redis connection
	log.Print("Connecting to Redis...")
	redisConnection, err := redis.DialURL(*redisConnectionString)
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
			redisConnection.Do("INCR", fmt.Sprintf("%s", team.Team))
			log.Printf("Incremented vote count for team %s", team.Team)

			msg.Ack(false)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-consume
}
