package channel_test

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"

	"github.com/heureka/gorabbit/channel"
	"github.com/heureka/gorabbit/rabbittest"
)

type TestSuite struct {
	rabbittest.ConnectionSuite

	channel *channel.Reopener
}

func (s *TestSuite) SetupTest() {
	ch, err := channel.New(s.Connection.Connection)
	if err != nil {
		s.FailNow("should not return error", err)
	}

	s.channel = ch
}

func (s *TestSuite) TearDownTest() {
	if err := s.channel.Close(); err != nil {
		s.FailNow("close channel", err)
	}
}

// TestReopenPublishing tests channel can reopen when publishing and channel got closed.
func (s *TestSuite) TestReopenPublishing() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exchange := "test-exchange"

	err := s.channel.ExchangeDeclare(exchange, "direct", false, false, false, false, nil)
	if err != nil {
		s.FailNow("should not return error", err)
	}
	defer func() {
		if err := s.channel.ExchangeDelete(exchange, false, false); err != nil {
			s.FailNow("delete exchange", err)
		}
	}()

	err = s.channel.PublishWithContext(ctx, exchange, "", false, false, amqp.Publishing{})
	if err != nil {
		s.FailNow("publish", err)
	}

	s.causeException(ctx)

	// should be able to publish again
	err = s.channel.PublishWithContext(ctx, exchange, "", true, false, amqp.Publishing{})
	if err != nil {
		s.FailNow("publish after channel close", err)
	}
}

// TestReopenConsume tests channel can reopen when consuming and channel got closed.
func (s *TestSuite) TestReopenConsume() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	consumerName := "test-consumer"

	queue, err := s.channel.QueueDeclare("test-queue", false, false, false, false, nil)
	if err != nil {
		s.FailNow("declare queue", err)
	}

	// Purge the queue from the publisher side to establish initial state
	if _, err := s.channel.QueuePurge(queue.Name, false); err != nil {
		s.FailNow("purge queue", err)
	}

	s.publish(ctx, queue.Name, "1")

	deliveries := s.channel.Consume(queue.Name, consumerName, false, false, false, false, nil)

	delivery := <-deliveries
	s.Equal([]byte("1"), delivery.Body, "should receive first message")

	if err := delivery.Ack(false); err != nil {
		s.FailNow("ack delivery", err)
	}

	//  Simulate an error like a server restart
	s.causeException(ctx)

	s.publish(ctx, queue.Name, "2")

	delivery = <-deliveries
	s.Equal([]byte("2"), delivery.Body, "should receive second message")

	if err := s.channel.Cancel(consumerName, false); err != nil {
		s.FailNow("cancel delivering", err)
	}
}

func (s *TestSuite) TestGracefulShutdown() {
	consumerName := "test-consumer"

	queue, err := s.channel.QueueDeclare("test-queue", false, false, false, false, nil)
	if err != nil {
		s.FailNow("declare queue", err)
	}
	defer func() {
		ch, err := channel.New(s.Connection.Connection) // create new channel, as previous was closed
		if err != nil {
			s.FailNow("create channel", err)
		}

		s.channel = ch

		if _, err := s.channel.QueueDelete(queue.Name, false, false, false); err != nil {
			s.FailNow("delete queue", err)
		}
	}()

	deliveries := s.channel.Consume(queue.Name, consumerName, true, false, false, false, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		//nolint:revive // need to read all values
		for range deliveries {
		}
	}()

	if err := s.channel.Cancel(consumerName, false); err != nil {
		s.FailNow("cancel consuming", err)
	}

	<-done // waiting for all deliveries

	if err := s.channel.Close(); err != nil {
		s.FailNow("close channel", err)
	}

	// on graceful close channel should be closed, not recreated, so we shouldn't be able to declare exchange
	err = s.channel.ExchangeDeclare("test-exchange", "direct", false, false, false, false, nil)
	s.Equal(amqp.ErrClosed, err, "should return error that channel is closed")
}

// publish in separate channel.
func (s *TestSuite) publish(ctx context.Context, queue string, bodies ...string) {
	channel, err := s.Connection.Channel()
	if err != nil {
		s.FailNow("create channel")
	}
	defer func() {
		if err := channel.Close(); err != nil {
			s.FailNow("close channel", err)
		}
	}()

	for _, b := range bodies {
		err = channel.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{Body: []byte(b)})
		if err != nil {
			s.FailNow("publish", err)
		}
	}
}

func (s *TestSuite) causeException(ctx context.Context) {
	closeCh := s.channel.NotifyClose(make(chan *amqp.Error))
	// cause an asynchronous channel exception causing the server to asynchronously close the channel
	err := s.channel.PublishWithContext(ctx, "nonexisting", "", true, false, amqp.Publishing{})
	if err != nil {
		s.FailNow("publish to nonexisting", err)
	}
	// wait for channel to be closed
	amqpErr := <-closeCh
	s.Assert().Equal(amqpErr.Code, 404, "should return correct error code", amqpErr)
}

func TestIntegrationChannel(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
