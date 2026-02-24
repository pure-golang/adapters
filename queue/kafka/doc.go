// Package kafka реализует [queue.Publisher] и [queue.Subscriber] для Apache Kafka.
//
// Поддерживает OpenTelemetry tracing через [kafka.Dialer].
//
// Использование (Publisher):
//
//	dialer := kafka.NewDialer(tracer)
//	pub := kafka.NewPublisher(brokers, topic, kafka.WithDialer(dialer))
//	err := pub.Publish(ctx, message)
//
// Использование (Subscriber):
//
//	sub := kafka.NewSubscriber(brokers, topic, groupID, kafka.WithDialer(dialer))
//	err := sub.Subscribe(ctx, handler)
//
// Конфигурация через переменные окружения:
//
//	KAFKA_BROKERS — список брокеров через запятую (default: localhost:9092)
//	KAFKA_GROUP   — consumer group ID
package kafka
