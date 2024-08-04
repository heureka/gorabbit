package rabbittest

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// BindQueue binds queue to exchange using routing key.
// Unbinds queue on test cleanup.
func BindQueue(t *testing.T, channel *amqp.Channel, name, key, exchange string) {
	t.Helper()

	if err := channel.QueueBind(name, key, exchange, false, nil); err != nil {
		t.Fatal("bind queue", err)
	}

	t.Cleanup(func() {
		if err := channel.QueueUnbind(name, key, exchange, nil); err != nil {
			t.Fatal("unbind queue", err)
		}
	})
}
