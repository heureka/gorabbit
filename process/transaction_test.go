package process

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUnitAck(t *testing.T) {
	deliveryTag := uint64(1)

	tests := map[string]struct {
		ackErr        error
		rejectErr     error
		rejectRequeue bool
		failed        bool
		wantErr       error
	}{
		"successful": {
			rejectRequeue: true,
			failed:        false,
			wantErr:       nil,
		},
		"failed": {
			rejectRequeue: true,
			failed:        true,
			wantErr:       nil,
		},
		"failed no requeue": {
			rejectRequeue: false,
			failed:        true,
			wantErr:       nil,
		},
		"reject error": {
			failed:    true,
			rejectErr: assert.AnError,
			wantErr:   assert.AnError,
		},
		"ack error": {
			ackErr:  assert.AnError,
			wantErr: assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			acker := new(mockAcknowledger)

			if tt.failed {
				acker.On("Reject", deliveryTag, tt.rejectRequeue).Return(tt.rejectErr)
			} else {
				acker.On("Ack", deliveryTag, false).Return(tt.ackErr)
			}

			err := ack(acker, deliveryTag, tt.failed, tt.rejectRequeue)
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "should return correct error")
			} else {
				assert.NoError(t, err, "should not return error")
			}

			acker.AssertExpectations(t)
		})
	}
}

type mockAcknowledger struct {
	mock.Mock
}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	args := m.Called(tag, multiple)
	return args.Error(0)
}

func (m *mockAcknowledger) Nack(tag uint64, multiple, requeue bool) error {
	args := m.Called(tag, multiple, requeue)
	return args.Error(0)
}

func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	args := m.Called(tag, requeue)
	return args.Error(0)
}
