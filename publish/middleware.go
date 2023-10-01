package publish

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
