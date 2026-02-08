package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pure-golang/adapters/queue"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ConsumeRetryInterval интервал между попытками переподключения
	ConsumeRetryInterval = 5 * time.Second

	// DefaultLastMessageTimeout время ожидания последнего сообщения перед переподключением
	DefaultLastMessageTimeout = time.Hour
)

var _ queue.Subscriber = (*Subscriber)(nil)

// Subscriber реализует интерфейс queue.Subscriber для Kafka
type Subscriber struct {
	topic              string
	mx                 sync.Mutex
	dialer             *Dialer
	cfg                SubscriberConfig
	groupID            string
	logger             *slog.Logger
	close              chan struct{}
	wg                 sync.WaitGroup
	reader             *kafka.Reader
	lastMessageTime    time.Time
	lastMessageTimeout time.Duration
}

// SubscriberConfig содержит параметры для Subscriber
type SubscriberConfig struct {
	Name          string        // имя потребителя (для логирования)
	PrefetchCount int           // максимальное количество сообщений, обрабатываемых одновременно
	MaxTryNum     int           // максимальное количество попыток обработки сообщения (-1 для бесконечных попыток)
	Backoff       time.Duration // время ожидания между попытками
}

// NewDefaultSubscriber создает Subscriber с параметрами по умолчанию
func NewDefaultSubscriber(dialer *Dialer, topic string) *Subscriber {
	return NewSubscriber(dialer, topic, SubscriberConfig{})
}

// NewSubscriber создает новый Subscriber для Kafka
func NewSubscriber(dialer *Dialer, topic string, cfg SubscriberConfig) *Subscriber {
	if cfg.Name == "" {
		cfg.Name = uuid.NewString()
	}
	if cfg.PrefetchCount <= 0 {
		cfg.PrefetchCount = 1
	}
	if cfg.MaxTryNum == 0 {
		cfg.MaxTryNum = 3
	}
	if cfg.Backoff == 0 {
		cfg.Backoff = 5 * time.Second
	}

	logger := dialer.logger.With("subscriber", cfg.Name).With("topic", topic)

	groupID := dialer.cfg.GroupID
	if groupID == "" {
		groupID = cfg.Name // используем имя потребителя как дефолтный group ID
	}

	return &Subscriber{
		topic:              topic,
		dialer:             dialer,
		cfg:                cfg,
		groupID:            groupID,
		logger:             logger,
		close:              make(chan struct{}),
		lastMessageTimeout: DefaultLastMessageTimeout,
	}
}

// Listen начинает слушать сообщения из Kafka
func (s *Subscriber) Listen(handler queue.Handler) {
	s.wg.Add(1)
	defer s.wg.Done()

	s.logger.Info("listening...")

	for {
		needRestart, err := s.listen(handler)
		if !needRestart {
			return
		}

		if err != nil {
			s.logger.With("error", err.Error()).Error("listen error")
		}

		time.Sleep(ConsumeRetryInterval)
	}
}

// listen обрабатывает сообщения из одной сессии чтения
func (s *Subscriber) listen(handler queue.Handler) (bool, error) {
	reader, err := s.getReader(s.topic)
	if err != nil {
		return true, errors.Wrap(err, "failed to create reader")
	}
	defer s.closeReader()

	s.mx.Lock()
	s.reader = reader
	s.mx.Unlock()

	for {
		select {
		case <-s.close:
			return false, nil

		case <-time.After(s.lastMessageTimeout):
			if time.Since(s.lastMessageTime) > s.lastMessageTimeout {
				s.logger.Warn("timeout waiting for messages, reconnecting")
				return true, nil
			}

		default:
			// Используем context с timeout для операций чтения
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			msg, err := s.reader.ReadMessage(ctx)
			cancel()

			if err != nil {
				// Проверяем, была ли ошибка связана с закрытием
				select {
				case <-s.close:
					return false, nil
				default:
					return true, errors.Wrap(err, "failed to read message")
				}
			}

			s.lastMessageTime = time.Now()

			// Обрабатываем сообщение с retry внутри этой сессии
			if err := s.handleMessageWithRetry(msg, handler); err != nil {
				// При ошибке в retry возвращаем true для перезапуска
				return true, err
			}
		}
	}
}

// handleMessageWithRetry обрабатывает одно сообщение с retry в рамках одной сессии
func (s *Subscriber) handleMessageWithRetry(msg kafka.Message, handler queue.Handler) error {
	var lastErr error
	maxAttempts := s.cfg.MaxTryNum

	if maxAttempts < 0 {
		// Бесконечные попытки - обрабатываем как одну попытку с последующим возвратом ошибки
		maxAttempts = 1
	}

	// Извлекаем заголовки один раз
	headers := make(map[string]string)
	for _, h := range msg.Headers {
		headers[h.Key] = string(h.Value)
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Извлекаем контекст трейсинга из заголовков
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), headersCarrier(headers))
		ctx, span := tracer.Start(ctx, fmt.Sprintf("Kafka.Consume.%s", msg.Topic), trace.WithSpanKind(trace.SpanKindConsumer))

		if attempt > 1 {
			span.SetAttributes(attribute.Int("retry_attempt", attempt))
		}

		span.SetAttributes(
			attribute.String("topic", msg.Topic),
			attribute.Int("partition", int(msg.Partition)),
			attribute.Int64("offset", msg.Offset),
			attribute.Int("body_size", len(msg.Value)),
			attribute.Int("headers_count", len(msg.Headers)),
		)

		// Создаем queue.Delivery
		delivery := queue.Delivery{
			Headers: headers,
			Body:    msg.Value,
		}

		// Вызываем обработчик
		shouldRetry, err := handler(ctx, delivery)

		if err == nil {
			// Сообщение успешно обработано
			span.SetStatus(codes.Ok, "")
			span.End()
			return nil
		}

		// Сохраняем последнюю ошибку
		lastErr = err

		// Произошла ошибка
		s.logger.With("error", err.Error(), "attempt", attempt).Error("handle message error")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()

		// Если ошибка не retry-able или это последняя попытка
		if !shouldRetry || attempt >= maxAttempts {
			break
		}

		// Ждем перед следующей попыткой
		select {
		case <-s.close:
			return errors.New("subscriber closed during retry")
		case <-time.After(s.cfg.Backoff):
		}
	}

	// Если мы используем MaxTryNum < 0 (бесконечные попытки), возвращаем ошибку для перезапуска сессии
	if s.cfg.MaxTryNum < 0 {
		return errors.Wrap(lastErr, "message processing failed (will retry on reconnect)")
	}

	// Все попытки исчерпаны, возвращаем ошибку
	return errors.Wrapf(lastErr, "message processing failed after %d attempts", maxAttempts)
}

// Close останавливает потребителя и закрывает все ресурсы
func (s *Subscriber) Close() error {
	s.logger.Info("closing subscriber...")

	select {
	case <-s.close:
		// Уже закрыт
	default:
		close(s.close)
	}

	// Ждем завершения listen() сначала - он закроет reader через defer
	s.wg.Wait()

	// Закрываем reader после того, как все горутины завершились
	s.closeReader()
	s.logger.Info("subscriber closed")
	return nil
}

// getReader возвращает или создает reader для указанной темы
func (s *Subscriber) getReader(topic string) (*kafka.Reader, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.reader != nil {
		return s.reader, nil
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        s.dialer.cfg.Brokers,
		GroupID:        s.groupID,
		Topic:          topic,
		MinBytes:       1,    // 1B - fetch immediately when any message is available
		MaxBytes:       10e6, // 10MB
		MaxWait:        1 * time.Second,
		CommitInterval: 1 * time.Second, // Автоматический коммит каждую секунду
		Logger:         kafka.LoggerFunc(s.dialer.logger.Info),
		ErrorLogger:    kafka.LoggerFunc(s.dialer.logger.Error),
	})

	return reader, nil
}

// closeReader закрывает reader, если он существует
func (s *Subscriber) closeReader() {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.reader != nil {
		if err := s.reader.Close(); err != nil {
			s.logger.With("error", err.Error()).Error("failed to close reader")
		}
		s.reader = nil
	}
}
