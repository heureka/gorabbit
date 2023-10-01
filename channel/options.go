package channel

import (
	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

// WithCreateCallback sets callback function on channel creation.
// Given function will receive a newly created channel.
// Could be used to set up QoS or listeners for various evens (NotifyClose, NotifyFlow, etc.) after channel recreation.
func WithCreateCallback(fn func(channel *amqp.Channel) error) Option {
	return func(r *Reconnector) {
		r.onCreate = append(r.onCreate, fn)
	}
}

// WithQOS creates new connection callback which sets channel's QOS on each channel creation.
func WithQOS(prefetchCount, prefetchSize int, global bool) Option {
	return WithCreateCallback(func(channel *amqp.Channel) error {
		return channel.Qos(prefetchCount, prefetchSize, global)
	})
}

// WithBackoff sets backoff function for reconnection.
func WithBackoff(bo backoff.BackOff) Option {
	return func(r *Reconnector) {
		r.backoff = bo
	}
}
