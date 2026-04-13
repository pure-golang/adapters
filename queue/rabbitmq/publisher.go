package rabbitmq

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/pure-golang/adapters/queue"
	"github.com/pure-golang/adapters/queue/encoders"
)

var _ queue.Publisher = (*Publisher)(nil)

type Publisher struct {
	logger   *slog.Logger
	mx       sync.Mutex
	dialer   channelProvider
	cfg      PublisherConfig
	channel  *amqp.Channel
	closed   <-chan *amqp.Error
	confirms <-chan amqp.Confirmation
}

type DeliveryMode uint8

const (
	Transient  = DeliveryMode(amqp.Transient)
	Persistent = DeliveryMode(amqp.Persistent)
)

// PublisherConfig - config that can be passed to Publisher constructor.
type PublisherConfig struct {
	Exchange, RoutingKey string
	DeliveryMode         DeliveryMode
	Encoder              queue.Encoder
	MessageTTL           time.Duration // precision to milliseconds
	// PublisherConfirms задаёт размер батча broker-подтверждений.
	// 0 (default) — confirm mode отключён, fire-and-forget.
	// 1 — подтверждение на каждое сообщение (1 RTT/msg).
	// N > 1 — батч: публикуется N сообщений, затем ждём N confirm'ов (1 RTT/N msg).
	PublisherConfirms int
}

// NewPublisher2 создаёт Publisher, принимая зависимость через интерфейс channelProvider.
// Предпочтительный конструктор: позволяет подменять dialer в тестах без конкретного *Dialer.
func NewPublisher2(cfg PublisherConfig, dialer channelProvider) *Publisher {
	if cfg.Encoder == nil {
		cfg.Encoder = encoders.JSON{}
	}
	if cfg.DeliveryMode == 0 {
		cfg.DeliveryMode = Persistent
	}

	closed := make(chan *amqp.Error, 1)
	close(closed)

	return &Publisher{
		logger: dialer.Logger().WithGroup("publisher").
			With("exchange", cfg.Exchange).With("routing_key", cfg.RoutingKey),
		dialer: dialer,
		cfg:    cfg,
		closed: closed,
	}
}

// NewPublisher создаёт Publisher с конкретным *Dialer.
//
// Deprecated: используйте [NewPublisher2], который принимает интерфейс [channelProvider].
func NewPublisher(dialer *Dialer, cfg PublisherConfig) *Publisher {
	return NewPublisher2(cfg, dialer)
}

// Publish messages to queue. Method is sync.
func (p *Publisher) Publish(ctx context.Context, messages ...queue.Message) error {
	p.mx.Lock()
	select {
	case <-p.closed:
		channel, err := p.dialer.Channel()
		if err != nil {
			p.mx.Unlock()
			p.logger.With("error", err).Error("failed to open channel")
			return err
		}
		if p.cfg.PublisherConfirms > 0 {
			if err := channel.Confirm(false); err != nil {
				channel.Close()
				p.mx.Unlock()
				return errors.Wrap(err, "confirm mode")
			}
			p.confirms = channel.NotifyPublish(make(chan amqp.Confirmation, p.cfg.PublisherConfirms))
		}
		p.channel = channel
		p.closed = p.channel.NotifyClose(make(chan *amqp.Error, 1))
		p.logger.Debug("channel reconnected")
	default:
	}

	if p.cfg.PublisherConfirms == 0 {
		p.mx.Unlock()
		for _, msg := range messages {
			if err := p.publish(ctx, msg); err != nil {
				return err
			}
		}
		return nil
	}

	// confirms mode: mutex held for entire batch to serialize publish+confirm pairs
	batchSize := p.cfg.PublisherConfirms
	pending := 0
	var publishErr error
	for _, msg := range messages {
		if publishErr = p.publish(ctx, msg); publishErr != nil {
			break
		}
		pending++
		if pending == batchSize {
			if publishErr = p.drainConfirms(pending); publishErr != nil {
				break
			}
			pending = 0
		}
	}
	if publishErr == nil && pending > 0 {
		publishErr = p.drainConfirms(pending)
	}
	p.mx.Unlock()
	return publishErr
}

func (p *Publisher) drainConfirms(n int) error {
	for range n {
		if c := <-p.confirms; !c.Ack {
			return errors.New("publish nacked by broker")
		}
	}
	return nil
}

func (p *Publisher) publish(ctx context.Context, msg queue.Message) error {
	ctx, span := tracer.Start(ctx, "RabbitMQ", trace.WithSpanKind(trace.SpanKindProducer))
	defer span.End()
	prop := otel.GetTextMapPropagator()

	body, err := msg.EncodeValue(p.cfg.Encoder)
	if err != nil {
		return err
	}

	amqpMsg := amqp.Publishing{
		ContentType:  p.cfg.Encoder.ContentType(),
		MessageId:    uuid.NewString(),
		DeliveryMode: uint8(p.cfg.DeliveryMode),
		Body:         body,
		Headers:      amqp.Table{},
	}
	for k, v := range msg.Headers {
		amqpMsg.Headers[k] = v
	}
	if p.cfg.MessageTTL > 0 {
		amqpMsg.Expiration = strconv.FormatInt(p.cfg.MessageTTL.Milliseconds(), 10)
	}
	if msg.TTL > 0 {
		amqpMsg.Expiration = strconv.FormatInt(msg.TTL.Milliseconds(), 10)
	}

	prop.Inject(ctx, tableCarrier(amqpMsg.Headers))

	routingKey := p.cfg.RoutingKey
	if msg.Topic != "" {
		routingKey = msg.Topic
	}

	err = p.channel.PublishWithContext(
		ctx,
		p.cfg.Exchange,
		routingKey,
		false,
		false,
		amqpMsg,
	)

	span.SetAttributes(
		attribute.String("id", amqpMsg.MessageId),
		attribute.String("exchange", p.cfg.Exchange),
		attribute.String("key", routingKey),
		attribute.String("body", string(amqpMsg.Body)),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		p.logger.With("error", err).Error("publish failed")
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}
