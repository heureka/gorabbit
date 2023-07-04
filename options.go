package gorabbit

import (
	"github.com/heureka/gorabbit/channel"
)

// Option allows to configure RabbitMQ Consumer.
type Option func(c *config)

// WithChannelQOS sets channel's Quality of Service. Will reset QOS on reconnection.
// Please refer to https://www.rabbitmq.com/confirms.html#channel-qos-prefetch.
func WithChannelQOS(prefetchCount, prefetchSize int, global bool) Option {
	return func(c *config) {
		c.qos.prefetchCount = prefetchCount
		c.qos.prefetchSize = prefetchSize
		c.qos.global = global
	}
}

// WithReconnector configures channel.Reconnector used in consumer.
func WithReconnector(ops ...channel.Option) Option {
	return func(c *config) {
		c.channelOps = append(c.channelOps, ops...)
	}
}

// ConsumeOption allows to configure RabbitMQ consume options.
type ConsumeOption func(c *consumeCfg)

// WithConsume sets up consuming configuration.
// Please refer to https://pkg.go.dev/github.com/rabbitmq/amqp091-go?utm_source=godoc#Channel.Consume.
func WithConsume(ops ...ConsumeOption) Option {
	return func(c *config) {
		for _, op := range ops {
			op(&c.consume)
		}
	}
}

// WithConsumerTag sets consumer consumerTag. Otherwise library will generate a unique identity.
func WithConsumerTag(tag string) ConsumeOption {
	return func(c *consumeCfg) {
		c.tag = tag
	}
}

// WithConsumeAutoAck sets the server to acknowledge deliveries to this consumer
// prior to writing the delivery to the network.
func WithConsumeAutoAck() ConsumeOption {
	return func(c *consumeCfg) {
		c.autoAck = true
	}
}

// WithConsumeExclusive sets the server to ensure that this is the sole consumer from this queue.
func WithConsumeExclusive() ConsumeOption {
	return func(c *consumeCfg) {
		c.exclusive = true
	}
}

// WithConsumeNoWait sets the server to not wait to confirm the request
// and immediately begin deliveries.
func WithConsumeNoWait() ConsumeOption {
	return func(c *consumeCfg) {
		c.noWait = true
	}
}

// WithConsumeArgs sets additional arguments for consuming.
func WithConsumeArgs(args map[string]interface{}) ConsumeOption {
	return func(c *consumeCfg) {
		c.args = args
	}
}
