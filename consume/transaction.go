package consume

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Transaction defines process function for handling single message.
type Transaction func(context.Context, []byte) error

// BatchTransaction defines process function for handling batch of messages.
// Batch is a slice of messages, should return slice of one-to-one errors for each message.
type BatchTransaction func(context.Context, [][]byte) (status []error)

// ack rejects or ack message based on if it's failed or not.
func ack(acker amqp.Acknowledger, tag uint64, failed, rejectRequeue bool) error {
	if failed {
		if err := acker.Reject(tag, rejectRequeue); err != nil {
			return fmt.Errorf("reject delivery: %w", err)
		}
	} else {
		if err := acker.Ack(tag, false); err != nil {
			return fmt.Errorf("ack delivery: %w", err)
		}
	}

	return nil
}
