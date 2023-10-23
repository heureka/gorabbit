package one

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/heureka/gorabbit"
	"github.com/heureka/gorabbit/consume"
	"github.com/heureka/gorabbit/consume/one"
	"github.com/rs/zerolog"
)

func main() {
	consumer, err := gorabbit.NewConsumer("amqp://localhost:5672", "my-queue")
	if err != nil {
		log.Panic(err)
	}
	// transaction function for processing single message
	tx := func(ctx context.Context, msg []byte) error {
		log.Println(string(msg))
		return nil
	}

	err = consumer.Start(context.Background(), consume.ByOne(tx, false, one.NewDeliveryLogging(zerolog.New(os.Stdout))))
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
