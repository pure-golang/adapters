---
name: "queue_rabbitmq"
description: "Паттерны RabbitMQ: Topology/Definitions, Publisher, Subscriber (retry via x-death), MultiQueueSubscriber"
---
# Queue Patterns (RabbitMQ)

## Topology (Definitions)

`Definitions` mirrors the RabbitMQ management API JSON format:
- `.JSON()` → `rabbitmq_definitions.json` for docker-compose `load_definitions` / `rabbitmqctl import_definitions`
- `.applyDefinitions(ch *amqp.Channel)` → declare directly via AMQP (test-only helper, defined in `topology_helpers_test.go`)

See `queue/rabbitmq/README.md` for full DLX topology example.

## Publisher

Constructor signature:
```go
NewPublisher(dialer, PublisherConfig{
    Exchange:     "my-exchange",
    RoutingKey:   "my.routing.key",
    Encoder:      encoders.JSON{},
    DeliveryMode: amqp.Persistent,
    MessageTTL:   30 * time.Second,
})
```

## Subscriber (single queue)

Retry logic via `x-death` header (survives process restarts):

| Condition | Action |
|-----------|--------|
| error + `x-death < MaxRetries` | publish to `RetryQueueName` + Ack |
| error + `x-death >= MaxRetries` | `Nack(requeue=false)` → DLQ via `x-dead-letter-*` binding |
| publish-to-retry failure | fallback `Nack(requeue=true)` |

**Note:** `bool` return from handler is **ignored** — retry is always DLX-based via x-death header.

Constructor signature:
```go
NewSubscriber(dialer, SubscriberConfig{
    QueueName:     "my-queue",
    RetryQueueName: "my-queue.retry",
    MaxRetries:    3,
    PrefetchCount: 1,
    Encoder:       encoders.JSON{},
})
```

## MultiQueueSubscriber (multiple queues, single channel)

- Single channel with `Qos(N, global=true)` — prefetch limit across all consumers
- Single handler goroutine (fan-in pattern)
- Suitable for CPU/IO-heavy workloads where parallelism is undesirable

Constructor signature:
```go
NewMultiQueueSubscriber(dialer, MultiQueueSubscriberConfig{
    Queues:        []string{"queue-1", "queue-2"},
    PrefetchCount: 10,
    Encoder:       encoders.JSON{},
})
```

See `queue/rabbitmq/README.md` for full usage examples.
