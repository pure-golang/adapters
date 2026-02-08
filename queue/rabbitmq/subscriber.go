package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/pure-golang/adapters/queue"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ConsumeRetryInterval      = 5 * time.Second
	InfiniteRetriesIndicator  = -1
	KeyCountRetries           = "x-count-retries"
	DefaultLastMessageTimeout = time.Hour
)

var _ queue.Subscriber = (*Subscriber)(nil)

// Subscriber implements queue.Subscriber interface
// Handle one task per time
type Subscriber struct {
	name               string
	queueName          string
	wg                 sync.WaitGroup
	cfg                SubscriberOptions
	dialer             *Dialer
	close              chan struct{}
	logger             *slog.Logger
	lastMessageTime    time.Time
	lastMessageTimeout time.Duration
}

type SubscriberOptions struct {
	Name                     string
	PrefetchCount, MaxTryNum int
	Backoff                  time.Duration
}

func NewDefaultSubscriber(dialer *Dialer, queueName string) *Subscriber {
	return NewSubscriber(dialer, queueName, SubscriberOptions{})
}

func NewSubscriber(dialer *Dialer, queueName string, cfg SubscriberOptions) *Subscriber {
	var name string
	if cfg.Name == "" {
		name = uuid.NewString()
	}
	if cfg.MaxTryNum <= 0 {
		cfg.MaxTryNum = 0
	}
	if cfg.Backoff == 0 {
		cfg.Backoff = 5 * time.Second
	}

	return &Subscriber{
		name:               name,
		queueName:          queueName,
		dialer:             dialer,
		logger:             dialer.options.Logger.With("subscriber", name).With("queue", queueName),
		close:              make(chan struct{}),
		cfg:                cfg,
		lastMessageTimeout: DefaultLastMessageTimeout,
	}
}

func (s *Subscriber) Listen(handler queue.Handler) {
	s.wg.Add(1)
	defer s.wg.Done()

	s.logger.Info("listening...")
	for {
		s.lastMessageTime = time.Now()
		needRestart, err := s.listen(handler)
		if !needRestart {
			return
		}

		if err != nil {
			s.logger.With("error", err.Error()).Error("s.listen error")
		}

		time.Sleep(ConsumeRetryInterval)
	}
}

func (s *Subscriber) listen(handler queue.Handler) (bool, error) {
	channel, err := s.dialer.Channel()
	if err != nil {
		return true, errors.Wrap(err, "failed to make channel")
	}
	defer func() {
		// Channel close error is not critical here as we're about to exit anyway
		// and RabbitMQ will clean up the channel on the server side
		_ = channel.Close()
	}()
	notifyClose := channel.NotifyClose(make(chan *amqp.Error, 1))
	closeChannel := make(chan error, 1)
	defer close(closeChannel)
	freezeClose := s.freezeClose(closeChannel)

	if err := channel.Qos(s.cfg.PrefetchCount, 0, false); err != nil {
		return true, errors.Wrap(err, "failed to set prefetch count")
	}

	deliveries, err := channel.Consume(
		s.queueName,
		s.cfg.Name,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return true, errors.Wrapf(err, "failed to start consuming from %q", s.queueName)
	}

	for {
		select {
		case <-s.close:
			if err := channel.Cancel(s.name, false); err != nil {
				return true, errors.Wrapf(err, "cancel consumer %q", s.name)
			}
		case amqpErr := <-notifyClose:
			if amqpErr != nil {
				return true, errors.Wrap(amqpErr, "channel is closed")
			}
		case delivery, ok := <-deliveries:
			if !ok {
				// Rest of messages is drained
				return false, nil
			}
			if err := s.handleDelivery(channel, delivery, handler); err != nil {
				return true, err
			}
		case <-freezeClose:
			s.logger.Warn("last message received. need resubscribe", "time", s.lastMessageTime.UTC())
			return true, nil
		}
	}
}

func (s *Subscriber) freezeClose(closeChanel chan error) chan struct{} {
	c := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-time.After(s.lastMessageTimeout):
				if time.Since(s.lastMessageTime) > s.lastMessageTimeout {
					c <- struct{}{}
					return
				}
			case _, ok := <-closeChanel:
				if !ok {
					return
				}
			}
		}
	}()

	return c
}

func (s *Subscriber) Close() error {
	s.logger.Info("Closing subscriber...")
	close(s.close)
	s.wg.Wait()
	return nil
}

func (s *Subscriber) handleDelivery(channel *amqp.Channel, delivery amqp.Delivery, handler queue.Handler) error {
	s.logger.Debug("handleDelivery")
	s.lastMessageTime = time.Now()
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), tableCarrier(delivery.Headers))
	ctx, span := tracer.Start(ctx, s.queueName, trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	// TODO
	//s.logger.With("trace_id", span.SpanContext().TraceID())
	//ctx = log.NewContext(ctx, logger) // pass ctx into  handler

	span.SetAttributes(
		attribute.String("id", delivery.MessageId),
		attribute.String("body", string(delivery.Body)),
		attribute.String("consumer_name", s.name),
	)

	retry, err := handler(ctx, newDelivery(delivery))
	if err == nil {
		if err := channel.Ack(delivery.DeliveryTag, false); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return errors.Wrap(err, "failed to ack")
		}
		return nil
	}

	s.logger.With("error", err.Error()).Error("Handle message")
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	// Reject non-retryable error immediately
	if !retry {
		if err := channel.Reject(delivery.DeliveryTag, false); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return errors.Wrap(err, "failed to reject")
		}
		return nil
	}

	headers := delivery.Headers
	if headers == nil {
		headers = amqp.Table{}
	}
	countRetries := headers[KeyCountRetries].(int32) // 0 if not exists

	if s.cfg.MaxTryNum != InfiniteRetriesIndicator {
		countRetries++
		if int(countRetries) >= s.cfg.MaxTryNum {
			if err := channel.Reject(delivery.DeliveryTag, false); err != nil {
				span.SetStatus(codes.Error, err.Error())
				return errors.Wrap(err, "failed to reject")
			}
			return nil
		}
	}

	headers[KeyCountRetries] = countRetries
	msg := amqp.Publishing{
		MessageId:    delivery.MessageId,
		ContentType:  delivery.ContentType,
		DeliveryMode: delivery.DeliveryMode,
		Body:         delivery.Body,
		Headers:      headers,
	}
	if err := channel.Publish("", s.queueName, false, false, msg); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to publish")
	}
	if err := channel.Ack(delivery.DeliveryTag, false); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to ack")
	}

	select {
	case <-s.close:
	case <-time.After(s.cfg.Backoff):
	}
	return nil
}

func newDelivery(msg amqp.Delivery) queue.Delivery {
	headers := make(map[string]string, len(msg.Headers))
	for k, v := range msg.Headers {
		headers[k] = fmt.Sprintf("%v", v)
	}
	return queue.Delivery{
		Headers: headers,
		Body:    msg.Body,
	}
}
