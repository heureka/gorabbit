package publisher_test

import (
	"context"
	"testing"

	"github.com/heureka/gorabbit/publisher"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrap(t *testing.T) {
	mw1 := func(channel publisher.Channel) publisher.Channel {
		return publisher.ChannelFunc(func(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
			msg.Body = append(msg.Body, '1')

			return channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
		})
	}

	mw2 := func(channel publisher.Channel) publisher.Channel {
		return publisher.ChannelFunc(func(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
			msg.Body = append(msg.Body, '2')

			return channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
		})
	}
	ch := &fakeChannel{}
	wrapped := publisher.Wrap(ch, mw1, mw2)
	err := wrapped.PublishWithContext(context.TODO(), "test", "test", false, false, amqp.Publishing{})
	require.NoError(t, err)

	require.Len(t, ch.published, 1, "should publish one message")
	assert.Equal(t, []byte("12"), ch.published[0].Body, "middlewares should be applied in correct order")
}

type fakeChannel struct {
	published []amqp.Publishing
}

func (c *fakeChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	c.published = append(c.published, msg)
	return nil
}
