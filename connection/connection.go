package connection

import (
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/heureka/gorabbit/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultHeartbeat = 10 * time.Second
	defaultLocale    = "en_US"
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
	redialer := Redialer{
		mux: sync.Mutex{},
		url: url,
		cfg: amqp.Config{
			// defaults are the same as in amqp091-go:
			// https://github.com/rabbitmq/amqp091-go/blob/a6fa7f7d76ecb00f1f46d42d02adfe0454c5514f/connection.go#L160
			Heartbeat: defaultHeartbeat,
			Locale:    defaultLocale,
		},
		backoff:   backoff.NewExponentialBackOff(),
		onDialled: nil,
	}

	for _, op := range ops {
		op(&redialer)
	}

	if err := redialer.dial(redialer.cfg); err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return &redialer, nil
}

// Option to configure Reconnector.
type Option func(r *Redialer)

// Channel create new channel.Reopener with channel re-open capabilities.
// Will re-dial if connection is closed.
// Access original channel with r.Connection.Channel().
func (r *Redialer) Channel(ops ...channel.Option) (*channel.Reopener, error) {
	if r.IsClosed() {
		if err := r.dial(r.cfg); err != nil {
			return nil, err
		}
	}

	return channel.New(r.Connection, ops...)
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
