// Package rabbittest provides "stretchr/testify/suite" testing suite with prepared RabbitMQ connection, channel or topology.
//
// Usage:
//
//	// embedding ChannelSuite
//	type TestSuite struct {
//	  rabbittest.ChannelSuite
//	}
//
//	func (s *TestSuite) TestMyFunction() {
//	  s.Channel.Publish(...)
//	}
//
// Set up RABBITMQ_URL environment variable and tun tests:
//
//	RABBITMQ_URL=amqp://localhost:5672 go test -v  ./...
package rabbittest
