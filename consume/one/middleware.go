package one

import (
	"context"

	"github.com/heureka/gorabbit/consume"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

// NewDeliveryLogging creates middleware which logs all incoming deliveries.
func NewDeliveryLogging(logger zerolog.Logger) consume.Middleware {
	return func(h consume.DeliveryHandler) consume.DeliveryHandler {
		return func(ctx context.Context, d amqp.Delivery) error {
			logger.Info().
				Int("bytes", len(d.Body)).
				Str("routing_key", d.RoutingKey).
				Msg("got delivery")

			return h(ctx, d)
		}
	}
}

// NewErrorLogging creates middleware which logs all processing errors.
func NewErrorLogging(logger zerolog.Logger) consume.Middleware {
	return func(h consume.DeliveryHandler) consume.DeliveryHandler {
		return func(ctx context.Context, d amqp.Delivery) error {
			err := h(ctx, d)
			if err != nil {
				logger.Err(err).
					Msg("can't handle delivery")
			}

			return nil
		}
	}
}
