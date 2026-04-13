// Package kafka реализует [queue.Publisher] и [queue.Subscriber] для Apache Kafka.
//
// Поддерживает OpenTelemetry tracing и автоматическое переподключение.
//
// Использование (Publisher):
//
//	dialer := kafka.NewDialer(kafka.Config{
//	    Brokers: []string{"localhost:9092"},
//	    GroupID: "my-group",
//	}, nil)
//	defer dialer.Close()
//
//	pub := kafka.NewPublisher(dialer, kafka.PublisherConfig{
//	    Encoder: encoders.JSON{},
//	})
//	err := pub.Publish(ctx, queue.Message{Topic: "my-topic", Body: payload})
//
// Использование (Subscriber):
//
//	sub := kafka.NewSubscriber(dialer, "my-topic", kafka.SubscriberConfig{
//	    MaxTryNum: 3,
//	    Backoff:   5 * time.Second,
//	})
//	defer sub.Close()
//	sub.Listen(ctx, handler) // блокируется до отмены ctx или вызова Close
package kafka
