package one

import (
	"context"
	"log"
	"time"

	"github.com/heureka/gorabbit"
	"github.com/heureka/gorabbit/channel"
	"github.com/heureka/gorabbit/connection"
	"github.com/heureka/gorabbit/process"
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

	consumer := gorabbit.NewConsumer(ch, "my-queue")
	// transaction function for processing single message
	tx := func(ctx context.Context, msg []byte) error {
		log.Println(string(msg))
		return nil
	}

	err = consumer.Start(context.Background(), process.ByOne(tx, false))
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
