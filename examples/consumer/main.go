package main

import (
	"context"
	"log"
	"time"

	"github.com/heureka/gorabbit"
	"github.com/heureka/gorabbit/channel"
	"github.com/heureka/gorabbit/connection"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// connection with re-dialing capabilities.
	conn, err := connection.Dial("amqp://localhost:5672")
	if err != nil {
		log.Panic(err)
	}

	// channel with re-connection capabilities.
	ch, err := channel.New(conn)
	if err != nil {
		log.Panic(err)
	}

	consumer := gorabbit.NewConsumer(
		ch,
		"example-queue",
		gorabbit.WithConsumerTag("example-consumer"),           // set up custom consumer tag
		gorabbit.WithConsumeArgs(amqp.Table{"example": "tag"}), // add additional consumer tags
		gorabbit.WithConsumeAutoAck(),                          // automatically ACK all messages
		gorabbit.WithConsumeExclusive(),                        // tell the server to ensure that this is the sole consumer from this queue
		gorabbit.WithConsumeNoWait(),                           //  tell the server to immediately begin deliveries
	)
	err = consumer.Start(context.Background(), gorabbit.ProcessFunc(func(ctx context.Context, deliveries <-chan amqp.Delivery) error {
		for d := range deliveries {
			log.Println(d.Body)
		}
		return nil
	}))
	if err != nil {
		log.Panic(err)
	}

	time.Sleep(10 * time.Second) // consumer for 10 seconds
	if err := consumer.Stop(); err != nil {
		log.Panic(err)
	}
}
