// Package rabbitmq реализует [queue.Publisher] и [queue.Subscriber] для RabbitMQ.
//
// Поддерживает:
//   - простую публикацию и подписку
//   - мультиподписку через [MultiQueueSubscriber]
//   - DLX-based retry с настраиваемыми политиками повторов
//   - топологию через [Definitions] (декларация exchange, queue, bindings)
//   - OpenTelemetry tracing
//
// Использование (Publisher):
//
//	dialer, err := rabbitmq.Dial(ctx, rabbitmq.Config{URL: "amqp://guest:guest@localhost:5672/"})
//	pub, err := rabbitmq.NewPublisher(ctx, dialer, "exchange", rabbitmq.WithPublisherLogger(l))
//	err = pub.Publish(ctx, "routing.key", payload)
//
// Использование (Subscriber):
//
//	sub, err := rabbitmq.NewSubscriber(ctx, dialer, "queue", rabbitmq.WithSubscriberLogger(l))
//	err = sub.Subscribe(ctx, handler)
//
// Конфигурация через переменные окружения:
//
//	RABBITMQ_URL — URL подключения (default: amqp://guest:guest@localhost:5672/)
package rabbitmq
