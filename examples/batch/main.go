package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/heureka/gorabbit"
	"github.com/heureka/gorabbit/consume"
	"github.com/heureka/gorabbit/consume/batch"
	"github.com/rs/zerolog"
)

func main() {
	consumer, err := gorabbit.NewConsumer("amqp://localhost:5672", "my-queue")
	if err != nil {
		log.Panic(err)
	}

	// transaction function for processing batch of messages
	tx := func(ctx context.Context, msgs [][]byte) []error {
		for _, msg := range msgs {
			log.Println(string(msg))
		}
		// must return errors one-to-one for each processed message, no errors in this case
		return make([]error, len(msgs))
	}

	// process 100 messages or 1 second of messages at once, whichever comes first
	err = consumer.Start(context.Background(), consume.InBatches(100, time.Second, tx, false, batch.NewDeliveryLogging(zerolog.New(os.Stdout))))
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
