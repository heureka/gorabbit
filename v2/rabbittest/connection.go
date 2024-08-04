package rabbittest

import (
	"os"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection opens a new connection to RabbitMQ.
// Closes the connection on test cleanup.
func Connection(t *testing.T) *amqp.Connection {
	t.Helper()

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Fatal("please provide rabbitMQ URL via RABBITMQ_URL environment variable")
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Fatalf("close RabbitMQ connection: %v", err)
		}
	})

	return conn
}
