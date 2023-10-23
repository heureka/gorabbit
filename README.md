[![Go Reference](https://pkg.go.dev/badge/github.com/heureka/gorabbit.svg)](https://pkg.go.dev/github.com/heureka/gorabbit)
![GitHub tag (with filter)](https://img.shields.io/github/v/tag/heureka/gorabbit)
![GitHub Workflow Status (with event)](https://img.shields.io/github/actions/workflow/status/heureka/gorabbit/go.yml)


# GoRabbit

GoRabbit provides additional capabilities for official RabbitMQ Go client [amqp091-go](https://github.com/rabbitmq/amqp091-go).

Quick start example:

```go
consumer, _ := gorabbit.NewConsumer("amqp://localhost:5672", "my-queue")
handleMessage := func(ctx context.Context, message []byte) error {
    log.Println(message)
    return nil
}
consumer.Start(context.Background(), consume.ByOne(handleMessage, false))
```

This will create consumer which will connect to RabbitMQ on localhost,
start consuming messages one-by-one from "my-queue" and pass them to `handleMessage` function.

It will re-dial if connection got broken and re-open channel if it will be closed.

You can use prepared consumers and publishers, or just re-dialing of connection and re-opening of a channel.


## Consumer

GoRabbit provides [consumer](./consumer/consumer.go), 
which simplifies creation of RabbitMQ consumer. 
It aims to provide sane defaults, but at the same time is fully configurable:

```go
cons, _ := gorabbit.NewConsumer("amqp://localhost:5672", "example-queue")
cons.Start(context.Background(), consumer.ProcessorFunc(func(ctx context.Context, deliveries <-chan amqp.Delivery) error {
    for d := range deliveries {
        // process deliveries
    }
    return nil
}))
```
See full example at [examples/consumer](./examples/consumer/main.go).

To simplify it even more, GoRabbit is providing [consume](./consume) building blocks:
[ByOne](#one) and [InBatches](#batch). All you need is to write a function 
for handling received messages:
if function returns error, it will automatically NACK message, or ACK on success.

### One

[One consume](process/one.go) reads messages one-by-one and passes them to `Transaction`. 
Will ACK them on success, NACK on error.

Example of usage is in [examples/one](examples/one/main.go).

### One Middlewares

Easily plug-in any middlewares and pass them to `consume.ByOne`:

```go
consume.ByOne(tx, false, one.NewDeliveryLogging(zerolog.New(os.Stdout)))
```

Or implement your own with simple API:

```go
type Middleware func(DeliveryHandler) DeliveryHandler
```

Check out more examples at [consume/middleware/one.go](./consume/middleware/one.go).

### Batch

If you want to process messages in batches, [Batch consume](consume/batch.go) is here for you.

Batch processor reads a batch of messages from a broker and passes them to `BatchTransaction`. 
Your code expected to return one-to-one errors for each passed message (`len(messages) == len(errors)`).

Batch processor will ACK each of messages on success, NACK on error.

Example of usage is in [examples/batch](examples/batch/main.go).

### Batch Middlewares

Plug in any middlewares and pass them to the Batch consumer, implementing simple API:

```go
consume.InBatches(100, time.Second, tx, false, batch.NewDeliveryLogging(zerolog.New(os.Stdout)))
```

Or implement your own with simple API:

```go
type BatchMiddleware func(BatchDeliveryHandler) BatchDeliveryHandler
```

Check out more examples at [consume/middleware/batch.go](consume/middleware/batch.go).

# Publisher

GoRabbit provides [publisher](./publisher/publisher.go),
which simplifies creation of RabbitMQ publisher.
It aims to provide sane defaults, but at the same time is fully configurable:

```go
pub, _ := gorabbit.NewPublisher("amqp://localhost:5672", "example-exchange")
pub.Publish(context.Background(), "example-key", []byte("hello world!"))
```

See full example at [examples/publisher](examples/publisher/main.go).

## Connection re-dialing

Add re-dialing capabilities when connection got closed:

```go
conn, _ := connection.Dial("amqp://localhost:5672")
ch, _ := conn.Channel() // use as regular connection, but it will re-dial if connection is closed 
ch.Publish(...)
```

See full example at [examples/connection](./examples/connection/main.go).

## Channel re-opening

Add channel re-opening capabilities on the top of a plain channel,
so you can reliably consume or publish, even is something bad happened over the network:

```go
conn, _ := connection.Dial("amqp://localhost:5672")
ch, _ := conn.Channel(conn) // open channel with re-opening capabilities
// get constant flow of deliveries, channel will be re-opened if something goes south
for delivery := range ch.Consume("example-queue", "", false, false, false, false, nil) {
    // process deliveries
}
// or publisher reliably
ch.PublishWithContext(ctx, "example-exchange", "", false, false, amqp.Publishing{})
```
See full example at [examples/channel](./examples/channel/main.go).


# Testing

GoRabbit comes with testing helpers in a form of [stretchr/testify/suite](https://pkg.go.dev/github.com/stretchr/testify/suite)
testing suite,  with prepared RabbitMQ connection, channel or topology.

```go
type TestSuite struct {
  rabbittest.ChannelSuite
}

func (s *TestSuite) TestMyFunction() {
  s.Channel.Publish(...)
}
```

Set up RABBITMQ_URL environment variable and tun tests:
```shell
RABBITMQ_URL=amqp://localhost:5672 go test -v  ./...
```
