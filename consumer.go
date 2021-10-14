package rabbit

import (
	"context"
	"fmt"
	"sync"

	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/heureka/gorabbit/channel"
)

// Consumer is Consumer for RabbiMQ.
// Will automatically recreate channel on channel errors.
// Reconnection is done with exponential backoff.
type Consumer struct {
	channel       *channel.Reconnector
	queueName     string
	rejectRequeue bool
	consumeCfg    consumeCfg

	done chan struct{}
}

type config struct {
	consume consumeCfg
	backoff backoff.BackOff
	qos     channelQOS
}

// NewConsumer creates new RabbitMQ Consumer.
// Default configuration will create a durable queue and is ready for competing consumers.
//
// An empty consumer name will cause the library to generate a unique identity.
// An empty queue name will cause the broker to generate a unique name https://www.rabbitmq.com/queues.html#server-named-queues.
func NewConsumer(conn *amqp.Connection, queue string, ops ...ConsumerOption) (*Consumer, error) {
	cfg := config{
		consume: consumeCfg{
			tag:       "", // amqp will generate unique ID if not set
			autoAck:   false,
			exclusive: false, // the server will fairly distribute deliveries across multiple consumers.
			noWait:    false,
			args:      map[string]interface{}{},
		},
		backoff: backoff.NewExponentialBackOff(),
		qos: channelQOS{
			prefetchCount: 1,
			prefetchSize:  0,
			global:        false,
		},
	}

	for _, op := range ops {
		op(&cfg)
	}
	ch, err := channel.New(
		conn,
		append(
			cfg.channelOps,
			channel.WithReconnectionCallback(newReconnCallback(cfg.qos)),
		)...,
	)

	if err != nil {
		return nil, fmt.Errorf("channel creation: %w", err)
	}

	if err := ch.Qos(cfg.qos.prefetchCount, cfg.qos.prefetchSize, cfg.qos.global); err != nil {
		return nil, fmt.Errorf("set channel QOS: %w", err)
	}

	return &Consumer{
		channel:       ch,
		queueName:     queue,
		rejectRequeue: true,
		consumeCfg:    cfg.consume,
		done:          nil,
	}, nil
}

func newReconnCallback(qos channelQOS) func(*amqp.Channel) error {
	return func(ch *amqp.Channel) error {
		return ch.Qos(qos.prefetchCount, qos.prefetchSize, qos.global)
	}
}

// consumeCfg configuration.
type consumeCfg struct {
	tag       string
	autoAck   bool
	exclusive bool
	noWait    bool
	args      map[string]interface{}
}

// ChannelQOS is channel's Quality of Service configuration.
// Please refer to https://www.rabbitmq.com/consumer-prefetch.html.
type channelQOS struct {
	prefetchCount int
	prefetchSize  int
	global        bool
}

// Transactor consumes all provided deliveries.
type Transactor interface {
	Consume(ctx context.Context, deliveries <-chan amqp.Delivery) error
}

// Start consuming messages and pass them to Transaction.
// If autoAck is not set, will Reject messages if Transaction returns error, otherwise Ack them.
// Call Stop to stop consuming.
// Returns channel with reading errors, channel MUST be read.
func (c *Consumer) Start(ctx context.Context, consumer Transactor) error {
	deliveries := c.channel.ConsumeReconn(
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

	c.done = make(chan struct{})
	defer close(c.done)

	return consumer.Consume(ctx, deliveries)
}

// Stop consuming, wait for all in-flight messages to be processed and close a channel.
func (c *Consumer) Stop() error {
	// with noWait == true potentially could drop deliveries in-flight
	if err := c.channel.Cancel(c.consumeCfg.tag, false); err != nil {
		return fmt.Errorf("cancel consuming: %w", err)
	}

	// wait for consuming to stop
	if c.done != nil {
		<-c.done
	}

	if err := c.channel.Close(); err != nil {
		return fmt.Errorf("close channel: %w", err)
	}

	return nil
}

// consumerTagSetter sets consumerTag from deliveries.
func consumerTagSetter(consumerTag *string, deliveries <-chan amqp.Delivery) <-chan amqp.Delivery {
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

// ignoreAck sets ackIgnorer as amqp.Acknowledger to all deliveries.
// Useful to avoid calls when autoAck is set.
func ignoreAck(deliveries <-chan amqp.Delivery) <-chan amqp.Delivery {
	proxy := make(chan amqp.Delivery)

	go func() {
		defer close(proxy)

		for d := range deliveries {
			d.Acknowledger = ackIgnorer{}
			proxy <- d
		}
	}()

	return proxy
}

// ackIgnorer is amqp.Acknowledger which ignores acknowledge calls.
type ackIgnorer struct{}

func (a ackIgnorer) Ack(tag uint64, multiple bool) error {
	return nil
}

func (a ackIgnorer) Nack(tag uint64, multiple bool, requeue bool) error {
	return nil
}

func (a ackIgnorer) Reject(tag uint64, requeue bool) error {
	return nil
}