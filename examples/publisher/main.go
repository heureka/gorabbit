package main

import (
	"context"
	"log"
	"time"

	"github.com/heureka/gorabbit"
	"github.com/heureka/gorabbit/publisher"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	pub, err := gorabbit.NewPublisher(
		"amqp://localhost:5672",
		"example-exchange",
		publisher.WithHeaders(amqp.Table{"x-example-header": "example-value"}),
		publisher.WithTransientDeliveryMode(),
		publisher.WithImmediate(),
		publisher.WithMandatory(),
		publisher.WithExpiration(time.Minute),
	)
	if err != nil {
		log.Panic(err)
	}
	// publish
	err = pub.Publish(context.Background(), "example-key", []byte("hello world!"))
	if err != nil {
		log.Panic(err)
	}

	// override config only for this publishing
	err = pub.Publish(
		context.Background(),
		"example-key",
		[]byte("hello again!"),
		// add headers
		publisher.WithHeaders(amqp.Table{"x-other-header": "only for thing publishing"}),
		// set another expiration
		publisher.WithExpiration(time.Hour),
		// custom overriding
		func(channel publisher.Channel) publisher.Channel {
			return publisher.ChannelFunc(func(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
				// set another exchange only for this publishing
				return channel.PublishWithContext(ctx, "other-exchange", key, mandatory, immediate, msg)
			})
		},
	)
	if err != nil {
		log.Panic(err)
	}
}
