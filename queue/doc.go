// Package queue определяет интерфейсы для работы с очередями сообщений.
//
// Пакет предоставляет базовые интерфейсы для Publisher/Subscriber паттерна.
// Реализации находятся в дочерних пакетах:
//   - [queue/rabbitmq] — RabbitMQ адаптер
//   - [queue/kafka] — Kafka адаптер
//
// Интерфейсы:
//   - [Publisher] — отправка сообщений в очередь
//   - [Subscriber] — получение сообщений из очереди
//   - [Encoder] — кодирование/декодирование сообщий
//   - [Handler] — обработчик входящих сообщений
//
// Типы:
//   - [Message] — структура для отправки сообщения
//   - [Delivery] — структура полученного сообщения
//
// Использование (Publisher):
//
//	var pub queue.Publisher = rabbitmq.NewPublisher(...)
//	err := pub.Publish(ctx, queue.Message{
//	    Topic: "orders",
//	    Body:  order,
//	})
//
// Использование (Subscriber):
//
//	var sub queue.Subscriber = rabbitmq.NewSubscriber(...)
//	sub.Listen(func(ctx context.Context, msg queue.Delivery) (bool, error) {
//	    // Обработка сообщения
//	    // return true для retry, false для подтверждения
//	    return false, nil
//	})
package queue
