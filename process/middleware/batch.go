package middleware

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	"github.com/heureka/gorabbit/process"
)

// NewBatchDeliveryLogging creates batch middleware which logs all incoming deliveries.
func NewBatchDeliveryLogging(logger *zerolog.Logger) process.BatchMiddleware {
	return func(handler process.BatchDeliveryHandler) process.BatchDeliveryHandler {
		return func(ctx context.Context, deliveries []amqp.Delivery) []error {
			for i := range deliveries {
				d := &deliveries[i]

				logger.Info().
					Int("bytes", len(d.Body)).
					Str("routing_key", d.RoutingKey).
					Msg("got delivery")
			}

			return handler(ctx, deliveries)

		}
	}
}

// NewBatchErrorLogging creates batch middleware which logs all processing errors.
func NewBatchErrorLogging(logger *zerolog.Logger) process.BatchMiddleware {
	return func(handler process.BatchDeliveryHandler) process.BatchDeliveryHandler {
		return func(ctx context.Context, deliveries []amqp.Delivery) []error {
			status := handler(ctx, deliveries)
			for _, err := range status {
				if err != nil {
					logger.Err(err).
						Msg("can't handle delivery")
				}
			}

			return status
		}
	}
}
