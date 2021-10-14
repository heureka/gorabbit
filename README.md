# GoRabbit

GoRabbit provides additional capabilities for official RabbitMQ Go client [amqp091-go](https://github.com/rabbitmq/amqp091-go).

## Reconnection

This package adds channel reconnection capabilities on the top of plain channel.

```go
TODO
```

You are welcomed to use it separately or together with [consumer](#consumer) and [producer](#producer).


## Consumer

Consumers are hard to write correctly. That's why we are providing consumers building blocks:

- One and Batch consumers, where all you need is to write `Transaction` and pass it to a consumer.
    `Transcation` is a function for handling received message.
    If it returns error, consumer will automatically NACK message, ACK on success.

- you can write your own consumer, implementing simple interface:
    ```go
    type Consumer interface {
        Consume(ctx context.Context, deliveries <-chan amqp.Delivery) error
    }
    ```
    All set-ups and reconnections will be handled for you.


All consumers have sane defaults, but at the same time fully configurable.

### One

One consumer reads messages one-by-one and passes them to `Transaction`. 
Will ACK them on success, NACK on error.

### Batch

If you want to process messages in batches, Batch Consumer is here for you.

Batch consumer reads batch of messages from a broker and passes them to `BatchTransaction`. 
Your code expected to return one-to-one errors for each passed message (`len(messages) == len(errors)`).

Consumer will ACK each of messages on success, NACK on error.

## Producer

Producer simply sends messages to a broker. As it uses [reconnect](#reconnection) feature,
it will reliably send messages, even if channel was closed due to previous error.

### Middleware

Additionally, plenty of [prepared middlewares](TODO: add link) are available to plug-in.
Use them or write your own by following simple signature.

Please check [prepared middlewares] if you'd like to write your own.
