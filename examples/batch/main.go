package main

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/heureka/gorabbit"
	"github.com/heureka/gorabbit/process"
)

func main() {
	conn, err := amqp.Dial("amqp://localhost:5672")
	if err != nil {
		log.Panic(err)
	}

	consumer, err := gorabbit.NewConsumer(conn, "my-queue")
	if err != nil {
		log.Panic(err)
	}

	// transaction function for processing batch of messages
	tx := func(ctx context.Context, msgs [][]byte) []error {
		for _, msg := range msgs {
			log.Println(string(msg))
		}

		return nil
	}

	// process 100 messages or 1 second of messages at once, whichever comes first
	err = consumer.Start(context.Background(), process.InBatches(100, time.Second, tx, false))
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if err := consumer.Stop(); err != nil {
			log.Panic(err)
		}
	}()

	// consume for 10 seconds
	time.Sleep(10 * time.Second)
}
