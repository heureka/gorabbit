package gorabbit

import amqp "github.com/rabbitmq/amqp091-go"

// Option allows to configure RabbitMQ Consumer.
// Please refer to https://pkg.go.dev/github.com/rabbitmq/amqp091-go?utm_source=godoc#Channel.Consume.
type Option func(c *consumeCfg)

// WithConsumerTag sets consumer consumerTag. Otherwise, library will generate a unique identity.
func WithConsumerTag(tag string) Option {
	return func(c *consumeCfg) {
		c.tag = tag
	}
}

// WithConsumeAutoAck sets the server to acknowledge deliveries to this consumer
// prior to writing the delivery to the network.
func WithConsumeAutoAck() Option {
	return func(c *consumeCfg) {
		c.autoAck = true
	}
}

// WithConsumeExclusive sets the server to ensure that this is the sole consumer from this queue.
func WithConsumeExclusive() Option {
	return func(c *consumeCfg) {
		c.exclusive = true
	}
}

// WithConsumeNoWait sets the server to not wait to confirm the request
// and immediately begin deliveries.
func WithConsumeNoWait() Option {
	return func(c *consumeCfg) {
		c.noWait = true
	}
}

// WithConsumeArgs sets additional arguments for consuming.
func WithConsumeArgs(args amqp.Table) Option {
	return func(c *consumeCfg) {
		c.args = args
	}
}
