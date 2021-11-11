# GoRabbit

TODO: add labels

GoRabbit provides additional capabilities for official RabbitMQ Go client [amqp091-go](https://github.com/rabbitmq/amqp091-go).

## Reconnection

This package adds channel reconnection capabilities on the top of a plain channel.

```go
ch := channel.New(conn)
// will reconnect on channel errors
deliveries := ch.ConsumeReconn("my-queue", "", false, false, false, false, nil)
```

You are welcomed to use it separately or together with [consumer](#consumer).


## Consumer

Consumers are hard to write correctly. That's why we are providing [process](./process) building blocks:

- One and Batch consumers, where all you need is to write `Transaction` and pass it to a consumer.
    `Transaction` is a function for handling received message:
    if it returns error, consumer will automatically NACK message, ACK on success.

- you can write your own consumer, implementing simple interface:
    ```go
    type Consumer interface {
        Consume(ctx context.Context, deliveries <-chan amqp.Delivery) error
    }
    ```
    Set-ups and reconnections will be handled for you.


All consumers have sane defaults, but at the same time fully configurable.

### One

[One consumer](process/one.go) reads messages one-by-one and passes them to `Transaction`. 
Will ACK them on success, NACK on error.

### One Middlewares

Easily plug in any middlewares and pass them to `Consumer`, implementing simple API:

```go
type Middleware func(DeliveryHandler) DeliveryHandler
```

Check out more examples at [process/middleware/one.go](./process/middleware/one.go).

### Batch

If you want to process messages in batches, [Batch consumer](process/batch.go) is here for you.

Batch consumer reads batch of messages from a broker and passes them to `BatchTransaction`. 
Your code expected to return one-to-one errors for each passed message (`len(messages) == len(errors)`).

Consumer will ACK each of messages on success, NACK on error.

### Batch Middlewares

Plug in any middlewares and pass them to the Batch consumer, implementing simple API:

```go
type BatchMiddleware func(BatchDeliveryHandler) BatchDeliveryHandler
```

Check out more examples at [process/middleware/batch.go](./process/middleware/batch.go).
