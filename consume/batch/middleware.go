package batch

import (
	"context"

	"github.com/heureka/gorabbit/consume"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

// NewDeliveryLogging creates batch middleware which logs all incoming deliveries.
func NewDeliveryLogging(logger zerolog.Logger) consume.BatchMiddleware {
	return func(handler consume.BatchDeliveryHandler) consume.BatchDeliveryHandler {
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

// NewErrorLogging creates batch middleware which logs all processing errors.
func NewErrorLogging(logger zerolog.Logger) consume.BatchMiddleware {
	return func(handler consume.BatchDeliveryHandler) consume.BatchDeliveryHandler {
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
