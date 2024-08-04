package gorabbit

import (
	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Option is a configuration option for the client.
type Option func(c *Client)

// WithDialConfig allows to configure the dial config.
func WithDialConfig(fns ...func(c *amqp.Config)) Option {
	return func(c *Client) {
		for _, fn := range fns {
			fn(&c.dialConfig)
		}
	}
}

// WithBackoff sets the backoff strategy for reconnecting.
func WithBackoff(bo backoff.BackOff) Option {
	return func(c *Client) {
		c.bo = bo
	}
}

// WithOnConnectionClosed sets the function to be called when the connection is closed.
func WithOnConnectionClosed(fn func(error)) Option {
	return func(c *Client) {
		c.onConnectionClosed = append(c.onConnectionClosed, fn)
	}
}

// WithOnChannelClosed sets the function to be called when the channel is closed.
func WithOnChannelClosed(fn func(error)) Option {
	return func(c *Client) {
		c.onChannelClosed = append(c.onChannelClosed, fn)
	}
}
