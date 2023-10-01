package channel

import (
	"context"
	"fmt"
	"sync"

	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Reconnector wraps channel to add reconnection capabilities.
// All methods of the regular channel are also available.
type Reconnector struct {
	*amqp.Channel

	mux     sync.Mutex
	conn    Channeler
	backoff backoff.BackOff

	onCreate    []func(*amqp.Channel) error
	onReconnect []chan<- error
	onConsume   []chan<- error
}

// Channeler creates new channel. Implemented by amqp.Connection and connection.Redialer.
type Channeler interface {
	Channel() (*amqp.Channel, error)
}

// New creates new Reconnector with channel re-creation capabilities.
// Accepts additional options, like setting up QoS and registering notifications for events.
func New(conn Channeler, ops ...Option) (*Reconnector, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}

	r := Reconnector{
		Channel: ch,
		mux:     sync.Mutex{},
		conn:    conn,
		backoff: backoff.NewExponentialBackOff(),
	}

	for _, op := range ops {
		op(&r)
	}

	if err := r.createCallback(ch); err != nil {
		return nil, err
	}

	return &r, err
}

// Option to configure Reconnector.
type Option func(r *Reconnector)

// NotifyReconnect registers a listener for reconnection attempts.
// Will send reconnect result for each reconnection attempt.
// Channel will be closed on graceful shutdown.
func (r *Reconnector) NotifyReconnect(c chan error) <-chan error {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.onReconnect = append(r.onReconnect, c)

	return c
}

// NotifyConsume registers a listener for start of the consuming process.
// Will send result of each start of the consuming process.
// Channel will be closed on graceful shutdown.
func (r *Reconnector) NotifyConsume(c chan error) <-chan error {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.onConsume = append(r.onConsume, c)

	return c
}

// PublishWithContext checks if channel is closed and reconnects if needed.
// Useful to reliably publish events even after channel errors (which closes channel).
//
//nolint:gocritic // interface should be the same as Publish
func (r *Reconnector) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	if r.Channel.IsClosed() {
		if err := r.reconnect(); err != nil {
			return fmt.Errorf("reconnect: %w", err)
		}
	}

	return r.Channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
}

// Consume consumes with reconnection of the channel. Provides constant flow of the deliveries.
// On graceful closing of the channel, will deliver all remaining deliveries and exit.
//
//nolint:gocognit // complex function, will rewrite later
func (r *Reconnector) Consume(
	queue, consumer string,
	autoAck, exclusive, noLocal, noWait bool,
	args amqp.Table,
) <-chan amqp.Delivery {
	deliveries := make(chan amqp.Delivery)

	go func() {
		defer close(deliveries)

		for {
			if r.Channel.IsClosed() {
				err := r.reconnect()
				if err != nil {
					continue
				}
			}

			// The chan provided will be closed when the Channel is closed and on a
			// graceful close, no error will be sent.
			channelClosedCh := r.Channel.NotifyClose(make(chan *amqp.Error, 1))

			newDelsCh, err := r.Channel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
			r.notifyConsume(err)
			if err != nil {
				continue
			}
			// forward all deliveries until closed
			for d := range newDelsCh {
				deliveries <- d
			}

			select {
			case amqpErr := <-channelClosedCh:
				if amqpErr == nil { // on graceful close no error will be sent
					closeAll(append(r.onConsume, r.onReconnect...))
					return
				}
			default: // if no channelClosed notification received, that means delivering was closed by user
				closeAll(append(r.onConsume, r.onReconnect...))
				return
			}
		}
	}()

	return deliveries
}

func (r *Reconnector) reconnect() error {
	r.mux.Lock()
	defer r.mux.Unlock()

	operation := func() error {
		channel, err := r.conn.Channel()
		r.notifyReconnect(err)
		if err != nil {
			return fmt.Errorf("create channel: %w", err)
		}

		if err := r.createCallback(channel); err != nil {
			return fmt.Errorf("on reconnect callback: %w", err)
		}

		r.Channel = channel

		return nil
	}

	return backoff.Retry(operation, r.backoff)
}

func (r *Reconnector) createCallback(channel *amqp.Channel) error {
	for _, fn := range r.onCreate {
		if err := fn(channel); err != nil {
			return fmt.Errorf("on create callback: %w", err)
		}
	}

	return nil
}

func (r *Reconnector) notifyReconnect(err error) {
	for _, ch := range r.onReconnect {
		ch <- err
	}
}

func (r *Reconnector) notifyConsume(err error) {
	for _, ch := range r.onConsume {
		ch <- err
	}
}

func closeAll(chs []chan<- error) {
	for _, ch := range chs {
		close(ch)
	}
}
