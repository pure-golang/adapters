---
name: "queue_kafka"
description: "Паттерны Kafka: dialer, publisher, subscriber с retry и backoff"
---
# Queue Patterns (Kafka)

```go
// Create Kafka dialer
cfg := kafka.Config{
    Brokers: []string{"localhost:9092"},
    GroupID: "my-consumer-group",
}
dialer := kafka.NewDialer(cfg, nil)

// Create publisher
pub := kafka.NewPublisher(dialer, kafka.PublisherConfig{
    Encoder:  encoders.JSON{},
    Balancer: &kafka.LeastBytes{},
})

// Publish message
msg := queue.Message{
    Topic: "my-topic",
    Body:  map[string]string{"key": "value"},
}
err := pub.Publish(ctx, msg)

// Create subscriber
sub := kafka.NewSubscriber(dialer, "my-topic", kafka.SubscriberConfig{
    Name:          "my-subscriber",
    PrefetchCount: 1,
    MaxTryNum:     3,
    Backoff:       5 * time.Second,
})

// Consume messages
go sub.Listen(func(ctx context.Context, msg queue.Delivery) (bool, error) {
    fmt.Println("Received:", string(msg.Body))
    return false, nil  // false = success, true = retry
})
```

## Handler Return Values
- `false, nil` — success, commit offset
- `true, err` — retry (up to `MaxTryNum` times with `Backoff` delay)
- `false, err` — skip message (no retry), log error
