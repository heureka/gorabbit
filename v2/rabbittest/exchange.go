package rabbittest

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DeclareExchange is a test helper to declare an exchange.
// By default, creates durable topic exchange.
// Exchange declaring parameters can be overwritten using ops.
// Deletes the exchange on test cleanup.
func DeclareExchange(t *testing.T, channel *amqp.Channel, name string, ops ...func(p *ExchangeParams)) {
	t.Helper()

	params := ExchangeParams{
		Name:       name,
		Kind:       "topic",
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
		Args:       nil,
	}
	for _, op := range ops {
		op(&params)
	}

	if err := channel.ExchangeDeclare(
		params.Name,
		params.Kind,
		params.Durable,
		params.AutoDelete,
		params.Internal,
		params.NoWait,
		params.Args,
	); err != nil {
		t.Fatal("declare exchange", err)
	}

	t.Cleanup(func() {
		if err := channel.ExchangeDelete(name, false, false); err != nil {
			t.Fatal("delete exchange", err)
		}
	})
}

// ExchangeParams for exchange declaration.
type ExchangeParams struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
}
