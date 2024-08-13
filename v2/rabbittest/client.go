package rabbittest

import (
	"os"
	"testing"

	"github.com/heureka/gorabbit/v2"
)

// NewClient creates a new RabbitMQ client.
// Closes the client on test cleanup.
func NewClient(t *testing.T, ops ...gorabbit.Option) *gorabbit.Client {
	t.Helper()

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Fatal("please provide rabbitMQ URL via RABBITMQ_URL environment variable")
	}

	client := gorabbit.New(url, ops...)

	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %s", err)
		}
	})

	return client
}
