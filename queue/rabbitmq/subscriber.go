package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pure-golang/adapters/queue"
)

const ConsumeRetryInterval = 5 * time.Second

// Subscriber читает сообщения из одной очереди RabbitMQ.
// Повторные попытки реализованы через DLX: при ошибке сообщение публикуется
// в retry-очередь (с x-message-ttl), откуда RabbitMQ возвращает его в основную
// очередь по истечении TTL. Счётчик попыток читается из стандартного заголовка
// x-death, который поддерживается RabbitMQ и сохраняется между перезапусками.
type Subscriber struct {
	name      string
	queueName string
	cfg       SubscriberOptions
	dialer    *Dialer
	logger    *slog.Logger
}

type SubscriberOptions struct {
	// Name — consumer tag для AMQP. Генерируется автоматически, если пустой.
	Name string
	// PrefetchCount — значение Qos канала. По умолчанию 1.
	PrefetchCount int
	// MaxRetries — максимальное число попыток доставки, после чего сообщение
	// отправляется в dead-letter queue (Nack requeue=false). Должно быть > 0.
	MaxRetries int
	// RetryQueueName — очередь, куда публикуются неуспешные сообщения для
	// отложенной повторной обработки. По умолчанию queueName + ".retry".
	RetryQueueName string
	// MessageTimeout ограничивает время выполнения обработчика для одного сообщения.
	// 0 означает отсутствие таймаута (осторожно: RabbitMQ consumer timeout по умолчанию 30 мин).
	MessageTimeout time.Duration
}

func NewSubscriber(dialer *Dialer, queueName string, opts SubscriberOptions) *Subscriber {
	if opts.Name == "" {
		opts.Name = uuid.NewString()
	}
	if opts.PrefetchCount <= 0 {
		opts.PrefetchCount = 1
	}
	if opts.RetryQueueName == "" {
		opts.RetryQueueName = queueName + ".retry"
	}
	return &Subscriber{
		name:      opts.Name,
		queueName: queueName,
		cfg:       opts,
		dialer:    dialer,
		logger:    dialer.options.Logger.With("subscriber", opts.Name).With("queue", queueName),
	}
}

// Listen запускает чтение сообщений. Блокируется до отмены ctx.
// При потере соединения автоматически переподключается через ConsumeRetryInterval.
func (s *Subscriber) Listen(ctx context.Context, handler queue.Handler) {
	s.logger.Info("listening...")
	for {
		if err := s.listen(ctx, handler); err != nil {
			s.logger.With("error", err.Error()).Error("connection lost")
		}
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(ConsumeRetryInterval):
		}
	}
}

func (s *Subscriber) listen(ctx context.Context, handler queue.Handler) error {
	ch, err := s.dialer.Channel()
	if err != nil {
		return errors.Wrap(err, "open channel")
	}
	defer ch.Close()

	if err := ch.Qos(s.cfg.PrefetchCount, 0, false); err != nil {
		return errors.Wrap(err, "set Qos")
	}

	deliveries, err := ch.Consume(s.queueName, s.name, false, false, false, false, nil)
	if err != nil {
		return errors.Wrapf(err, "consume %q", s.queueName)
	}

	notifyClose := ch.NotifyClose(make(chan *amqp.Error, 1))

	for {
		select {
		case <-ctx.Done():
			return nil
		case amqpErr := <-notifyClose:
			return errors.Wrap(amqpErr, "channel closed")
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			if err := s.handleDelivery(ctx, ch, &d, handler); err != nil {
				return err
			}
		}
	}
}

func (s *Subscriber) handleDelivery(ctx context.Context, ch *amqp.Channel, d *amqp.Delivery, handler queue.Handler) error {
	hCtx := otel.GetTextMapPropagator().Extract(ctx, tableCarrier(d.Headers))
	hCtx, span := tracer.Start(hCtx, s.queueName, trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	span.SetAttributes(
		attribute.String("id", d.MessageId),
		attribute.String("consumer_name", s.name),
	)

	handlerCtx := hCtx
	if s.cfg.MessageTimeout > 0 {
		var cancel context.CancelFunc
		handlerCtx, cancel = context.WithTimeout(hCtx, s.cfg.MessageTimeout)
		defer cancel()
	}

	_, err := handler(handlerCtx, newDelivery(d))
	if err == nil {
		if ackErr := ch.Ack(d.DeliveryTag, false); ackErr != nil {
			span.SetStatus(codes.Error, ackErr.Error())
			return errors.Wrap(ackErr, "ack")
		}
		return nil
	}

	s.logger.With("error", err.Error()).Error("handle message failed")
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	if deathCount(d) >= s.cfg.MaxRetries {
		// Попытки исчерпаны → dead-letter queue через x-dead-letter-* на основной очереди
		if nackErr := ch.Nack(d.DeliveryTag, false, false); nackErr != nil {
			span.SetStatus(codes.Error, nackErr.Error())
			return errors.Wrap(nackErr, "nack to DLQ")
		}
		return nil
	}

	// Публикуем в retry-очередь; RabbitMQ вернёт сообщение в основную очередь по истечении x-message-ttl
	msg := amqp.Publishing{
		MessageId:    d.MessageId,
		ContentType:  d.ContentType,
		DeliveryMode: d.DeliveryMode,
		Body:         d.Body,
		Headers:      d.Headers,
	}
	if pubErr := ch.Publish("", s.cfg.RetryQueueName, false, false, msg); pubErr != nil {
		span.SetStatus(codes.Error, pubErr.Error())
		// Запасной вариант: requeue, чтобы сообщение не потерялось
		if nackErr := ch.Nack(d.DeliveryTag, false, true); nackErr != nil {
			return errors.Wrap(nackErr, "nack requeue after retry-publish failure")
		}
		return nil
	}

	if ackErr := ch.Ack(d.DeliveryTag, false); ackErr != nil {
		span.SetStatus(codes.Error, ackErr.Error())
		return errors.Wrap(ackErr, "ack after retry publish")
	}
	return nil
}

// deathCount возвращает суммарное число доставок из заголовка x-death.
// RabbitMQ автоматически увеличивает этот счётчик при каждом прохождении
// сообщения через цикл dead-letter, поэтому значение сохраняется после перезапусков.
func deathCount(d *amqp.Delivery) int {
	deaths, ok := d.Headers["x-death"].([]any)
	if !ok {
		return 0
	}
	var total int
	for _, entry := range deaths {
		if table, ok := entry.(amqp.Table); ok {
			if count, ok := table["count"].(int64); ok {
				total += int(count)
			}
		}
	}
	return total
}

func newDelivery(msg *amqp.Delivery) queue.Delivery {
	headers := make(map[string]string, len(msg.Headers))
	for k, v := range msg.Headers {
		headers[k] = fmt.Sprintf("%v", v)
	}
	return queue.Delivery{
		Headers: headers,
		Body:    msg.Body,
	}
}
