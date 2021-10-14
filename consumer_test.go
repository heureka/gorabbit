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

	tests := map[string]struct {
		consumerTag     string
		wantConsumerTag string
	}{
		"preset": {
			consumerTag:     "some-consumer",
			wantConsumerTag: "some-consumer",
		},
		"not set": {
			consumerTag:     "",
			wantConsumerTag: "other-consumer",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			deliveries := make(chan amqp.Delivery)
			go func() {
				defer close(deliveries)

				for _, m := range messages {
					deliveries <- m
				}
			}()

			// read all values
			for range consumerTagSetter(&tt.consumerTag, deliveries) {
			}

			assert.Equal(t, tt.wantConsumerTag, tt.consumerTag, "should set correct consumer tag")
		})
	}
}
