package consume

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUnitTxWithMiddlewares(t *testing.T) {
	var orderOfCalls []string
	firstMW := func(h DeliveryHandler) DeliveryHandler {
		return func(ctx context.Context, d amqp.Delivery) error {
			orderOfCalls = append(orderOfCalls, "first")
			return h(ctx, d)
		}
	}
	secondMW := func(h DeliveryHandler) DeliveryHandler {
		return func(ctx context.Context, d amqp.Delivery) error {
			orderOfCalls = append(orderOfCalls, "second")
			return h(ctx, d)
		}
	}
	tx := func(ctx context.Context, bytes []byte) error {
		orderOfCalls = append(orderOfCalls, "tx")
		return nil
	}

	h := txWithMiddlewares(tx, firstMW, secondMW)

	err := h(context.TODO(), amqp.Delivery{})
	assert.Nil(t, err, "should not return error")
	assert.Equal(t, []string{"first", "second", "tx"}, orderOfCalls, "should call middlewares in correct order")
}

func TestUnitOneConsume(t *testing.T) {
	tests := map[string]struct {
		txErr     error
		ackErr    error
		rejectErr error
		wantErr   error
	}{
		"successful": {
			txErr:   nil,
			wantErr: nil,
		},
		"tx error": {
			txErr:   assert.AnError,
			wantErr: nil,
		},
		"reject error": {
			txErr:     errors.New("some error"),
			rejectErr: assert.AnError,
			wantErr:   assert.AnError,
		},
		"ack error": {
			txErr:   nil,
			ackErr:  assert.AnError,
			wantErr: assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			acker := new(mockAcknowledger)
			if tt.txErr != nil {
				acker.On("Reject", mock.AnythingOfType("uint64"), false).Return(tt.rejectErr)
			} else {
				acker.On("Ack", mock.AnythingOfType("uint64"), false).Return(tt.ackErr)
			}

			deliveries := []amqp.Delivery{
				{
					Acknowledger: acker,
					DeliveryTag:  1,
					Body:         []byte("1"),
				},
				{
					Acknowledger: acker,
					DeliveryTag:  2,
					Body:         []byte("2"),
				},
				{
					Acknowledger: acker,
					DeliveryTag:  3,
					Body:         []byte("3"),
				},
			}
			wantReceived := [][]byte{
				[]byte("1"),
				[]byte("2"),
				[]byte("3"),
			}

			deliveriesCh := make(chan amqp.Delivery)
			go func() {
				defer close(deliveriesCh)

				for _, d := range deliveries {
					deliveriesCh <- d
				}
			}()

			var received [][]byte
			tx := func(ctx context.Context, bytes []byte) error {
				received = append(received, bytes)
				return tt.txErr
			}

			one := ByOne(tx, false)
			err := one.Process(context.TODO(), deliveriesCh)
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "should return correct error")
			} else {
				assert.NoError(t, err, "should not return error")
				assert.Equal(t, wantReceived, received, "should receive expected messages")
			}

			acker.AssertExpectations(t)
		})
	}
}
