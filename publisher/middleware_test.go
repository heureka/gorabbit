package publisher_test

import (
	"context"
	"testing"
	"time"

	"github.com/heureka/gorabbit/publisher"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishWithHeaders(t *testing.T) {
	headers := amqp.Table{"test": "header"}

	tests := map[string]struct {
		msg  amqp.Publishing
		want amqp.Table
	}{
		"no pre-existing headers": {
			msg:  amqp.Publishing{},
			want: headers,
		},
		"pre-existing headers": {
			msg: amqp.Publishing{
				Headers: amqp.Table{"other": "header"},
			},
			want: amqp.Table{
				"test":  "header",
				"other": "header",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			channel := &fakeChannel{}
			wrapped := publisher.Wrap(channel, publisher.WithHeaders(headers))

			err := wrapped.PublishWithContext(context.TODO(), "test", "test", false, false, tt.msg)
			require.NoError(t, err)

			require.Len(t, channel.published, 1, "should publish one message")

			assert.Equal(t, tt.want, channel.published[0].Headers, "should publish with expected headers")
		})
	}
}

func TestPublishWithExpiration(t *testing.T) {
	channel := &fakeChannel{}
	wrapped := publisher.Wrap(channel, publisher.WithExpiration(time.Second))

	err := wrapped.PublishWithContext(context.TODO(), "test", "test", false, false, amqp.Publishing{})
	require.NoError(t, err)

	require.Len(t, channel.published, 1, "should publish one message")

	assert.Equal(t, "1000", channel.published[0].Expiration, "should publish 1000ms expiration")
}
