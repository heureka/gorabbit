package rabbittest

import (
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"
)

// ConnectionSuite creates RabbitMQ connection on test setup.
type ConnectionSuite struct {
	suite.Suite
	Connection *amqp.Connection
}

func (s *ConnectionSuite) SetupSuite() {
	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		s.FailNow("please provide rabbitMQ URL via RABBITMQ_URL environment variable")
	}

	var connection *amqp.Connection
	connectFn := func() error {
		var err error
		connection, err = amqp.Dial(rmqURL)
		return err
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 10 * time.Second
	if err := backoff.Retry(connectFn, bo); err != nil {
		s.FailNow("can't open connection", "%q: %s", rmqURL, err)
	}

	s.Connection = connection
}

func (s *ConnectionSuite) TearDownSuite() {
	if err := s.Connection.Close(); err != nil {
		s.Fail("close connection", err)
	}
}
