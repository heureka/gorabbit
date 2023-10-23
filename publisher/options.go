package publisher

import (
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Option allows to configure RabbitMQ Publisher.
type Option func(p *Publisher)

// WithConstHeaders sets constant publishing headers, that will be added to all publishings.
func WithConstHeaders(headers amqp.Table) Option {
	return func(p *Publisher) {
		if p.headers == nil {
			p.headers = make(amqp.Table)
		}

		for k, v := range headers {
			p.headers[k] = v
		}
	}
}

// WithTransientDeliveryMode sets publishing to the Transient delivery mode.
// Transient means higher throughput but messages will not be
// restored on broker restart.
// See https://github.com/rabbitmq/amqp091-go/blob/main/types.go#L123.
func WithTransientDeliveryMode() Option {
	return func(p *Publisher) {
		p.deliveryMode = amqp.Transient
	}
}

// WithMandatory sets server to discard a message if no queue is
// bound that matches the routing key and server will return an
// undeliverable message with a Return method.
// See https://www.rabbitmq.com/amqp-0-9-1-reference.html#basic.publish.mandatory.
func WithMandatory() Option {
	return func(c *Publisher) {
		c.mandatory = true
	}
}

// WithImmediate sets server to discard a message when
// no consumer on the matched queue is ready to accept the delivery
// and server will return an undeliverable message with a Return method.
// See https://www.rabbitmq.com/amqp-0-9-1-reference.html#basic.publish.immediate.
func WithImmediate() Option {
	return func(p *Publisher) {
		p.immediate = true
	}
}

// WithExpiration sets publishing Expire property.
func WithExpiration(expire time.Duration) Option {
	return func(c *Publisher) {
		c.expiration = strconv.FormatInt(expire.Milliseconds(), 10)
	}
}
