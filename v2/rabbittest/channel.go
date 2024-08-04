package rabbittest

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Channel opens a new channel on the connection.
// CLoses the channel on test cleanup.
func Channel(t *testing.T, conn *amqp.Connection) *amqp.Channel {
	t.Helper()

	channel, err := conn.Channel()
	if err != nil {
		t.Fatalf("open channel: %s", err)
	}

	t.Cleanup(func() {
		if err := channel.Close(); err != nil {
			t.Errorf("close RabbitMQ channel: %s", err)
		}
	})

	return channel
}
