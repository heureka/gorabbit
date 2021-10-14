package rabbittest

import (
	"os"
	"time"

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

	s.Connection = connection
}

func (s *ConnectionSuite) TearDownSuite() {
	if err := s.Connection.Close(); err != nil {
		s.Fail("close connection", err)
	}
}
