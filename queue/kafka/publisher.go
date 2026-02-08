package kafka

import (
	"context"
	"sync"

	"github.com/pure-golang/adapters/queue"
	"github.com/pure-golang/adapters/queue/encoders"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ queue.Publisher = (*Publisher)(nil)

// Publisher реализует интерфейс queue.Publisher для Kafka
type Publisher struct {
	mx      sync.Mutex
	dialer  *Dialer
	cfg     PublisherConfig
	writers map[string]*kafka.Writer
	closed  bool
}

// PublisherConfig содержит параметры для Publisher
type PublisherConfig struct {
	Balancer kafka.Balancer // стратегия балансировки сообщений между партициями
	Encoder  queue.Encoder  // кодировщик сообщений (по умолчанию JSON)
}

// NewPublisher создает новый Publisher для Kafka
func NewPublisher(dialer *Dialer, cfg PublisherConfig) *Publisher {
	if cfg.Encoder == nil {
		cfg.Encoder = encoders.JSON{}
	}
	if cfg.Balancer == nil {
		cfg.Balancer = &kafka.LeastBytes{}
	}

	return &Publisher{
		dialer:  dialer,
		cfg:     cfg,
		writers: make(map[string]*kafka.Writer),
		closed:  false,
	}
}

// Publish публикует сообщения в Kafka (синхронно)
func (p *Publisher) Publish(ctx context.Context, messages ...queue.Message) error {
	for _, msg := range messages {
		if err := p.publish(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

// publish публикует одно сообщение в Kafka
func (p *Publisher) publish(ctx context.Context, msg queue.Message) error {
	ctx, span := tracer.Start(ctx, "Kafka.Publish", trace.WithSpanKind(trace.SpanKindProducer))
	defer span.End()

	prop := otel.GetTextMapPropagator()

	// Кодируем тело сообщения
	body, err := msg.EncodeValue(p.cfg.Encoder)
	if err != nil {
		return errors.Wrap(err, "failed to encode message body")
	}

	// Определяем тему (используем Topic из сообщения или дефолтное значение)
	topic := msg.Topic
	if topic == "" {
		topic = p.dialer.cfg.Brokers[0] // fallback: используем первый брокер как дефолтный topic (не идеально, но для совместимости)
	}

	// Получаем или создаем writer для темы
	writer, err := p.getWriter(topic)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Создаем Kafka message
	kafkaMsg := kafka.Message{
		Topic: topic,
		Key:   []byte(uuid.NewString()), // используем UUID как ключ для распределения по партициям
		Value: body,
	}

	// Копируем заголовки и добавляем трейсинг
	headers := make(map[string]string)
	for k, v := range msg.Headers {
		headers[k] = v
	}

	// Внедряем трейсинг в заголовки
	prop.Inject(ctx, headersCarrier(headers))
	for k, v := range headers {
		kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	// Устанавливаем атрибуты спана
	span.SetAttributes(
		attribute.String("topic", topic),
		attribute.String("key", string(kafkaMsg.Key)),
		attribute.Int("body_size", len(kafkaMsg.Value)),
		attribute.Int("headers_count", len(kafkaMsg.Headers)),
	)

	// Публикуем сообщение
	err = writer.WriteMessages(ctx, kafkaMsg)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to publish message to Kafka")
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// getWriter возвращает или создает writer для указанной темы
func (p *Publisher) getWriter(topic string) (*kafka.Writer, error) {
	p.mx.Lock()
	defer p.mx.Unlock()

	if p.closed {
		return nil, errors.New("publisher is closed")
	}

	if writer, exists := p.writers[topic]; exists {
		return writer, nil
	}

	// Создаем новый writer для темы
	// НЕ указываем Topic в Writer, чтобы можно было отправлять в разные темы через один writer
	writer := &kafka.Writer{
		Addr:        kafka.TCP(p.dialer.cfg.Brokers...),
		Balancer:    p.cfg.Balancer,
		Async:       false, // синхронная запись для надежности
		Logger:      kafka.LoggerFunc(p.dialer.logger.Info),
		ErrorLogger: kafka.LoggerFunc(p.dialer.logger.Error),
	}

	p.writers[topic] = writer
	return writer, nil
}

// Close закрывает все writer'ы и освобождает ресурсы
func (p *Publisher) Close() error {
	p.mx.Lock()
	defer p.mx.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Закрываем все writers
	for topic, writer := range p.writers {
		if err := writer.Close(); err != nil {
			p.dialer.logger.With("topic", topic, "error", err.Error()).Error("failed to close writer")
		}
	}

	p.writers = make(map[string]*kafka.Writer)
	p.dialer.logger.Info("Kafka publisher closed")
	return nil
}
