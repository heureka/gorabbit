package publisher_test

import (
	"context"
	"testing"
	"time"

	"github.com/heureka/gorabbit/publisher"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type callArgs struct {
	exchange  string
	key       string
	mandatory bool
	immediate bool
	msg       amqp.Publishing
}

func TestUnitPublish(t *testing.T) {
	exchange := "test-exchange"
	routingKey := "test-key"
	message := []byte(`test message`)

	tests := map[string]struct {
		middlewares       []publisher.Middleware
		publishMiddleware []publisher.Middleware
		publishError      error
		wantCallArgs      callArgs
		wantError         error
	}{
		"no middlewares": {
			wantCallArgs: callArgs{
				exchange:  exchange,
				key:       routingKey,
				mandatory: false,
				immediate: false,
				msg: amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					Body:         message,
				},
			},
		},
		"publish error": {
			wantCallArgs: callArgs{
				exchange:  exchange,
				key:       routingKey,
				mandatory: false,
				immediate: false,
				msg: amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					Body:         message,
				},
			},
			publishError: assert.AnError,
			wantError:    assert.AnError,
		},
		"all middlewares": {
			middlewares: []publisher.Middleware{
				publisher.WithHeaders(amqp.Table{"test": "header"}),
				publisher.WithTransientDeliveryMode(),
				publisher.WithMandatory(),
				publisher.WithImmediate(),
				publisher.WithExpiration(time.Second),
			},

			wantCallArgs: callArgs{
				exchange:  exchange,
				key:       routingKey,
				mandatory: true,
				immediate: true,
				msg: amqp.Publishing{
					Headers:      amqp.Table{"test": "header"},
					DeliveryMode: amqp.Transient,
					Body:         message,
					Expiration:   "1000",
				},
			},
		},
		"publish middlewares": {
			middlewares: []publisher.Middleware{
				publisher.WithHeaders(amqp.Table{"test": "header"}),
				publisher.WithTransientDeliveryMode(),
				publisher.WithMandatory(),
				publisher.WithImmediate(),
				publisher.WithExpiration(time.Second),
			},
			publishMiddleware: []publisher.Middleware{
				publisher.WithHeaders(amqp.Table{"test": "other-header", "test2": "header"}),
				publisher.WithExpiration(2 * time.Second),
			},
			wantCallArgs: callArgs{
				exchange:  exchange,
				key:       routingKey,
				mandatory: true,
				immediate: true,
				msg: amqp.Publishing{
					Headers:      amqp.Table{"test": "other-header", "test2": "header"},
					DeliveryMode: amqp.Transient,
					Body:         message,
					Expiration:   "2000",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			channel := new(mockChannel)
			channel.On(
				"PublishWithContext",
				mock.Anything,
				tt.wantCallArgs.exchange,
				tt.wantCallArgs.key,
				tt.wantCallArgs.mandatory,
				tt.wantCallArgs.immediate,
				tt.wantCallArgs.msg,
			).Return(tt.publishError)

			pub := publisher.New(channel, exchange, tt.middlewares...)
			err := pub.Publish(context.TODO(), routingKey, message, tt.publishMiddleware...)
			if tt.wantError != nil {
				assert.ErrorIs(t, err, tt.wantError)
				return
			}

			assert.NoError(t, err)
			channel.AssertExpectations(t)
		})
	}
}

type mockChannel struct {
	mock.Mock
}

func (m *mockChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return m.Called(ctx, exchange, key, mandatory, immediate, msg).Error(0)
}
