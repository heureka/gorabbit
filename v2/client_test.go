package gorabbit_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/heureka/gorabbit/v2"
	"github.com/heureka/gorabbit/v2/rabbittest"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/goleak"
	"golang.org/x/sync/errgroup"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestReconnectsOnClosedChannel(t *testing.T) {
	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		t.Fatal("please provide rabbitMQ URL via RABBITMQ_URL environment variable")
	}

	t.Run("publishing", func(t *testing.T) {
		exchange, queue, key := t.Name(), t.Name(), t.Name()
		setup(t, exchange, queue, key)

		client := gorabbit.New(
			rmqURL,
			gorabbit.WithOnChannelClosed(func(err error) {
				t.Logf("channel closed: %s", err)
			}),
			gorabbit.WithOnConnectionClosed(func(err error) {
				t.Logf("connection closed: %s", err)
			}),
		)
		if err := client.Publish(context.TODO(), exchange, key, amqp.Publishing{Body: []byte{'1'}}); err != nil {
			t.Fatalf("publish: %v", err)
		}

		// cause exception on the channel currently used
		causeException(t, client.Channel())

		if err := client.Publish(context.TODO(), exchange, key, amqp.Publishing{Body: []byte{'2'}}); err != nil {
			t.Fatalf("publish: %s", err)
		}

		if err := client.Close(); err != nil {
			t.Fatalf("close: %s", err)
		}
	})

	t.Run("consuming", func(t *testing.T) {
		exchange, queue, key := t.Name(), t.Name(), t.Name()
		channel := setup(t, exchange, queue, key)

		client := gorabbit.New(
			rmqURL,
			gorabbit.WithOnChannelClosed(func(err error) {
				t.Logf("channel closed: %s", err)
			}),
			gorabbit.WithOnConnectionClosed(func(err error) {
				t.Logf("connection closed: %s", err)
			}),
		)

		dels := make(chan amqp.Delivery)
		var eg errgroup.Group
		eg.Go(func() error {
			return client.Consume(context.Background(), queue, 2, dels)
		})

		if err := channel.Publish(exchange, key, false, false, amqp.Publishing{Body: []byte{'1'}}); err != nil {
			t.Fatalf("publish: %v", err)
		}
		// wait for delivery
		assertDelivery(t, dels, []byte{'1'})

		// cause exception on the channel currently used
		causeException(t, client.Channel())

		if err := channel.Publish(exchange, key, false, false, amqp.Publishing{Body: []byte{'2'}}); err != nil {
			t.Fatalf("publish: %s", err)
		}

		// wait for delivery, should be on new channel
		assertDelivery(t, dels, []byte{'2'})

		if err := client.Close(); err != nil {
			t.Fatalf("close: %s", err)
		}
		if err := eg.Wait(); err != nil {
			t.Fatalf("wait for consumign to stop: %s", err)
		}
	})

	t.Run("consuming and publishing", func(t *testing.T) {
		exchange, queue, key := t.Name(), t.Name(), t.Name()
		setup(t, exchange, queue, key)

		client := gorabbit.New(
			rmqURL,
			gorabbit.WithOnChannelClosed(func(err error) {
				t.Logf("channel closed: %s", err)
			}),
			gorabbit.WithOnConnectionClosed(func(err error) {
				t.Logf("connection closed: %s", err)
			}),
		)

		dels := make(chan amqp.Delivery)
		var eg errgroup.Group
		eg.Go(func() error {
			return client.Consume(context.Background(), queue, 2, dels)
		})

		if err := client.Publish(context.TODO(), exchange, key, amqp.Publishing{Body: []byte{'1'}}); err != nil {
			t.Fatalf("publish: %v", err)
		}
		// wait for delivery
		assertDelivery(t, dels, []byte{'1'})

		// cause exception on the channel currently used
		causeException(t, client.Channel())

		if err := client.Publish(context.TODO(), exchange, key, amqp.Publishing{Body: []byte{'2'}}); err != nil {
			t.Fatalf("publish: %s", err)
		}

		// wait for delivery, should be on new channel
		assertDelivery(t, dels, []byte{'2'})

		if err := client.Close(); err != nil {
			t.Fatalf("close: %s", err)
		}
		if err := eg.Wait(); err != nil {
			t.Fatalf("wait for consumign to stop: %s", err)
		}
	})
}

func TestReconnectsOnBrokenConnection(t *testing.T) {
	t.Run("publishing", func(t *testing.T) {
		exchange, queue, key := t.Name(), t.Name(), t.Name()
		setup(t, exchange, queue, key)

		breakSignal := make(chan struct{}, 1)
		client := breakableClient(t, breakSignal)

		if err := client.Publish(context.TODO(), exchange, key, amqp.Publishing{}); err != nil {
			t.Fatalf("publish: %s", err)
		}

		// break connection once
		breakSignal <- struct{}{}

		if err := client.Publish(context.TODO(), exchange, key, amqp.Publishing{}); err != nil {
			t.Fatalf("publish: %s", err)
		}

		if err := client.Close(); err != nil {
			t.Fatalf("close: %s", err)
		}
	})

	t.Run("consuming", func(t *testing.T) {
		exchange, queue, key := t.Name(), t.Name(), t.Name()
		channel := setup(t, exchange, queue, key)

		breakSignal := make(chan struct{}, 1)
		client := breakableClient(t, breakSignal)

		dels := make(chan amqp.Delivery, 1)
		var eg errgroup.Group
		eg.Go(func() error {
			return client.Consume(context.Background(), queue, 2, dels)
		})

		if err := channel.Publish(exchange, key, false, false, amqp.Publishing{Body: []byte{'1'}}); err != nil {
			t.Fatalf("publish: %v", err)
		}
		// wait for delivery
		assertDelivery(t, dels, []byte{'1'})

		// break connection once
		breakSignal <- struct{}{}

		if err := channel.Publish(exchange, key, false, false, amqp.Publishing{Body: []byte{'2'}}); err != nil {
			t.Fatalf("publish: %v", err)
		}

		// wait for delivery, should be on new channel
		assertDelivery(t, dels, []byte{'2'})

		if err := client.Close(); err != nil {
			t.Fatalf("close: %s", err)
		}
		if err := eg.Wait(); err != nil {
			t.Fatalf("wait for consumign to stop: %s", err)
		}
	})

	t.Run("publishing and consuming", func(t *testing.T) {
		exchange, queue, key := t.Name(), t.Name(), t.Name()
		setup(t, exchange, queue, key)

		breakSignal := make(chan struct{}, 1)
		client := breakableClient(t, breakSignal)

		dels := make(chan amqp.Delivery, 1)
		var eg errgroup.Group
		eg.Go(func() error {
			return client.Consume(context.Background(), queue, 2, dels)
		})

		if err := client.Publish(context.TODO(), exchange, key, amqp.Publishing{Body: []byte{'1'}}); err != nil {
			t.Fatalf("publish: %v", err)
		}
		// wait for delivery
		assertDelivery(t, dels, []byte{'1'})

		// break current connection once
		breakSignal <- struct{}{}

		if err := client.Publish(context.TODO(), exchange, key, amqp.Publishing{Body: []byte{'2'}}); err != nil {
			t.Fatalf("publish: %v", err)
		}

		// wait for delivery, should be on new channel
		assertDelivery(t, dels, []byte{'2'})

		if err := client.Close(); err != nil {
			t.Fatalf("close: %s", err)
		}
		if err := eg.Wait(); err != nil {
			t.Fatalf("wait for consumign to stop: %s", err)
		}
	})
}

func TestReconnectsUntilBackoff(t *testing.T) {
	t.Run("publishing", func(t *testing.T) {
		disconnect := make(chan struct{})
		client := breakableClient(t, disconnect)

		// break forever
		close(disconnect)

		err := client.Publish(context.Background(), "exchange", "key", amqp.Publishing{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if client.IsReady() {
			t.Fatal("expected to be not ready")
		}
		if client.IsLive() {
			t.Fatal("expected connection to be closed")
		}
	})

	t.Run("consuming", func(t *testing.T) {
		disconnect := make(chan struct{})
		client := breakableClient(t, disconnect)

		// break forever
		close(disconnect)

		err := client.Consume(context.Background(), "queue", 1, make(chan amqp.Delivery))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if client.IsReady() {
			t.Fatal("expected to be not ready")
		}
		if client.IsLive() {
			t.Fatal("expected to be not live")
		}
	})
}

func setup(t *testing.T, exchange, queue, key string) *amqp.Channel {
	channel := rabbittest.Channel(t, rabbittest.Connection(t))
	rabbittest.DeclareExchange(t, channel, exchange)
	rabbittest.DeclareQueue(t, channel, queue)
	rabbittest.BindQueue(t, channel, t.Name(), key, exchange)

	return channel
}

func assertDelivery(t *testing.T, dels <-chan amqp.Delivery, body []byte) {
	t.Helper()

	select {
	case del := <-dels:
		if err := del.Ack(false); err != nil {
			t.Fatalf("ack delivery %q: %s", del.Body, err)
		}

		for i := 0; i < len(body); i++ {
			if !bytes.Equal([]byte{body[i]}, del.Body) {
				t.Fatalf("unexpected delivery: %s", del.Body)
			}
		}

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for delivery")
	}
}

func causeException(t *testing.T, channel *amqp.Channel) {
	closeCh := channel.NotifyClose(make(chan *amqp.Error, 1))
	// cause an asynchronous channel exception causing the server to asynchronously close the channel
	err := channel.Publish("nonexisting", "", true, false, amqp.Publishing{})
	if err != nil {
		t.Fatalf("publish to nonexisting: %v", err)
	}
	// wait for channel to be closed
	amqpErr := <-closeCh
	if amqpErr.Code != amqp.NotFound {
		t.Fatalf("unexpected error code, want %d, got %d", amqp.NotFound, amqpErr.Code)
	}
}

func breakableClient(t *testing.T, signal chan struct{}) *gorabbit.Client {
	t.Helper()

	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		t.Fatal("please provide rabbitMQ URL via RABBITMQ_URL environment variable")
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 2 * time.Second
	bo.InitialInterval = 10 * time.Millisecond

	client := gorabbit.New(
		rmqURL,
		gorabbit.WithBackoff(bo),
		gorabbit.WithOnChannelClosed(func(err error) {
			t.Logf("channel closed: %s", err)
		}),
		gorabbit.WithOnConnectionClosed(func(err error) {
			t.Logf("connection closed: %s", err)
		}),
		gorabbit.WithDialConfig(func(c *amqp.Config) {
			c.Dial = func(network, addr string) (net.Conn, error) {
				c, err := amqp.DefaultDial(10*time.Second)(network, addr)
				if err != nil {
					return nil, err
				}

				return breakableConn{c, signal}, nil
			}
		}),
	)

	if !client.IsLive() {
		t.Fatal("expected to be live")
	}
	if !client.IsReady() {
		t.Fatal("expected to be ready")
	}

	return client
}

var errBroken = errors.New("broken")

type breakableConn struct {
	net.Conn
	done chan struct{}
}

func (c breakableConn) Read(b []byte) (int, error) {
	num, err := c.Conn.Read(b)
	// for Read it is important to check for done before returning, otherwise races is possible
	select {
	case <-c.done:
		return 0, errBroken
	default:

	}

	return num, err
}

func (c breakableConn) Write(b []byte) (int, error) {
	select {
	case <-c.done:
		return 0, errBroken
	default:
	}

	return c.Conn.Write(b)
}
