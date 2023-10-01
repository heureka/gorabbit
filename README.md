# GoRabbit

GoRabbit provides additional capabilities for official RabbitMQ Go client [amqp091-go](https://github.com/rabbitmq/amqp091-go).

## Connection re-dialing

Add re-dialing capabilities when connection got closed,
you can reliably open new channel:

```go
conn, _ := connection.Dial("amqp://localhost:5672")
ch, _ := conn.Channel() // use as regular connection, but it will re-dial if connection is closed 
ch.Publish(...)
```

See full example at [examples/connection](./examples/connection/main.go).

## Channel re-opening

Add channel re-opening capabilities on the top of a plain channel, 
so you can get reliably consume or publish, even is something bad happened over the network.

```go
conn, _ := connection.Dial("amqp://localhost:5672")
ch, _ := channel.New(conn) // add re-opening capabilities
// get constant flow of deliveries, channel will be re-opened if something goes south
deliveries := ch.Consume("example-queue", "", false, false, false, false, nil)
for d := range deliveries {
    // process deliveries
}
// or publish reliably
ch.PublishWithContext(ctx, "example-exchange", "", false, false, amqp.Publishing{})
```
See full example at [examples/channel](./examples/channel/main.go).

## Consumers

GoRabbit provides [consumer](./consumer.go), 
which simplifies creation of RabbitMQ consumer. 
It aims to provide sane defaults, but at the same time is fully configurable:

```go
conn, _ := connection.Dial("amqp://localhost:5672")
ch, _ := channel.New(conn) 
consumer := gorabbit.NewConsumer(channel, "example-queue")
consumer.Start(context.Background(), gorabbit.ProcessorFunc(func(ctx context.Context, deliveries <-chan amqp.Delivery) error {
    for d := range deliveries {
        // process deliveries
    }
    return nil
}))

```
See full example at [examples/consumer](./examples/consumer/main.go).

To simplify it even more, GoRabbit is providing [process](./process) building blocks:
[ByOne](#one) and [InBatches](#batch). All you need is to write a function 
for handling received messages:
if it returns error, processor will automatically NACK message, ACK on success.

### One

[One consumer](process/one.go) reads messages one-by-one and passes them to `Transaction`. 
Will ACK them on success, NACK on error.

Example of usage is in [examples/one](examples/one/main.go).

### One Middlewares

Easily plug-in any middlewares and pass them to `process.ByOne`:

```go
process.ByOne(myHandler, true, middleware.NewErrorLogging(zerolog.New(os.Stderr))) 
```

Or implement your own with simple API:

```go
type Middleware func(DeliveryHandler) DeliveryHandler
```

Check out more examples at [process/middleware/one.go](./process/middleware/one.go).

### Batch

If you want to process messages in batches, [Batch processor](process/batch.go) is here for you.

Batch processor reads a batch of messages from a broker and passes them to `BatchTransaction`. 
Your code expected to return one-to-one errors for each passed message (`len(messages) == len(errors)`).

Consumer will ACK each of messages on success, NACK on error.

Example of usage is in [examples/batch](examples/batch/main.go).

### Batch Middlewares

Plug in any middlewares and pass them to the Batch consumer, implementing simple API:

```go
process.InBatches(100, time.Second, tx, false, middleware.NewBatchErrorLogging(zerolog.New(os.Stderr))) 
```

Or implement your own with simple API:

```go
type BatchMiddleware func(BatchDeliveryHandler) BatchDeliveryHandler
```

Check out more examples at [process/middleware/batch.go](./process/middleware/batch.go).

# Publishing

Publishing is done using channel [`channel.New`](#channel-re-opening) the same way as in regular RabbitMQ client.
But GoRabbit also has helpers to simplify adding middlewares to your publisher.
Use `publish.Wrap` do add as much middlewares as you want:

```go
p := publish.Wrap(ch, publish.WithHeaders(amqp.Table{"x-example-header": "example-value"}))
p.PublishWithContext(
    context.Background(),
    "example-exchange",
    "example-key",
    false,
    false,
    amqp.Publishing{Body: []byte("hello world!")},
)

```

See full example at [examples/publish](./examples/publish/main.go).

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
