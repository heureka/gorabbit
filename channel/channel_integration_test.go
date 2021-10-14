package channel_test

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"

	"github.com/heureka/gorabbit/channel"
	"github.com/heureka/gorabbit/rabbittest"
)

type TestSuite struct {
	rabbittest.ConnectionSuite

	channel *channel.Reconnector
}

func (s *TestSuite) SetupTest() {
	ch, err := channel.New(s.Connection)
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

func (s *TestSuite) TestReconnectPublishing() {
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

	err = s.channel.PublishReconn(exchange, "", false, false, amqp.Publishing{})
	if err != nil {
		s.FailNow("publish", err)
	}

	s.causeException()

	// should be able to publish again
	err = s.channel.PublishReconn(exchange, "", true, false, amqp.Publishing{})
	if err != nil {
		s.FailNow("publish after channel close", err)
	}
}

// FIXME: can't cause exception, it is blocked on receiving closing notification
func (s *TestSuite) TestReconnectConsume() {
	consumerName := "test-consumer"

	queue, err := s.channel.QueueDeclare("test-queue", false, false, false, false, nil)
	if err != nil {
		s.FailNow("declare queue", err)
	}

	s.publish(queue.Name, "1")

	deliveries := s.channel.ConsumeReconn(queue.Name, consumerName, true, false, false, false, nil)

	// done := make(chan struct{})
	// var dels []amqp.Delivery
	// go func() {
	// 	defer close(done)
	// 	for d := range deliveries {
	// 		dels = append(dels, d)
	// 	}
	// }()

	d := <-deliveries
	s.Equal([]byte("1"), d.Body, "should receive first message")

	// s.causeException()

	s.publish(queue.Name, "2")

	d = <-deliveries
	s.Equal([]byte("2"), d.Body, "should receive second message")

	if err := s.channel.Cancel(consumerName, false); err != nil {
		s.FailNow("cancel delivering", err)
	}
	// go func() {
	// 	_ = s.channel.Close()
	//
	// }()
	// <-done

	// d = <-deliveries
	// s.Assert().Equal([]byte("2"), d.Body, "should receive second message")
}

func (s *TestSuite) TestGracefulShutdown() {
	consumerName := "test-consumer"

	queue, err := s.channel.QueueDeclare("test-queue", false, false, false, false, nil)
	if err != nil {
		s.FailNow("declare queue", err)
	}
	defer func() {
		ch, err := channel.New(s.Connection) // create new channel, as previous was closed
		if err != nil {
			s.FailNow("create channel", err)
		}

		s.channel = ch

		if _, err := s.channel.QueueDelete(queue.Name, false, false, false); err != nil {
			s.FailNow("delete queue", err)
		}
	}()

	deliveries := s.channel.ConsumeReconn(queue.Name, consumerName, true, false, false, false, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
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

func (s *TestSuite) publish(queue string, bodies ...string) {
	ch, err := s.Connection.Channel()
	if err != nil {
		s.FailNow("create channel")
	}
	defer func() {
		if err := ch.Close(); err != nil {
			s.FailNow("close channel", err)
		}
	}()

	for _, b := range bodies {
		err = ch.Publish("", queue, false, false, amqp.Publishing{Body: []byte(b)})
		if err != nil {
			s.FailNow("publish", err)
		}
	}
}

func (s *TestSuite) causeException() {
	closeCh := s.channel.NotifyClose(make(chan *amqp.Error))
	// cause an asynchronous channel exception causing the server to asynchronously close the channel
	err := s.channel.Publish("nonexisting", "", true, false, amqp.Publishing{})
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
