package consumer

import (
	"context"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer is Consumer for RabbiMQ.
// Will automatically recreate channel on channel errors.
// Reconnection is done with exponential backoff.
type Consumer struct {
	channel    Channel
	queueName  string
	consumeCfg consumeCfg

	done sync.WaitGroup
}

// Channel is a RabbitMQ channel opened for consuming deliveries.
type Channel interface {
	Consume(queue string, consumer string, autoAck bool, exclusive bool, noLocal bool, noWait bool, args amqp.Table) <-chan amqp.Delivery
	Cancel(consumer string, noWait bool) error
	Close() error
}

// New creates new RabbitMQ Consumer.
// By default, it will consume with autoAck=false, exclusive=false, noWait=false, and empty args.
// Pass Options to configure it as you wish.
//
// An empty consumer name will cause the library to generate a unique identity.
// An empty queue name will cause the broker to generate a unique name https://www.rabbitmq.com/queues.html#server-named-queues.
func New(channel Channel, queue string, ops ...Option) *Consumer {
	cfg := consumeCfg{
		tag:       "", // amqp will generate unique ID if not set
		autoAck:   false,
		exclusive: false, // the server will fairly distribute deliveries across multiple consumers.
		noWait:    false,
		args:      map[string]interface{}{},
	}

	for _, op := range ops {
		op(&cfg)
	}

	return &Consumer{
		channel:    channel,
		queueName:  queue,
		consumeCfg: cfg,
	}
}

// consumeCfg configuration.
type consumeCfg struct {
	tag       string
	autoAck   bool
	exclusive bool
	noWait    bool
	args      amqp.Table
}

// Processor consumes all provided deliveries.
type Processor interface {
	Process(ctx context.Context, deliveries <-chan amqp.Delivery) error
}

// ProcessFunc type is an adapter to allow the use of
// ordinary functions as Processor.
type ProcessFunc func(ctx context.Context, deliveries <-chan amqp.Delivery) error

// Process implements Processor.
func (f ProcessFunc) Process(ctx context.Context, deliveries <-chan amqp.Delivery) error {
	return f(ctx, deliveries)
}

// Start consuming messages and pass them to Processor.
// If autoAck is not set, will Reject messages if Processor returns error, otherwise Ack them.
// Call Stop to stop consuming.
func (c *Consumer) Start(ctx context.Context, processor Processor) error {
	deliveries := c.channel.Consume(
		c.queueName,
		c.consumeCfg.tag,
		c.consumeCfg.autoAck,
		c.consumeCfg.exclusive,
		false, // noLocal is not supported by RabbitMQ
		c.consumeCfg.noWait,
		c.consumeCfg.args,
	)

	if c.consumeCfg.tag == "" {
		deliveries = consumerTagProxy(&c.consumeCfg.tag, deliveries)
	}
	if c.consumeCfg.autoAck {
		deliveries = acknowledgerProxy(ackIgnorer{}, deliveries)
	}

	c.done.Add(1)
	defer c.done.Done() // done when .Process is unblocked

	return processor.Process(ctx, deliveries)
}

// Stop consuming, wait for all in-flight messages to be processed and close a channel.
func (c *Consumer) Stop() error {
	// with noWait == true potentially could drop deliveries in-flight
	if err := c.channel.Cancel(c.consumeCfg.tag, false); err != nil {
		return fmt.Errorf("cancel consuming: %w", err)
	}

	// wait for consuming to stop
	c.done.Wait()

	if err := c.channel.Close(); err != nil {
		return fmt.Errorf("close channel: %w", err)
	}

	return nil
}

// consumerTagProxy sets consumerTag from deliveries.
func consumerTagProxy(consumerTag *string, deliveries <-chan amqp.Delivery) <-chan amqp.Delivery {
	proxy := make(chan amqp.Delivery)
	var once sync.Once

	go func() {
		defer close(proxy)

		for d := range deliveries {
			once.Do(func() {
				*consumerTag = d.ConsumerTag
			})

			proxy <- d
		}
	}()

	return proxy
}

// acknowledgerProxy sets amqp.Acknowledger to all deliveries.
func acknowledgerProxy(acker amqp.Acknowledger, deliveries <-chan amqp.Delivery) <-chan amqp.Delivery {
	proxy := make(chan amqp.Delivery)

	go func() {
		defer close(proxy)

		for d := range deliveries {
			d.Acknowledger = acker
			proxy <- d
		}
	}()

	return proxy
}

// ackIgnorer is amqp.Acknowledger which ignores acknowledge calls.
// Useful to avoid calls when autoAck is set.
type ackIgnorer struct{}

// Ack acknowledges a message.
func (a ackIgnorer) Ack(uint64, bool) error {
	return nil
}

// Nack negatively acknowledges a message.
func (a ackIgnorer) Nack(uint64, bool, bool) error {
	return nil
}

// Reject rejects a message.
func (a ackIgnorer) Reject(uint64, bool) error {
	return nil
}
