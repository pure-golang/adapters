// Package rabbitmq реализует [queue.Publisher] и [queue.Subscriber] для RabbitMQ.
//
// Поддерживает:
//   - простую публикацию и подписку
//   - мультиподписку через [MultiQueueSubscriber]
//   - DLX-based retry с настраиваемыми политиками повторов
//   - broker-подтверждения (publisher confirms) с батчевой оптимизацией
//   - топологию через [Definitions] (декларация exchange, queue, bindings)
//   - OpenTelemetry tracing
//
// Использование (Publisher):
//
//	dialer := rabbitmq.NewDialer("amqp://guest:guest@localhost:5672/", nil)
//	dialer.Connect()
//	defer dialer.Close()
//
//	// Предпочтительный конструктор.
//	pub := rabbitmq.NewPublisher2(rabbitmq.PublisherConfig{
//	    RoutingKey: "my-queue",
//	    Encoder:    encoders.JSON{},
//	    // PublisherConfirms: 10, // батч по 10 сообщений; 0 — fire-and-forget (default)
//	}, dialer)
//	err = pub.Publish(ctx, queue.Message{Body: payload})
//
//	// Устаревший конструктор (оставлен для обратной совместимости).
//	pub := rabbitmq.NewPublisher(dialer, rabbitmq.PublisherConfig{RoutingKey: "my-queue"})
//
// Использование (Subscriber):
//
//	// Предпочтительный конструктор.
//	sub := rabbitmq.NewSubscriber2(rabbitmq.SubscriberConfig{
//	    QueueName:      "my-queue",
//	    MaxRetries:     3,
//	    RetryQueueName: "my-queue.retry", // default: QueueName+".retry"
//	}, dialer)
//	defer sub.Close()
//	sub.Listen(ctx, handler) // блокируется до отмены ctx или вызова Close
//
//	// Устаревший конструктор (оставлен для обратной совместимости).
//	sub := rabbitmq.NewSubscriber(dialer, "my-queue", rabbitmq.SubscriberOptions{MaxRetries: 3})
//
// Использование (MultiQueueSubscriber):
//
//	sub := rabbitmq.NewMultiQueueSubscriber(dialer, rabbitmq.MultiQueueOptions{
//	    PrefetchCount: 10,
//	    MaxRetries:    3,
//	})
//	defer sub.Close()
//	sub.Listen(ctx, // блокируется до отмены ctx или вызова Close
//	    rabbitmq.QueueHandler{QueueName: "queue-1", Handler: h1},
//	    rabbitmq.QueueHandler{QueueName: "queue-2", Handler: h2},
//	)
//
// Использование (Connector):
//
//	dialer := rabbitmq.NewDialer("amqp://guest:guest@localhost:5672/", nil)
//	connector := rabbitmq.NewConnector(dialer)
//	if err := connector.ConnectWithRetry(ctx); err != nil { ... }
//	// опционально — если exchange или очередь создаются внешним сервисом:
//	connector.WaitForExchange(ctx, "my-exchange")
//	connector.WaitForQueue(ctx, "my-queue")
//
// Конфигурация через переменные окружения:
//
//	RABBITMQ_URL — URL подключения (default: amqp://guest:guest@localhost:5672/)
//
// Ограничения:
//
//   - Publisher: thread-safe при PublisherConfirms == 0; при PublisherConfirms > 0
//     конкурентные вызовы Publish сериализуются внутри через mutex.
//   - Subscriber/MultiQueueSubscriber: не thread-safe, Listen вызывается однократно.
package rabbitmq
