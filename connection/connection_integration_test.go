package connection_test

import (
	"github.com/cenkalti/backoff/v4"
	"github.com/heureka/gorabbit/connection"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
	"time"
)

type TestSuite struct {
	suite.Suite

	rmqURL  string
	backoff backoff.BackOff
}

func (s *TestSuite) SetupSuite() {
	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		s.FailNow("please provide rabbitMQ URL via RABBITMQ_URL environment variable")
	}

	s.rmqURL = rmqURL
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 10 * time.Millisecond
	bo.MaxElapsedTime = 5 * time.Second
	s.backoff = backoff.WithMaxRetries(bo, 5)
}

func (s *TestSuite) TestDial() {
	tests := map[string]struct {
		rmqURL  string
		wantErr bool
	}{
		"correct url": {
			rmqURL:  s.rmqURL,
			wantErr: false,
		},
		"wrong rmq url": {
			rmqURL:  "amqp://guest:guest@abc:5672/",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			conn, err := connection.Dial(tt.rmqURL, connection.WithBackoff(s.backoff))
			if err == nil {
				defer conn.Close()
			}

			if !tt.wantErr {
				s.NoError(err)
			} else {
				s.Error(err, "should return error")
			}
		})
	}
}

func (s *TestSuite) TestDialledCallback() {
	tests := map[string]struct {
		rmqURL          string
		wantCalledTimes int
	}{
		"correct url": {
			rmqURL:          s.rmqURL,
			wantCalledTimes: 1,
		},
		"wrong rmq url": {
			rmqURL:          "amqp://guest:guest@abc:5672/",
			wantCalledTimes: 0,
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			count := 0
			conn, err := connection.Dial(
				tt.rmqURL,
				connection.WithBackoff(s.backoff),
				connection.WithDialledCallback(func(a *amqp.Connection) {
					count++
				}),
			)
			if err == nil {
				defer conn.Close()
			}

			s.Equal(tt.wantCalledTimes, count, "should call expected times")
		})
	}
}

func (s *TestSuite) TestDialAttemptCallback() {
	tests := map[string]struct {
		rmqURL          string
		wantCalledTimes int
	}{
		"correct url": {
			rmqURL:          s.rmqURL,
			wantCalledTimes: 1,
		},
		"wrong rmq url": {
			rmqURL:          "amqp://guest:guest@abc:5672/",
			wantCalledTimes: 6,
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			count := 0
			_, _ = connection.Dial(
				tt.rmqURL,
				connection.WithBackoff(s.backoff),
				connection.WithDialAttemptCallback(func(_ error) {
					count++
				}),
			)

			s.Equal(tt.wantCalledTimes, count, "should call expected times")
		})
	}
}

func (s *TestSuite) TestChannelCreation() {
	conn, closeFn := s.dial()
	defer closeFn()

	closedConn, closedCloseFn := s.dial()
	closedCloseFn()

	tests := map[string]struct {
		conn      *connection.Redialer
		wantError error
	}{
		"from opened connection": {
			conn: conn,
		},
		"from closed connection": {
			conn: closedConn,
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			ch, err := tt.conn.Channel()
			s.NoError(err)
			defer ch.Close()
		})
	}
}

func (s *TestSuite) dial() (*connection.Redialer, func()) {
	s.T().Helper()

	conn, err := connection.Dial(s.rmqURL, connection.WithBackoff(s.backoff))
	s.Require().NoError(err)

	return conn, func() {
		err := conn.Close()
		s.Require().NoError(err)
	}
}

func TestIntegrationConnection(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
