package publisher

import (
	"context"

	"github.com/heureka/gorabbit/publish"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher is a Publisher to RabbiMQ.
type Publisher struct {
	channel      publish.Channel
	exchange     string
	headers      amqp.Table
	deliveryMode uint8
	mandatory    bool
	immediate    bool
	expiration   string
}

// New creates new RabbitMQ Publisher.
// By default, it will publish with Persistent delivery mode, mandatory=false, immediate=false and empty args.
// Pass Options to configure it as you wish.
func New(channel publish.Channel, exchange string, ops ...Option) Publisher {
	pub := Publisher{
		channel:      channel,
		exchange:     exchange,
		headers:      nil,
		deliveryMode: amqp.Persistent,
		mandatory:    false,
		immediate:    false,
		expiration:   "",
	}

	for _, op := range ops {
		op(&pub)
	}

	return pub
}

// Publish message with routing key.
func (p Publisher) Publish(ctx context.Context, key string, message []byte, mws ...publish.Middleware) error {
	publishing := amqp.Publishing{
		Headers:      p.headers,
		DeliveryMode: p.deliveryMode,
		Expiration:   p.expiration,
		Body:         message,
	}
	channel := publish.Wrap(p.channel, mws...)

	return channel.PublishWithContext(ctx, p.exchange, key, p.mandatory, p.immediate, publishing)

}
