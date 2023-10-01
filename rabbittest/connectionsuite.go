package rabbittest

import (
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/heureka/gorabbit/connection"
	"github.com/stretchr/testify/suite"
)

// ConnectionSuite creates RabbitMQ connection on test setup.
type ConnectionSuite struct {
	suite.Suite
	Connection *connection.Redialer
}

// SetupSuite creates new RabbitMQ connection.
// Implements suite.SetupAllSuite.
func (s *ConnectionSuite) SetupSuite() {
	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		s.FailNow("please provide rabbitMQ URL via RABBITMQ_URL environment variable")
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 10 * time.Second
	conn, err := connection.Dial(rmqURL, connection.WithBackoff(bo))
	if err != nil {
		s.FailNow("can't open connection", "%q: %s", rmqURL, err)
	}

	s.Connection = conn
}

// TearDownSuite closes RabbitMQ connection.
// Implements suite.TearDownAllSuite.
func (s *ConnectionSuite) TearDownSuite() {
	if err := s.Connection.Close(); err != nil {
		s.Fail("close connection", err)
	}
}
