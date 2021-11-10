package channel

import (
	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

// WithReconnectionCallback sets callback function on reconnection.
// Given function will receive a newly created channel.
// Could be used to set up QoS or listeners for various evens (NotifyClose, NotifyFlow, etc.) after channel recreation.
func WithReconnectionCallback(fn func(channel *amqp.Channel) error) Option {
	return func(r *Reconnector) {
		r.onReconnect = append(r.onReconnect, fn)
	}
}

// WithBackoff sets backoff function for reconnection.
func WithBackoff(bo backoff.BackOff) Option {
	return func(r *Reconnector) {
		r.backoff = bo
	}
}
