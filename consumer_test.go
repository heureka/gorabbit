package rabbit

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestUnitConsumerTagSetter(t *testing.T) {
	messages := []amqp.Delivery{
		{
			ConsumerTag: "other-consumer",
		},
		{
			ConsumerTag: "other-consumer",
		},
	}

	wantConsumerTag := "other-consumer"
	consumerTag := ""

	deliveries := make(chan amqp.Delivery)
	go func() {
		defer close(deliveries)

		for _, m := range messages {
			deliveries <- m
		}
	}()

	// read all values
	//nolint:revive // need to read all values
	for range consumerTagProxy(&consumerTag, deliveries) {
	}

	assert.Equal(t, wantConsumerTag, consumerTag, "should set correct consumer tag")
}
