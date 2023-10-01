package rabbittest

import amqp "github.com/rabbitmq/amqp091-go"

// ChannelSuite uses ConnectionSuite to create connection and then creates new channel for each test.
type ChannelSuite struct {
	ConnectionSuite

	Channel *amqp.Channel
}

// SetupTest creates new channel.
// Implements suite.SetupTestSuite.
func (s *ChannelSuite) SetupTest() {
	ch, err := s.Connection.Channel()
	if err != nil {
		s.FailNow("create channel", err)
	}

	s.Channel = ch
}

// TearDownTest closes channel.
// Implements suite.TearDownTestSuite.
func (s *ChannelSuite) TearDownTest() {
	if err := s.Channel.Close(); err != nil {
		s.Fail("close channel", err)
	}
}
