package main

import (
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/heureka/gorabbit/connection"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 10 * time.Second

	// create new connection with re-dialing capabilities.
	conn, err := connection.Dial(
		"amqp://localhost:5672",
		// set up different backoff strategy.
		connection.WithBackoff(bo),
		// provide different then default dialing config.
		connection.WithConfig(amqp.Config{
			Heartbeat: 10 * time.Second, // e.g. set different heartbeat interval
		}),
		// log dial attempts
		connection.WithDialAttemptCallback(func(err error) {
			if err != nil {
				log.Println("can't dial rabbitmq", err)
				return
			}
			log.Println("successfully dial rabbitmq")
		}),
		// configure connection as you wish on each successful dial.
		connection.WithDialledCallback(func(conn *amqp.Connection) {
			// add listener for clos notification
			closeNotif := conn.NotifyClose(make(chan *amqp.Error))
			go func() {
				for err := range closeNotif {
					log.Println("connection got closed", err)
				}
			}()
		}),
	)
	if err != nil {
		log.Panic(err)
	}

	conn.Channel() // crate new channel, do whatever you do with connection.
}
