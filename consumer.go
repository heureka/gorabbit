package rabbit

import (
	"context"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"

	channel2 "github.com/heureka/gorabbit/channel"
)

// Consumer is Consumer for RabbiMQ.
// Will automatically recreate channel on channel errors.
// Reconnection is done with exponential backoff.
type Consumer struct {
	channel       *channel2.Reconnector
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
	ch, err := channel2.New(
		conn,
		channel2.WithBackoff(cfg.backoff),
		channel2.WithReconnectionCallback(
			func(ch *amqp.Channel) error {
				return ch.Qos(cfg.qos.prefetchCount, cfg.qos.prefetchSize, cfg.qos.global)
			}),
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
func (r *Consumer) Start(ctx context.Context, consumer Transactor) error {
	deliveries := r.channel.ConsumeReconn(
		r.queueName,
		r.consumeCfg.tag,
		r.consumeCfg.autoAck,
		r.consumeCfg.exclusive,
		false, // noLocal is not supported by RabbitMQ
		r.consumeCfg.noWait,
		r.consumeCfg.args,
	)

	deliveries = consumerTagSetter(&r.consumeCfg.tag, deliveries)
	if r.consumeCfg.autoAck {
		deliveries = ignoreAck(deliveries)
	}

	r.done = make(chan struct{})
	defer close(r.done)

	return consumer.Consume(ctx, deliveries)
}

// Stop consuming, wait for all in-flight messages to be processed and close a channel.
func (r *Consumer) Stop() error {
	// if noWait == true - potentially could drop deliveries in-flight
	if err := r.channel.Cancel(r.consumeCfg.tag, false); err != nil {
		return fmt.Errorf("cancel consuming: %w", err)
	}

	// wait for consuming to stop
	if r.done != nil {
		<-r.done
	}

	if err := r.channel.Close(); err != nil {
		return fmt.Errorf("close channel: %w", err)
	}

	return nil
}

// consumerTagSetter sets consumerTag from deliveries.
func consumerTagSetter(consumerTag *string, deliveries <-chan amqp.Delivery) <-chan amqp.Delivery {
	proxy := make(chan amqp.Delivery)

	go func() {
		defer close(proxy)

		isSet := *consumerTag != ""
		for d := range deliveries {
			if !isSet { // if consumerTag wasn't set, set consumerTag to generated one
				*consumerTag = d.ConsumerTag
			}

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
