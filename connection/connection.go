package connection

import (
	"fmt"
	"sync"

	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Redialer wraps connection to add re-dial capabilities.
// All methods of the regular channel are also available.
type Redialer struct {
	*amqp.Connection

	mux       sync.Mutex
	url       string
	cfg       amqp.Config
	backoff   backoff.BackOff
	onDialled []func(*amqp.Connection)
	onAttempt []func(error)
}

// Dial is a regular amqp.Dial with re-dialing and backoff.
func Dial(url string, ops ...Option) (*Redialer, error) {
	r := Redialer{
		mux:       sync.Mutex{},
		url:       url,
		cfg:       amqp.Config{},
		backoff:   backoff.NewExponentialBackOff(),
		onDialled: nil,
	}

	for _, op := range ops {
		op(&r)
	}

	if err := r.dial(r.cfg); err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return &r, nil
}

// Option to configure Reconnector.
type Option func(r *Redialer)

// Channel opens new amqp.Channel.
// Will create connection if channel is closed.
func (r *Redialer) Channel() (*amqp.Channel, error) {
	if r.IsClosed() {
		if err := r.dial(r.cfg); err != nil {
			return nil, err
		}
	}

	return r.Connection.Channel()
}

func (r *Redialer) dial(cfg amqp.Config) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	operation := func() error {
		conn, err := amqp.DialConfig(r.url, cfg)
		r.notifyAttempt(err)
		if err != nil {
			err := fmt.Errorf("connection dial: %w", err)
			return err
		}

		r.notifyDialled(conn)

		r.Connection = conn

		return nil
	}

	return backoff.Retry(operation, r.backoff)
}

func (r *Redialer) notifyDialled(conn *amqp.Connection) {
	for _, fn := range r.onDialled {
		fn(conn)
	}
}

func (r *Redialer) notifyAttempt(err error) {
	for _, fn := range r.onAttempt {
		fn(err)
	}
}
