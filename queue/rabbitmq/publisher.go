package rabbitmq

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/pure-golang/adapters/queue"
	"github.com/pure-golang/adapters/queue/encoders"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ queue.Publisher = (*Publisher)(nil)

type Publisher struct {
	mx      sync.Mutex
	dialer  *Dialer
	cfg     PublisherConfig
	channel *amqp.Channel
	closed  <-chan *amqp.Error
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
}

func NewPublisher(dialer *Dialer, cfg PublisherConfig) *Publisher {
	if cfg.Encoder == nil {
		cfg.Encoder = encoders.JSON{}
	}
	if cfg.DeliveryMode == 0 {
		cfg.DeliveryMode = Persistent
	}

	closed := make(chan *amqp.Error, 1)
	close(closed)

	return &Publisher{
		dialer: dialer,
		cfg:    cfg,
		closed: closed,
	}
}

// Publish messages to queue. Method is sync.
func (p *Publisher) Publish(ctx context.Context, messages ...queue.Message) error {
	p.mx.Lock()
	select {
	case <-p.closed:
		channel, err := p.dialer.Channel()
		if err != nil {
			defer p.mx.Unlock()
			return err
		}
		p.channel = channel
		p.closed = p.channel.NotifyClose(make(chan *amqp.Error, 1))
	default:
	}
	p.mx.Unlock()

	for _, msg := range messages {
		if err := p.publish(ctx, msg); err != nil {
			return err
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

	err = p.channel.Publish(
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
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}
