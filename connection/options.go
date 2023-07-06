package connection

import (
	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

// WithDialledCallback registers a callback for successful AMQP dial.
// Will call func with each created connection.
// Could be used to set up listeners for various evens (NotifyClose, NotifyFlow, etc.) after channel recreation.
func WithDialledCallback(fn func(*amqp.Connection)) Option {
	return func(r *Redialer) {
		r.onDialled = append(r.onDialled, fn)
	}
}

// WithDialAttemptCallback registers a callback for any AMQP dial attempt result.
// Will call func with result for each attempt to dial AMQP.
func WithDialAttemptCallback(fn func(error)) Option {
	return func(r *Redialer) {
		r.onAttempt = append(r.onAttempt, fn)
	}
}

// WithBackoff sets backoff function for reconnection.
func WithBackoff(bo backoff.BackOff) Option {
	return func(r *Redialer) {
		r.backoff = bo
	}
}
