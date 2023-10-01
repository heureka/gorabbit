package main

import (
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/heureka/gorabbit/channel"
	"github.com/heureka/gorabbit/connection"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn, err := connection.Dial("amqp://localhost:5672")
	if err != nil {
		log.Panic(err)
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 10 * time.Second

	// create new channel with re-creation capabilities.
	ch, err := channel.New(
		conn,
		// set up different backoff strategy.
		channel.WithBackoff(bo),
		// set QOS to pre-fetch 100 messages, it will be applied every time channel is created.
		channel.WithQOS(100, 0, false),
		// configure channel as you wish, will be applied on each successful creation of a channel.
		channel.WithOpenCallback(func(ch *amqp.Channel) error {
			cancelNotif := ch.NotifyCancel(make(chan string))
			go func() {
				for tag := range cancelNotif {
					log.Println("got cancel notification", tag)
				}
			}()

			return nil
		}),
	)
	if err != nil {
		log.Panic(err)
	}

	reopenNotif := ch.NotifyReopen(make(chan error)) // reopenNotif will be closed on graceful shutdown
	go func() {
		for err := range reopenNotif {
			if err != nil {
				log.Println("can't re-open channel", err)
				return
			}

			log.Println("successfully re-opened channel", err)
		}
	}()

	consumeNotif := ch.NotifyConsume(make(chan error)) // consumeNotif will be closed on graceful shutdown
	go func() {
		for err := range consumeNotif {
			if err != nil {
				log.Println("can't start consuming", err)
				return
			}

			log.Println("successfully started consuming", err)
		}
	}()

	deliveries := ch.Consume("example-queue", "", false, false, false, false, nil)
	go func() {
		for d := range deliveries {
			// do something with deliveries
			log.Println(d.Body)
		}
	}()

	time.Sleep(10 * time.Second) // consume for 10 seconds
	if err := ch.Close(); err != nil {
		log.Panic(err)
	}
}
