package consume

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestUnitBatchTxWithMiddlewares(t *testing.T) {
	var orderOfCalls []string
	firstMW := func(h BatchDeliveryHandler) BatchDeliveryHandler {
		return func(ctx context.Context, ds []amqp.Delivery) []error {
			orderOfCalls = append(orderOfCalls, "first")
			return h(ctx, ds)
		}
	}
	secondMW := func(h BatchDeliveryHandler) BatchDeliveryHandler {
		return func(ctx context.Context, ds []amqp.Delivery) []error {
			orderOfCalls = append(orderOfCalls, "second")
			return h(ctx, ds)
		}
	}
	tx := func(ctx context.Context, messages [][]byte) []error {
		orderOfCalls = append(orderOfCalls, "tx")
		return make([]error, len(messages))
	}

	h := batchTxWithMiddlewares(tx, firstMW, secondMW)

	deliveries := []amqp.Delivery{{}}
	status := h(context.TODO(), deliveries)
	assert.Len(t, status, len(deliveries), "len(status) should be the same as number of deliveries")
	for _, err := range status {
		assert.NoError(t, err, "should not return error")
	}
	assert.Equal(t, []string{"first", "second", "tx"}, orderOfCalls, "should call middlewares in correct order")
}

func TestUnitBatchInBatches(t *testing.T) {
	dels := []amqp.Delivery{
		{
			Body: []byte("1"),
		},
		{
			Body: []byte("2"),
		},
		{
			Body: []byte("3"),
		},
	}

	wantBatches := [][]amqp.Delivery{
		{
			{
				Body: []byte("1"),
			},
			{
				Body: []byte("2"),
			},
		},
		{
			{
				Body: []byte("3"),
			},
		},
	}

	tx := func(ctx context.Context, msgs [][]byte) []error {
		return make([]error, len(msgs))
	}
	batchProcessing := InBatches(2, time.Second, tx, false)

	deliveriesCh := make(chan amqp.Delivery)
	go func() {
		defer close(deliveriesCh)

		for _, d := range dels {
			deliveriesCh <- d
		}
	}()

	batches := batchProcessing.inBatches(deliveriesCh)

	got := make([][]amqp.Delivery, 0, 2)
	for b := range batches {
		got = append(got, b)
	}

	assert.Equal(t, wantBatches, got, "should split in correct batches")
}

func TestUnitBatchInBatches_Timeout(t *testing.T) {
	del := amqp.Delivery{
		Body: []byte("1"),
	}

	wantBatch := []amqp.Delivery{
		{
			Body: []byte("1"),
		},
	}

	tx := func(ctx context.Context, msgs [][]byte) []error {
		return make([]error, len(msgs))
	}
	process := InBatches(2, time.Millisecond, tx, false)

	deliveriesCh := make(chan amqp.Delivery)
	defer close(deliveriesCh)
	// only send 1 delivery, should timeout
	go func() {
		deliveriesCh <- del
	}()

	batches := process.inBatches(deliveriesCh)
	got := <-batches

	assert.Equal(t, wantBatch, got, "should receive correct batch")
}

func TestUnitBatchAck(t *testing.T) {
	tx := func(ctx context.Context, msgs [][]byte) []error {
		return make([]error, len(msgs))
	}

	processor := InBatches(2, time.Millisecond, tx, false)

	tests := map[string]struct {
		status       []error
		ackErr       error
		wantAcked    []uint64
		wantNacked   []uint64
		wantRejected []uint64
		wantErr      error
	}{
		"successful": {
			status: []error{
				nil,
				nil,
				nil,
			},
			wantAcked: []uint64{
				1, 2, 3,
			},
		},
		"all failed": {
			status: []error{
				assert.AnError,
				assert.AnError,
				assert.AnError,
			},
			wantRejected: []uint64{
				1, 2, 3,
			},
		},
		"partial fail": {
			status: []error{
				assert.AnError,
				nil,
				assert.AnError,
			},
			wantRejected: []uint64{
				1, 3,
			},
			wantAcked: []uint64{2},
		},
		"ack error": {
			status: []error{
				nil,
				nil,
				nil,
			},
			ackErr:  assert.AnError,
			wantErr: assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			acker := newAckRecorder(tt.ackErr)

			batch := []amqp.Delivery{
				{
					Acknowledger: acker,
					DeliveryTag:  1,
				},
				{
					Acknowledger: acker,
					DeliveryTag:  2,
				},
				{
					Acknowledger: acker,
					DeliveryTag:  3,
				},
			}

			err := processor.ack(batch, tt.status)
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "should return correct error")
			} else {
				assert.NoError(t, err, "should not return error")
				assert.Equal(t, tt.wantAcked, acker.acked, "should ack expected messages")
				assert.Equal(t, tt.wantNacked, acker.nacked, "should nack expected messages")
				assert.Equal(t, tt.wantRejected, acker.rejected, "should reject expected messages")
			}
		})
	}
}

func TestUnitBatchConsume(t *testing.T) {
	tests := map[string]struct {
		ackErr  error
		wantErr error
	}{
		"successful": {
			ackErr:  nil,
			wantErr: nil,
		},
		"ack error": {
			ackErr:  assert.AnError,
			wantErr: assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			acker := newAckRecorder(tt.ackErr)

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

			wantReceived := [][][]byte{
				{
					[]byte("1"),
					[]byte("2"),
				},
				{
					[]byte("3"),
				},
			}

			var received [][][]byte
			tx := func(ctx context.Context, msgs [][]byte) []error {
				received = append(received, msgs)
				return make([]error, len(msgs))
			}

			deliveriesCh := make(chan amqp.Delivery)
			go func() {
				defer close(deliveriesCh)

				for _, d := range deliveries {
					deliveriesCh <- d
				}
			}()

			b := InBatches(2, time.Second, tx, false)
			err := b.Process(context.TODO(), deliveriesCh)
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "should return correct error")
			} else {
				assert.NoError(t, err, "should not return error")
				assert.Equal(t, wantReceived, received)
			}
		})
	}
}

type ackRecorder struct {
	acked    []uint64
	nacked   []uint64
	rejected []uint64

	retError error
}

func newAckRecorder(retError error) *ackRecorder {
	return &ackRecorder{
		retError: retError,
	}
}

func (a *ackRecorder) Ack(tag uint64, _ bool) error {
	a.acked = append(a.acked, tag)
	return a.retError
}

func (a *ackRecorder) Nack(tag uint64, _, _ bool) error {
	a.nacked = append(a.nacked, tag)
	return a.retError
}

func (a *ackRecorder) Reject(tag uint64, _ bool) error {
	a.rejected = append(a.rejected, tag)
	return a.retError
}
