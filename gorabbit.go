package gorabbit

import (
	"fmt"

	"github.com/heureka/gorabbit/channel"
	"github.com/heureka/gorabbit/connection"
	"github.com/heureka/gorabbit/consumer"
	"github.com/heureka/gorabbit/publisher"
)

// NewConsumer creates a new consumer from RabbitMQ, which will consume from a queue.
// Will automatically re-open channel on channel errors.
// Reconnection is done with exponential backoff.
func NewConsumer(url string, queue string, ops ...consumer.Option) (*consumer.Consumer, error) {
	ch, err := prepareChannel(url)
	if err != nil {
		return nil, err
	}

	return consumer.New(ch, queue, ops...), nil
}

// NewPublisher creates a new published to RabbitMQ, which will publish to exchange.
// Will automatically re-open channel on channel errors.
// Reconnection is done with exponential backoff.
func NewPublisher(url string, exchange string, mws ...publisher.Middleware) (publisher.Publisher, error) {
	ch, err := prepareChannel(url)
	if err != nil {
		return publisher.Publisher{}, err
	}

	return publisher.New(ch, exchange, mws...), nil
}

func prepareChannel(url string) (*channel.Reopener, error) {
	conn, err := connection.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	return ch, nil
}
