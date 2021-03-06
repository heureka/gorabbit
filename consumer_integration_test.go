package gorabbit_test

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/heureka/gorabbit"
	"github.com/heureka/gorabbit/rabbittest"
)

type ConsumerTestSuite struct {
	rabbittest.TopologySuite
}

func (s *ConsumerTestSuite) TestConsume() {
	tests := map[string]struct {
		options      []gorabbit.ConsumerOption
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
			txreader, err := gorabbit.NewConsumer(
				s.Connection,
				s.Queue,
				tt.options...,
			)
			if err != nil {
				s.FailNow("create txreader", err)
			}

			receiveCh := make(chan []byte)
			defer close(receiveCh)

			consumer := newMockEchoConsumer(receiveCh)
			consumer.On("Consume", mock.Anything, mock.Anything).Return(tt.consumeErr)

			// done := make(chan struct{})
			go func() {
				defer func() {
					if err := txreader.Stop(); err != nil {
						s.Fail("stop txreader", err)
					}
				}()

				if err = txreader.Start(context.Background(), consumer); err != nil {
					s.Fail("start consuming messages", err)
				}
			}()

			// publish all messages
			for _, m := range tt.messages {
				s.publish(m)
			}

			// wait to consume all messages
			var received [][]byte
			for i := 0; i < len(tt.wantReceived); i++ {
				received = append(received, <-receiveCh)
			}

			s.Assert().ElementsMatch(tt.wantReceived, received, "should consume expected messages")
		})
	}
}

func (s *ConsumerTestSuite) TestImmediatelyStop() {
	rmq, err := gorabbit.NewConsumer(
		s.Connection,
		s.Queue,
	)
	if err != nil {
		s.FailNow("create rmq", err)
	}

	err = rmq.Stop()
	s.Assert().NoError(err, "should not return error")
}

func (s *ConsumerTestSuite) publish(body []byte) {
	if err := s.Channel.Publish(
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

func newMockEchoConsumer(received chan<- []byte) *mockEchoConsumer {
	return &mockEchoConsumer{
		received: received,
	}
}

func (m *mockEchoConsumer) Consume(ctx context.Context, deliveries <-chan amqp.Delivery) error {
	args := m.Called(ctx, deliveries)

	for d := range deliveries {
		m.received <- d.Body
		if err := d.Ack(false); err != nil {
			return err
		}
	}

	return args.Error(0)
}
