package rabbittest

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DeclareQueue is a test helper to declare a queue.
// Queue declaring parameters can be overwritten using ops.
// Deletes the queue on test cleanup.
func DeclareQueue(t *testing.T, channel *amqp.Channel, name string, ops ...func(p *QueueParams)) {
	t.Helper()

	params := QueueParams{
		Name:       name,
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
		Args:       nil,
	}
	for _, op := range ops {
		op(&params)
	}

	if _, err := channel.QueueDeclare(
		params.Name,
		params.Durable,
		params.AutoDelete,
		params.Exclusive,
		params.NoWait,
		params.Args,
	); err != nil {
		t.Fatal("declare queue", err)
	}

	t.Cleanup(func() {
		if _, err := channel.QueueDelete(name, false, false, false); err != nil {
			t.Fatal("delete queue", err)
		}
	})
}

// QueueParams for queue declaration.
type QueueParams struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}
