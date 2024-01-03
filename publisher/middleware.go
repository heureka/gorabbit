package publisher

import (
	"context"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// WithHeaders adds headers to the published message.
func WithHeaders(table amqp.Table) Middleware {
	return func(channel Channel) Channel {
		return ChannelFunc(func(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
			if msg.Headers == nil {
				msg.Headers = make(amqp.Table)
			}
			for k, v := range table {
				msg.Headers[k] = v
			}

			return channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
		})
	}
}

// WithExpiration sets publishing Expire property.
func WithExpiration(expire time.Duration) Middleware {
	return func(channel Channel) Channel {
		return ChannelFunc(func(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
			msg.Expiration = strconv.FormatInt(expire.Milliseconds(), 10)

			return channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
		})
	}
}

// WithTransientDeliveryMode sets publishing to the Transient delivery mode.
// Transient means higher throughput but messages will not be
// restored on broker restart.
// See https://github.com/rabbitmq/amqp091-go/blob/main/types.go#L123.
func WithTransientDeliveryMode() Middleware {
	return func(channel Channel) Channel {
		return ChannelFunc(func(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
			msg.DeliveryMode = amqp.Transient

			return channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
		})
	}
}

// WithMandatory sets server to discard a message if no queue is
// bound that matches the routing key and server will return an
// undeliverable message with a Return method.
// See https://www.rabbitmq.com/amqp-0-9-1-reference.html#basic.publish.mandatory.
func WithMandatory() Middleware {
	return func(channel Channel) Channel {
		return ChannelFunc(func(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {

			return channel.PublishWithContext(ctx, exchange, key, true, immediate, msg)
		})
	}
}

// WithImmediate sets server to discard a message when
// no consumer on the matched queue is ready to accept the delivery
// and server will return an undeliverable message with a Return method.
// See https://www.rabbitmq.com/amqp-0-9-1-reference.html#basic.publish.immediate.
func WithImmediate() Middleware {
	return func(channel Channel) Channel {
		return ChannelFunc(func(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {

			return channel.PublishWithContext(ctx, exchange, key, mandatory, true, msg)
		})
	}
}
