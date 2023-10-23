package main

import (
	"context"
	"log"
	"time"

	"github.com/heureka/gorabbit"
	"github.com/heureka/gorabbit/consumer"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cons, err := gorabbit.NewConsumer(
		"amqp://localhost:5672",
		"example-queue",
		consumer.WithConsumerTag("example-consumer"),    // set up custom consumer tag
		consumer.WithArgs(amqp.Table{"example": "tag"}), // add additional consumer tags
		consumer.WithAutoAck(),                          // automatically ACK all messages
		consumer.WithExclusive(),                        // tell the server to ensure that this is the sole consumer from this queue
		consumer.WithNoWait(),                           //  tell the server to immediately begin deliveries
	)
	// see also examples/one and examples/batch for examples of prepared ProcessFunc.
	err = cons.Start(context.Background(), consumer.ProcessFunc(func(ctx context.Context, deliveries <-chan amqp.Delivery) error {
		for d := range deliveries {
			log.Println(d.Body)
		}
		return nil
	}))
	if err != nil {
		log.Panic(err)
	}

	time.Sleep(10 * time.Second) // consumer for 10 seconds
	if err := cons.Stop(); err != nil {
		log.Panic(err)
	}
}
