package publisher_test

import (
	"context"
	"testing"

	"github.com/heureka/gorabbit/publisher"
	"github.com/heureka/gorabbit/rabbittest"
	"github.com/stretchr/testify/suite"
)

type PublisherTestSuite struct {
	rabbittest.TopologySuite
}

func (s *PublisherTestSuite) SetupTest() {
	s.TopologySuite.Queue = s.T().Name()
	s.TopologySuite.Key = s.T().Name()
	s.TopologySuite.SetupTest()
}

func (s *PublisherTestSuite) TestPublish() {
	message := []byte("test")
	pub := publisher.New(s.Channel, s.Exchange)
	err := pub.Publish(context.TODO(), s.Key, message)
	s.Require().NoError(err, "must be able to publish")

	channel, err := s.Connection.Channel()
	s.Require().NoError(err)
	deliveries := channel.Consume(s.Queue, "", false, false, false, false, nil)

	d := <-deliveries
	gotMessage := d.Body

	s.Equal(message, gotMessage, "should receive published message")
}

func TestRabbitMQIntegration(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}
