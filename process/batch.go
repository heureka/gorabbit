package process

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Batch consumes messages in batches.
// Implements rabbitmq.Consumer.
type Batch struct {
	amount        int
	timeout       time.Duration
	handler       BatchDeliveryHandler
	rejectRequeue bool
}

// InBatches creates new batch consumer. One batch is `amount` of messages or until timeout, whichever comes first.
func InBatches(amount int, timeout time.Duration, tx BatchTransaction, rejectRequeue bool, mws ...BatchMiddleware) *Batch {
	return &Batch{
		amount:        amount,
		timeout:       timeout,
		handler:       batchTxWithMiddlewares(tx, mws...),
		rejectRequeue: rejectRequeue,
	}
}

// BatchMiddleware specifies middlewares for processing batch of RabbitMQ deliveries.
// Could be used for logging, tracing, etc.
type BatchMiddleware func(BatchDeliveryHandler) BatchDeliveryHandler

// BatchDeliveryHandler specifies how to handle batch of RabbitMQ deliveries.
// Returns one-to-one errors of processed deliveries.
type BatchDeliveryHandler func(context.Context, []amqp.Delivery) (status []error)

// Process messages in batches.
func (c *Batch) Process(ctx context.Context, deliveries <-chan amqp.Delivery) error {
	for b := range c.inBatches(deliveries) {
		failed := c.handler(ctx, b)

		if err := c.ack(b, failed); err != nil {
			return err
		}
	}

	return nil
}

func (c *Batch) ack(batch []amqp.Delivery, status []error) error {
	for i := range batch {
		isFailed := status[i] != nil
		if err := ack(batch[i].Acknowledger, batch[i].DeliveryTag, isFailed, c.rejectRequeue); err != nil {
			return err
		}
	}

	return nil
}

// inBatches splits incoming deliveries into batches.
//
//nolint:gocognit //it is not too complex
func (c *Batch) inBatches(deliveries <-chan amqp.Delivery) <-chan []amqp.Delivery {
	batch := make([]amqp.Delivery, 0, c.amount)
	batches := make(chan []amqp.Delivery)
	ticker := time.NewTicker(c.timeout)

	go func() {
		defer close(batches)
		defer ticker.Stop()

		for {
			select {
			case d, ok := <-deliveries:
				if !ok {
					batches <- batch // send last collected batch
					return
				}

				batch = append(batch, d)
				if len(batch) >= c.amount {
					batches <- batch
					batch = make([]amqp.Delivery, 0, c.amount)
				}
			case <-ticker.C:
				if len(batch) == 0 {
					break
				}
				batches <- batch
				batch = make([]amqp.Delivery, 0, c.amount)
			}
		}
	}()

	return batches
}

// batchTxWithMiddlewares wraps transaction with middlewares.
func batchTxWithMiddlewares(tx BatchTransaction, mws ...BatchMiddleware) BatchDeliveryHandler {
	wrapped := func(ctx context.Context, ds []amqp.Delivery) []error {
		bodies := make([][]byte, 0, len(ds))
		for i := range ds {
			bodies = append(bodies, ds[i].Body)
		}

		return tx(ctx, bodies)
	}
	// apply in reverse order because we are wrapping
	for i := len(mws) - 1; i >= 0; i-- {
		wrapped = mws[i](wrapped)
	}

	return wrapped
}
