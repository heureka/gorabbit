package rabbit

import (
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
	conn    *amqp.Connection
	backoff backoff.BackOff

	onReconnect     func(*amqp.Channel) error
	reconnectErrors chan<- error
}

func New(conn *amqp.Connection, ops ...Option) (*Reconnector, error) {
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

	return &r, err
}

// Option to configure Reconnector.
type Option func(r *Reconnector)

// PublishReconn checks if channel is closed and reconnects if needed.
// Useful to reliably publish events even after channel errors.
func (r *Reconnector) PublishReconn(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	if r.Channel.IsClosed() {
		if err := r.reconnect(); err != nil {
			return fmt.Errorf("reconnect: %w", err)
		}
	}

	return r.Channel.Publish(exchange, key, mandatory, immediate, msg)

}

// TODO: test by sending to not existing exchange - should generate error and close channel

// ConsumeReconn consumes with reconnection of the channel. Provides constant flow of the deliveries.
// On graceful closing of the channel, will deliver all remaining deliveries and exit.
func (r *Reconnector) ConsumeReconn(
	queue, consumer string,
	autoAck, exclusive, noLocal, noWait bool,
	args amqp.Table,
) <-chan amqp.Delivery {
	deliveries := make(chan amqp.Delivery)

	go func() {
		defer close(deliveries)

		for {
			if !r.Channel.IsClosed() {
				err := r.reconnect()
				if err != nil { // can't reconnect event after retries
					r.notifyError(fmt.Errorf("reconnect: %w", err))
					return
				}

			}

			// The chan provided will be closed when the Channel is closed and on a
			// graceful close, no error will be sent.
			channelClosedCh := r.Channel.NotifyClose(make(chan *amqp.Error))

			newDels, err := r.Channel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
			if err != nil {
				r.notifyError(fmt.Errorf("consume from channel: %w", err))
				continue
			}
			// forward all deliveries until closed
			for d := range newDels {
				deliveries <- d
			}

			// on graceful close no error will be sent
			amqpErr := <-channelClosedCh
			if amqpErr == nil {
				return
			}

			r.notifyError(fmt.Errorf("channel closed: %w", amqpErr))
		}
	}()

	return deliveries
}

func (r *Reconnector) reconnect() error {
	r.mux.Lock()
	defer r.mux.Unlock()

	operation := func() error {
		channel, err := r.conn.Channel()
		if err != nil {
			return fmt.Errorf("create channel: %w", err)
		}

		if r.onReconnect != nil {
			if err := r.onReconnect(r.Channel); err != nil {
				return fmt.Errorf("on reconnet callback: %w", err)
			}
		}

		r.Channel = channel

		return nil
	}

	if err := backoff.Retry(operation, r.backoff); err != nil {
		return err
	}

	return nil
}

func (r *Reconnector) notifyError(err error) {
	if r.reconnectErrors != nil {
		r.reconnectErrors <- err
	}
}
