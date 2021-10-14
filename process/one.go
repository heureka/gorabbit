package process

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// One consumes messages one-by-one.
type One struct {
	handler       DeliveryHandler
	rejectRequeue bool
}

// ByOne creates new One consumer.
// It passes message into transaction and automatically ack/reject based on transaction's returned error.
func ByOne(tx Transaction, rejectRequeue bool, mws ...Middleware) *One {
	return &One{
		handler:       txWithMiddlewares(tx, mws...),
		rejectRequeue: rejectRequeue,
	}
}

// DeliveryHandler specifies how to handle raw RabbitMQ delivery.
type DeliveryHandler func(context.Context, amqp.Delivery) error

// Middleware specifies middlewares for processing RabbitMQ delivery.
// Could be used for logging, tracing, etc.
type Middleware func(DeliveryHandler) DeliveryHandler

// Consume deliveries.
func (c *One) Consume(ctx context.Context, deliveries <-chan amqp.Delivery) error {
	for d := range deliveries {
		err := c.handler(ctx, d)

		if ackErr := ack(d.Acknowledger, d.DeliveryTag, err != nil, c.rejectRequeue); ackErr != nil {
			return ackErr
		}
	}

	return nil
}

// txWithMiddlewares wraps transaction with middlewares.
func txWithMiddlewares(tx Transaction, mws ...Middleware) DeliveryHandler {
	wrapped := func(ctx context.Context, d amqp.Delivery) error {
		if err := tx(ctx, d.Body); err != nil {
			return fmt.Errorf("delivery transaction: %w", err)
		}

		return nil
	}
	// apply in reverse order because we are wrapping
	for i := len(mws) - 1; i >= 0; i-- {
		wrapped = mws[i](wrapped)
	}

	return wrapped
}
