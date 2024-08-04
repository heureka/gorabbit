package gorabbit

import (
	"context"
	"sync"

	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultLocale = "en_US"
)

// Client is a RabbitMQ client.
type Client struct {
	mux              sync.RWMutex
	connection       *amqp.Connection
	channel          *amqp.Channel
	channelClosed    chan *amqp.Error
	connectionClosed chan *amqp.Error

	bo                 backoff.BackOff
	dialConfig         amqp.Config
	onConnectionClosed []func(error)
	onChannelClosed    []func(error)
	done               chan struct{}
	err                error
}

// New creates a new RabbitMQ client.
func New(addr string, ops ...Option) *Client {
	client := Client{
		bo: backoff.NewExponentialBackOff(),
		dialConfig: amqp.Config{
			// defaults are the same as in amqp091-go:
			// https://github.com/rabbitmq/amqp091-go/blob/ddb7a2f0685689063e6d709b8e417dbf9d09469c/connection.go#L158
			Locale: defaultLocale,
		},
		done: make(chan struct{}),
	}

	for _, op := range ops {
		op(&client)
	}

	client.init(addr)

	go client.reconnect(addr)

	return &client
}

// Consume consumes messages from a queue into deliveries channel.
func (c *Client) Consume(ctx context.Context, queue string, prefetch int, deliveries chan<- amqp.Delivery) error {
	return c.Iter(ctx, queue, prefetch)(func(delivery amqp.Delivery) bool {
		deliveries <- delivery
		return true
	})
}

// Iter iterates over messages from a queue.
func (c *Client) Iter(ctx context.Context, queue string, prefetch int) func(yield func(delivery amqp.Delivery) bool) error {
	return func(yield func(delivery amqp.Delivery) bool) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c.done:
				return c.err
			default:
			}

			deliveries, err := c.deliveries(ctx, prefetch, queue)
			if err != nil {
				continue
			}

			for del := range deliveries {
				if !yield(del) {
					return nil
				}
			}
		}
	}
}

func (c *Client) deliveries(ctx context.Context, prefetch int, queue string) (<-chan amqp.Delivery, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if err := c.channel.Qos(prefetch, 0, false); err != nil {
		return nil, err
	}

	return c.channel.ConsumeWithContext(ctx, queue, "", false, false, false, false, nil)
}

// Publish publishes a message to an exchange.
func (c *Client) Publish(ctx context.Context, exchange, key string, pub amqp.Publishing) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.done:
			return c.err
		default:
		}

		if err := c.publish(ctx, exchange, key, pub); err != nil {
			continue
		}

		return nil
	}
}

func (c *Client) publish(ctx context.Context, exchange, key string, pub amqp.Publishing) error {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return c.channel.PublishWithContext(ctx, exchange, key, false, false, pub)
}

// Channel returns the current channel.
// Please note that the channel can be closed and Client will re-open another one and use it,
// but this one will remain closed.
func (c *Client) Channel() *amqp.Channel {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return c.channel
}

// IsLive returns true if connection and channel are opened.
func (c *Client) IsLive() bool {
	select {
	case <-c.done:
		return false
	default:

	}

	c.mux.RLock()
	defer c.mux.RUnlock()

	return !c.channel.IsClosed() && !c.connection.IsClosed()
}

// IsReady returns true if connection and channel are opened and ready to use.
func (c *Client) IsReady() bool {
	select {
	case <-c.done:
		return false
	default:
	}

	ready := c.mux.TryLock()
	defer c.mux.Unlock()

	return ready
}

func (c *Client) fatal(err error) {
	select {
	case <-c.done: // already closed
		return
	default:
	}

	close(c.done)
	c.err = err
}

func (c *Client) reconnect(addr string) {
	for {
		select {
		case <-c.done:
			return
		case err := <-c.connectionClosed:
			c.notifyConnectionClosed(err)
			c.init(addr)
		case err := <-c.channelClosed:
			c.notifyChannelClosed(err)
			if c.connection.IsClosed() {
				continue // connection closed, reconnect
			}

			c.mux.Lock()
			if err := c.openChannel(); err != nil {
				c.mux.Unlock()
				c.fatal(err)
				return
			}
			c.mux.Unlock()
		}
	}
}

func (c *Client) init(addr string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if err := c.dial(addr); err != nil {
		c.fatal(err)
		return
	}
	if err := c.openChannel(); err != nil {
		c.fatal(err)
		return
	}
}

func (c *Client) dial(addr string) error {
	operation := func() error {
		select {
		case <-c.done:
			return backoff.Permanent(c.err)
		default:
		}

		conn, err := amqp.DialConfig(addr, c.dialConfig)
		if err != nil {
			return err
		}

		c.connectionClosed = conn.NotifyClose(make(chan *amqp.Error, 1))
		c.connection = conn

		return nil
	}

	return backoff.Retry(operation, c.bo)
}

func (c *Client) openChannel() error {
	operation := func() error {
		select {
		case <-c.done:
			return backoff.Permanent(c.err)
		default:
		}

		ch, err := c.connection.Channel()
		if err != nil {
			return err
		}

		c.channelClosed = ch.NotifyClose(make(chan *amqp.Error, 1))
		c.channel = ch

		return nil
	}

	return backoff.Retry(operation, c.bo)
}

func (c *Client) notifyConnectionClosed(err error) {
	for _, f := range c.onConnectionClosed {
		f(err)
	}
}

func (c *Client) notifyChannelClosed(err error) {
	for _, f := range c.onChannelClosed {
		f(err)
	}
}

// Close closes the connection and channel.
func (c *Client) Close() error {
	select {
	case <-c.done:
		// already closed
		return nil
	default:
	}

	close(c.done)

	c.mux.Lock()
	defer c.mux.Unlock()

	if err := c.channel.Close(); err != nil {
		return err
	}
	if err := c.connection.Close(); err != nil {
		return err
	}

	return nil
}
