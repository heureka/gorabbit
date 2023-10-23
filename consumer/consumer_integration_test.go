package consumer_test

import (
	"context"
	"testing"
	"time"

	"github.com/heureka/gorabbit/channel"
	"github.com/heureka/gorabbit/consumer"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"

	"github.com/heureka/gorabbit/rabbittest"
)

type ConsumerTestSuite struct {
	rabbittest.TopologySuite
}

//nolint:gocognit // complexity is fine for a test
func (s *ConsumerTestSuite) TestConsume() {
	tests := map[string]struct {
		options      []consumer.Option
		messages     [][]byte
		consumeErr   error
		wantReceived [][]byte
		wantErr      error
	}{
		"no errors": {
			options: nil,
			messages: [][]byte{
				[]byte("1"),
				[]byte("2"),
				[]byte("3"),
			},
			wantReceived: [][]byte{
				[]byte("1"),
				[]byte("2"),
				[]byte("3"),
			},
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ch, err := channel.New(s.Connection.Connection)
			s.Require().NoError(err, "create channel")

			cons := consumer.New(
				ch,
				s.Queue,
				tt.options...,
			)

			receiveCh := make(chan []byte)
			defer close(receiveCh)

			processor := newMockEchoProcessor(receiveCh)
			processor.On("Process", mock.Anything, mock.Anything).Return(tt.consumeErr)

			var eg errgroup.Group
			eg.Go(func() error {
				return cons.Start(ctx, processor)
			})

			// publisher all messages
			for _, m := range tt.messages {
				s.publish(ctx, m)
			}

			// wait to consume all messages
			var received [][]byte
			for i := 0; i < len(tt.wantReceived); i++ {
				received = append(received, <-receiveCh)
			}

			s.Require().NoError(cons.Stop(), "must not return error on stop")
			s.Require().NoError(eg.Wait(), "must not return error on starting")

			s.Assert().ElementsMatch(tt.wantReceived, received, "should consume expected messages")
		})
	}
}

func (s *ConsumerTestSuite) TestImmediatelyStop() {
	ch, err := channel.New(s.Connection.Connection)
	s.Require().NoError(err)

	cons := consumer.New(ch, s.Queue)

	err = cons.Stop()
	s.Assert().NoError(err, "should not return error")
}

func (s *ConsumerTestSuite) publish(ctx context.Context, body []byte) {
	if err := s.Channel.PublishWithContext(
		ctx,
		s.Exchange,
		s.Key,
		false,
		false,
		amqp.Publishing{
			Body: body,
		}); err != nil {
		s.Fail("publish message", err)
	}
}

func TestRabbitMQIntegration(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}

type mockEchoConsumer struct {
	mock.Mock
	received chan<- []byte
}

func newMockEchoProcessor(received chan<- []byte) *mockEchoConsumer {
	return &mockEchoConsumer{
		received: received,
	}
}

func (m *mockEchoConsumer) Process(ctx context.Context, deliveries <-chan amqp.Delivery) error {
	args := m.Called(ctx, deliveries)

	for d := range deliveries {
		m.received <- d.Body
		if err := d.Ack(false); err != nil {
			return err
		}
	}

	return args.Error(0)
}
