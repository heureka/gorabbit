package publisher

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Channel is a RabbitMQ channel for publishing.
type Channel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

// ChannelFunc type is an adapter to allow the use of
// ordinary functions as Channel.
type ChannelFunc func(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error

// PublishWithContext implements Channel.
func (f ChannelFunc) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return f(ctx, exchange, key, mandatory, immediate, msg)
}

// Wrap channel with middlewares.
//
//nolint:ireturn // it is OK for a wrapper to return interface.
func Wrap(channel Channel, mws ...Middleware) Channel {
	// apply in reverse order because we are wrapping
	for i := len(mws) - 1; i >= 0; i-- {
		channel = mws[i](channel)
	}

	return channel
}

// Middleware adds new functionality for publishing.
type Middleware func(Channel) Channel
