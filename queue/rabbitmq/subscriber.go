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

var _ queue.Subscriber = (*Subscriber)(nil)

// Subscriber читает сообщения из одной очереди RabbitMQ.
// Повторные попытки реализованы через DLX: при ошибке сообщение публикуется
// в retry-очередь (с x-message-ttl), откуда RabbitMQ возвращает его в основную
// очередь по истечении TTL. Счётчик попыток читается из стандартного заголовка
// x-death, который поддерживается RabbitMQ и сохраняется между перезапусками.
type Subscriber struct {
	logger    *slog.Logger
	name      string
	queueName string
	cfg       SubscriberConfig
	dialer    channelProvider
	close     chan struct{}
}

// SubscriberConfig — конфигурация Subscriber для [NewSubscriber2].
type SubscriberConfig struct {
	// Name — consumer tag для AMQP. Генерируется автоматически, если пустой.
	Name string
	// QueueName — имя очереди, из которой читаются сообщения.
	QueueName string
	// PrefetchCount — значение Qos канала. По умолчанию 1.
	PrefetchCount int
	// MaxRetries — максимальное число попыток доставки, после чего сообщение
	// отправляется в dead-letter queue (Nack requeue=false). Должно быть > 0.
	MaxRetries int
	// RetryQueueName — очередь, куда публикуются неуспешные сообщения для
	// отложенной повторной обработки. По умолчанию QueueName + ".retry".
	RetryQueueName string
	// MessageTimeout ограничивает время выполнения обработчика для одного сообщения.
	// 0 означает отсутствие таймаута (осторожно: RabbitMQ consumer timeout по умолчанию 30 мин).
	MessageTimeout time.Duration
}

// SubscriberOptions — устаревший тип конфигурации. Используйте [SubscriberConfig] и [NewSubscriber2].
//
// Deprecated: используйте [SubscriberConfig].
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

// NewSubscriber2 создаёт Subscriber, принимая зависимость через интерфейс channelProvider.
// Предпочтительный конструктор: позволяет подменять dialer в тестах без конкретного *Dialer.
func NewSubscriber2(cfg SubscriberConfig, dialer channelProvider) *Subscriber {
	if cfg.Name == "" {
		cfg.Name = uuid.NewString()
	}
	if cfg.PrefetchCount <= 0 {
		cfg.PrefetchCount = 1
	}
	if cfg.RetryQueueName == "" {
		cfg.RetryQueueName = cfg.QueueName + ".retry"
	}
	return &Subscriber{
		logger: dialer.Logger().WithGroup("subscriber").
			With("name", cfg.Name).With("queue", cfg.QueueName),
		name:      cfg.Name,
		queueName: cfg.QueueName,
		cfg:       cfg,
		dialer:    dialer,
		close:     make(chan struct{}),
	}
}

// NewSubscriber создаёт Subscriber с конкретным *Dialer.
//
// Deprecated: используйте [NewSubscriber2] с [SubscriberConfig].
func NewSubscriber(dialer *Dialer, queueName string, opts SubscriberOptions) *Subscriber {
	return NewSubscriber2(SubscriberConfig{
		Name:           opts.Name,
		QueueName:      queueName,
		PrefetchCount:  opts.PrefetchCount,
		MaxRetries:     opts.MaxRetries,
		RetryQueueName: opts.RetryQueueName,
		MessageTimeout: opts.MessageTimeout,
	}, dialer)
}

// Close останавливает subscriber. Можно вызывать вместо (или вместе с) отменой ctx в Listen.
func (s *Subscriber) Close() error {
	select {
	case <-s.close:
	default:
		close(s.close)
	}
	return nil
}

// Listen запускает чтение сообщений. Блокируется до отмены ctx или вызова Close.
// При потере соединения автоматически переподключается через ConsumeRetryInterval.
func (s *Subscriber) Listen(ctx context.Context, handler queue.Handler) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-s.close:
			cancel()
		case <-ctx.Done():
		}
	}()

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

func (s *Subscriber) listen(ctx context.Context, handler queue.Handler) (err error) {
	ch, err := s.dialer.Channel()
	if err != nil {
		return errors.Wrap(err, "open channel")
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, "close channel")
		}
	}()
	defer func() { // LIFO: выполняется перед ch.Close(), брокер получает basic.cancel
		if err := ch.Cancel(s.name, false); err != nil {
			s.logger.With("error", err).Error("cancel consumer")
		}
	}()

	if err := ch.Qos(s.cfg.PrefetchCount, 0, false); err != nil {
		return errors.Wrap(err, "set Qos")
	}

	if err := ch.Confirm(false); err != nil {
		return errors.Wrap(err, "confirm mode")
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

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
			if err := s.handleDelivery(ctx, ch, confirms, &d, handler); err != nil {
				return err
			}
		}
	}
}

func (s *Subscriber) handleDelivery(ctx context.Context, ch *amqp.Channel, confirms <-chan amqp.Confirmation, d *amqp.Delivery, handler queue.Handler) error {
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

	// Ждём подтверждения от брокера перед Ack: гарантирует что retry-очередь приняла сообщение
	if c := <-confirms; !c.Ack {
		span.SetStatus(codes.Error, "retry-publish nacked by broker")
		if nackErr := ch.Nack(d.DeliveryTag, false, true); nackErr != nil {
			return errors.Wrap(nackErr, "nack requeue after broker nack")
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
		Headers:    headers,
		Body:       msg.Body,
		DeathCount: deathCount(msg),
	}
}
