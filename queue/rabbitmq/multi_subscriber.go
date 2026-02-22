package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pure-golang/adapters/queue"
)

// QueueHandler связывает имя очереди с функцией-обработчиком.
type QueueHandler struct {
	// QueueName — очередь, из которой читаются сообщения.
	QueueName string
	// RetryQueueName — очередь, куда публикуются неуспешные сообщения для отложенной повторной обработки.
	// По умолчанию QueueName + ".retry".
	RetryQueueName string
	// Handler обрабатывает каждое доставленное сообщение.
	Handler queue.Handler
}

// MultiQueueOptions настраивает MultiQueueSubscriber.
type MultiQueueOptions struct {
	// PrefetchCount — значение Qos prefetch, применяемое глобально ко всем
	// консьюмерам на общем канале (global=true).
	// Значение 1 гарантирует не более одного неподтверждённого сообщения одновременно.
	// По умолчанию 1.
	PrefetchCount int
	// MaxRetries — максимальное число попыток доставки, после чего сообщение
	// отправляется в dead-letter queue. Должно быть > 0.
	MaxRetries int
	// MessageTimeout ограничивает время выполнения обработчика для одного сообщения.
	// 0 означает отсутствие таймаута.
	MessageTimeout time.Duration
	// ReconnectDelay — пауза между попытками переподключения при сбое соединения или канала.
	// По умолчанию 5s.
	ReconnectDelay time.Duration
}

// MultiQueueSubscriber читает сообщения из нескольких очередей на одном AMQP-канале.
// Одна горутина обрабатывает доставки из всех очередей последовательно, что
// гарантирует не более одного сообщения в обработке одновременно (при PrefetchCount=1).
// Соответствует модели из RABBIT2.md для CPU-интенсивных задач.
type MultiQueueSubscriber struct {
	dialer *Dialer
	cfg    MultiQueueOptions
	logger *slog.Logger
}

func NewMultiQueueSubscriber(dialer *Dialer, opts MultiQueueOptions) *MultiQueueSubscriber {
	if opts.PrefetchCount <= 0 {
		opts.PrefetchCount = 1
	}
	if opts.ReconnectDelay <= 0 {
		opts.ReconnectDelay = 5 * time.Second
	}
	return &MultiQueueSubscriber{
		dialer: dialer,
		cfg:    opts,
		logger: dialer.options.Logger.With("component", "multi_subscriber"),
	}
}

// Listen запускает чтение из всех указанных очередей. Блокируется до отмены ctx.
// При потере соединения или канала автоматически переподключается.
func (s *MultiQueueSubscriber) Listen(ctx context.Context, handlers ...QueueHandler) error {
	// Нормализуем имена retry-очередей один раз.
	for i := range handlers {
		if handlers[i].RetryQueueName == "" {
			handlers[i].RetryQueueName = handlers[i].QueueName + ".retry"
		}
	}

	s.logger.Info("listening...")
	for {
		if err := s.connectAndConsume(ctx, handlers); err != nil {
			s.logger.With("error", err.Error()).Error("connection lost")
		}
		if ctx.Err() != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.cfg.ReconnectDelay):
		}
	}
}

type taggedDelivery struct {
	d amqp.Delivery
	h QueueHandler
}

func (s *MultiQueueSubscriber) connectAndConsume(ctx context.Context, handlers []QueueHandler) error {
	ch, err := s.dialer.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	// global=true: лимит prefetch применяется ко всем консьюмерам на этом канале.
	if err := ch.Qos(s.cfg.PrefetchCount, 0, true); err != nil {
		return fmt.Errorf("set Qos: %w", err)
	}

	type namedConsumer struct {
		deliveries <-chan amqp.Delivery
		handler    QueueHandler
	}

	consumers := make([]namedConsumer, 0, len(handlers))
	for _, h := range handlers {
		tag := uuid.NewString()
		dels, err := ch.Consume(h.QueueName, tag, false, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("consume %q: %w", h.QueueName, err)
		}
		consumers = append(consumers, namedConsumer{dels, h})
	}

	notifyClose := ch.NotifyClose(make(chan *amqp.Error, 1))

	// Fan-in: объединяем каналы доставки всех очередей в один.
	// Каждая очередь получает свою горутину для пересылки в объединённый канал.
	// Основной цикл остаётся однопоточным (нет конкурентных AMQP-операций).
	fanCtx, cancelFan := context.WithCancel(ctx)
	defer cancelFan()

	merged := make(chan taggedDelivery)
	var wg sync.WaitGroup
	for _, c := range consumers {
		wg.Add(1)
		go func(c namedConsumer) {
			defer wg.Done()
			for d := range c.deliveries {
				select {
				case merged <- taggedDelivery{d, c.handler}:
				case <-fanCtx.Done():
					return
				}
			}
		}(c)
	}
	go func() {
		wg.Wait()
		close(merged)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case amqpErr, ok := <-notifyClose:
			if !ok || amqpErr == nil {
				return fmt.Errorf("channel closed")
			}
			return fmt.Errorf("channel closed: %w", amqpErr)
		case td, ok := <-merged:
			if !ok {
				return nil
			}
			s.handleDelivery(ctx, ch, &td.d, td.h)
		}
	}
}

func (s *MultiQueueSubscriber) handleDelivery(ctx context.Context, ch *amqp.Channel, d *amqp.Delivery, h QueueHandler) {
	hCtx := otel.GetTextMapPropagator().Extract(ctx, tableCarrier(d.Headers))
	hCtx, span := tracer.Start(hCtx, h.QueueName, trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	span.SetAttributes(
		attribute.String("id", d.MessageId),
		attribute.String("queue", h.QueueName),
	)

	handlerCtx := hCtx
	if s.cfg.MessageTimeout > 0 {
		var cancel context.CancelFunc
		handlerCtx, cancel = context.WithTimeout(hCtx, s.cfg.MessageTimeout)
		defer cancel()
	}

	_, err := h.Handler(handlerCtx, newDelivery(d))
	if err == nil {
		if ackErr := ch.Ack(d.DeliveryTag, false); ackErr != nil {
			span.SetStatus(codes.Error, ackErr.Error())
			s.logger.With("error", ackErr.Error()).Error("ack failed")
		}
		return
	}

	s.logger.With("error", err.Error(), "queue", h.QueueName).Error("handle message failed")
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	if deathCount(d) >= s.cfg.MaxRetries {
		if nackErr := ch.Nack(d.DeliveryTag, false, false); nackErr != nil {
			s.logger.With("error", nackErr.Error()).Error("nack to DLQ failed")
		}
		return
	}

	msg := amqp.Publishing{
		MessageId:    d.MessageId,
		ContentType:  d.ContentType,
		DeliveryMode: d.DeliveryMode,
		Body:         d.Body,
		Headers:      d.Headers,
	}
	if pubErr := ch.Publish("", h.RetryQueueName, false, false, msg); pubErr != nil {
		span.SetStatus(codes.Error, pubErr.Error())
		// Запасной вариант: requeue, чтобы сообщение не потерялось
		if nackErr := ch.Nack(d.DeliveryTag, false, true); nackErr != nil {
			s.logger.With("error", nackErr.Error()).Error("nack requeue after retry-publish failure")
		}
		return
	}

	if ackErr := ch.Ack(d.DeliveryTag, false); ackErr != nil {
		span.SetStatus(codes.Error, ackErr.Error())
		s.logger.With("error", ackErr.Error()).Error("ack after retry publish failed")
	}
}
