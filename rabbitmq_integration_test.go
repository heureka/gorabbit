package rabbit_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	rabbit "github.com/heureka/gorabbit"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type RabbitTestSuite struct {
	suite.Suite
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	queue    string
	key      string
}

func (s *RabbitTestSuite) SetupSuite() {
	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		s.FailNow("please provide rabbitMQ URL via RABBITMQ_URL environment variable")
	}

	var err error
	var connection *amqp.Connection
	for i := 0; i < 5; i++ {
		connection, err = amqp.Dial(rmqURL)
		if err != nil {
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	if err != nil {
		s.FailNow("can't open connection", "%q: %s", rmqURL, err)
	}

	s.conn = connection
}

func (s *RabbitTestSuite) TearDownSuite() {
	if err := s.conn.Close(); err != nil {
		s.Fail("close connection", err)
	}
}

func (s *RabbitTestSuite) SetupTest() {
	ch, err := s.conn.Channel()
	if err != nil {
		s.FailNow("create channel", err)
	}

	s.channel = ch

	s.exchange = "my-exchange"

	if err := ch.ExchangeDeclare(
		s.exchange,
		"topic",
		false,
		false,
		false,
		false,
		nil,
	); err != nil {
		s.FailNow("declare exchange", err)
	}

	s.queue = "my-queue"

	if _, err := ch.QueueDeclare(
		s.queue,
		false,
		false,
		false,
		false,
		nil,
	); err != nil {
		s.FailNow("declare queue", err)
	}

	s.key = "my-key"
	if err := ch.QueueBind(
		s.queue,
		s.key,
		s.exchange,
		false,
		nil,
	); err != nil {
		s.FailNow("bind queue", err)
	}

}

func (s *RabbitTestSuite) TearDownTest() {
	if err := s.channel.ExchangeDelete(s.exchange, false, false); err != nil {
		s.Fail("delete exchange", s.exchange, err)
	}

	if _, err := s.channel.QueueDelete(s.queue, false, false, false); err != nil {
		s.Fail("delete queue", s.exchange, err)
	}

	if err := s.channel.Close(); err != nil {
		s.Fail("close channel", err)
	}
}

func (s *RabbitTestSuite) TestConsume() {
	tests := map[string]struct {
		options      []rabbit.ConsumerOption
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
		"consume err": {
			options:    nil,
			consumeErr: assert.AnError,
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
			wantErr: assert.AnError,
		},
		"with auto ACK": {
			options: []rabbit.ConsumerOption{
				rabbit.WithConsume(rabbit.WithConsumeAutoAck()),
			},
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
			txreader, err := rabbit.NewConsumer(
				s.conn,
				s.queue,
				tt.options...,
			)
			if err != nil {
				s.FailNow("create txreader", err)
			}

			defer func() {
				if err := txreader.Stop(); err != nil {
					s.Fail("stop txreader", err)
				}
			}()

			receiveCh := make(chan []byte)
			defer close(receiveCh)

			consumer := newMockEchoConsumer(receiveCh)
			consumer.On("Consume", mock.Anything, mock.Anything).Return(tt.consumeErr)

			errorsCh := txreader.Start(context.Background(), consumer)
			go func() {
				for e := range errorsCh {
					if tt.wantErr != nil {
						s.Assert().True(errors.Is(e, tt.wantErr), "should return correct error")
					} else {
						s.Fail("reading messages", e)
					}
				}
			}()

			for _, m := range tt.messages {
				if err := s.channel.Publish(
					s.exchange,
					s.key,
					false,
					false,
					amqp.Publishing{
						Body: m,
					}); err != nil {
					s.Fail("publish message", err)
				}
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

func (s *RabbitTestSuite) TestImmediatelyStop() {
	rmq, err := rabbit.NewConsumer(
		s.conn,
		s.queue,
	)
	if err != nil {
		s.FailNow("create rmq", err)
	}

	err = rmq.Stop()
	s.Assert().Nil(err, "should not return error")
}

func TestRabbitMQIntegration(t *testing.T) {
	suite.Run(t, new(RabbitTestSuite))
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
