package publisher

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher is a Publisher to RabbiMQ.
type Publisher struct {
	channel     Channel
	middlewares []Middleware
	exchange    string
}

// New creates new RabbitMQ Publisher.
// By default, it will publish with Persistent delivery mode, mandatory=false, immediate=false and empty args.
// Pass Options to configure it as you wish.
func New(channel Channel, exchange string, mws ...Middleware) Publisher {
	return Publisher{
		channel:     channel,
		middlewares: mws,
		exchange:    exchange,
	}
}

// Publish message with routing key. Allows to override middleware for one publishing.
func (p Publisher) Publish(ctx context.Context, key string, message []byte, mws ...Middleware) error {
	publishing := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Body:         message,
	}
	channel := Wrap(p.channel, append(p.middlewares, mws...)...)

	return channel.PublishWithContext(ctx, p.exchange, key, false, false, publishing)
}
