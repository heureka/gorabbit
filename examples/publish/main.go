package main

import (
	"context"
	"log"

	"github.com/heureka/gorabbit/channel"
	"github.com/heureka/gorabbit/connection"
	"github.com/heureka/gorabbit/publish"
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

	p := publish.Wrap(ch, publish.WithHeaders(amqp.Table{"x-example-header": "example-value"}))
	err = p.PublishWithContext(
		context.Background(),
		"example-exchange",
		"example-key",
		false,
		false,
		amqp.Publishing{Body: []byte("hello world!")},
	)
	if err != nil {
		log.Panic(err)
	}
}
