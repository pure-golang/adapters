package rabbitmq

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// dialer определяет методы для подключения к RabbitMQ и открытия каналов.
//
//go:generate mockery --name=dialer --exported
type dialer interface {
	Connect() error
	Channel() (*amqp.Channel, error)
}

// Connector управляет первичным подключением к RabbitMQ с повторными попытками.
type Connector struct {
	dialer dialer
	log    *slog.Logger
}

// NewConnector создаёт новый Connector.
func NewConnector(dialer dialer) *Connector {
	return &Connector{
		dialer: dialer,
		log:    slog.Default().With("module", "rabbitmq.connector"),
	}
}

// ConnectWithRetry пробует подключиться к RabbitMQ пока ctx не отменён.
func (c *Connector) ConnectWithRetry(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "rabbitmq.ConnectWithRetry", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	for attempt := 1; ; attempt++ {
		err := c.dialer.Connect()
		if err == nil {
			span.SetStatus(codes.Ok, "")
			return nil
		}
		c.log.Warn("RabbitMQ connection attempt failed", slog.Int("attempt", attempt), slog.Any("err", err))
		select {
		case <-ctx.Done():
			span.RecordError(ctx.Err())
			span.SetStatus(codes.Error, ctx.Err().Error())
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// WaitForExchange ожидает появления exchange с пассивной проверкой.
// Используется если exchange создаётся внешним сервисом.
func (c *Connector) WaitForExchange(ctx context.Context, exchangeName string) error {
	ctx, span := tracer.Start(ctx, "rabbitmq.WaitForExchange", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	c.log.Info("waiting for exchange", slog.String("exchange", exchangeName))
	for {
		err := c.tryCheckExchange(exchangeName)
		if err == nil {
			c.log.Info("exchange is ready", slog.String("exchange", exchangeName))
			return nil
		}
		c.log.Debug("exchange not ready", slog.String("exchange", exchangeName), slog.Any("err", err))
		select {
		case <-ctx.Done():
			span.RecordError(ctx.Err())
			span.SetStatus(codes.Error, ctx.Err().Error())
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// WaitForQueue ожидает появления очереди с пассивной проверкой.
// Используется если очередь создаётся внешним сервисом.
func (c *Connector) WaitForQueue(ctx context.Context, queueName string) error {
	ctx, span := tracer.Start(ctx, "rabbitmq.WaitForQueue", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	c.log.Info("waiting for queue", slog.String("queue", queueName))
	for {
		err := c.tryCheckQueue(queueName)
		if err == nil {
			c.log.Info("queue is ready", slog.String("queue", queueName))
			return nil
		}
		c.log.Debug("queue not ready", slog.String("queue", queueName), slog.Any("err", err))
		select {
		case <-ctx.Done():
			span.RecordError(ctx.Err())
			span.SetStatus(codes.Error, ctx.Err().Error())
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// tryCheckExchange пассивно проверяет наличие exchange.
// Каждый вызов открывает новый канал, т.к. passive declare при ошибке NOT_FOUND
// закрывает канал на уровне протокола.
func (c *Connector) tryCheckExchange(exchangeName string) (err error) {
	ch, err := c.dialer.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to open channel")
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, "failed to close channel")
		}
	}()
	return errors.Wrap(
		ch.ExchangeDeclarePassive(exchangeName, "topic", true, false, false, false, nil),
		"failed to declare exchange passively",
	)
}

// tryCheckQueue открывает временный канал и пассивно проверяет существование очереди.
// Каждый вызов открывает новый канал по той же причине, что и tryCheckExchange.
func (c *Connector) tryCheckQueue(queueName string) (err error) {
	ch, err := c.dialer.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to open channel")
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, "failed to close channel")
		}
	}()
	_, err = ch.QueueDeclarePassive(queueName, true, false, false, false, nil)
	return errors.Wrap(err, "failed to declare queue passively")
}
