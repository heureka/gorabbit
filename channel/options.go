package channel

import (
	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

// WithNotifyErrors registers a listener for any reconnection problems.
// Will send errors appeared during reconnection process to provided channel.
func WithNotifyErrors(ch chan<- error) Option {
	return func(r *Reconnector) {
		r.reconnectErrors = ch
	}
}

// WithReconnectionCallback sets callback function on reconnection.
// Given function will receive a newly created channel.
// Could be used to set up QoS or listeners for various evens (NotifyClose, NotifyFlow, etc.) after channel recreation.
func WithReconnectionCallback(fn func(channel *amqp.Channel) error) Option {
	return func(r *Reconnector) {
		r.onReconnect = fn
	}
}

// WithBackoff sets backoff function for reconnection.
func WithBackoff(bo backoff.BackOff) Option {
	return func(r *Reconnector) {
		r.backoff = bo
	}
}
