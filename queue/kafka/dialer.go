package kafka

import (
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
)

var ErrConnectionClosed = errors.New("connection is closed")

// Dialer управляет подключением к Kafka
type Dialer struct {
	dialer *kafka.Dialer
	cfg    Config
	logger *slog.Logger
	closed bool
}

// Option определяет функцию для настройки Dialer
type Option func(*Dialer)

// WithLogger устанавливает логгер для Dialer
func WithLogger(logger *slog.Logger) Option {
	return func(d *Dialer) {
		if logger != nil {
			d.logger = logger.WithGroup("kafka")
		}
	}
}

// NewDialer создает новый Dialer для работы с Kafka
func NewDialer(cfg Config, opts ...Option) *Dialer {
	d := &Dialer{
		cfg: cfg,
		dialer: &kafka.Dialer{
			Timeout:   10 * time.Second,
			DualStack: true,
		},
	}

	// Применяем опции
	for _, opt := range opts {
		opt(d)
	}

	// Устанавливаем значения по умолчанию
	if d.logger == nil {
		d.logger = slog.Default().WithGroup("kafka")
	}

	return d
}

// NewDefaultDialer создает Dialer с параметрами по умолчанию
func NewDefaultDialer(brokers []string) *Dialer {
	return NewDialer(Config{Brokers: brokers})
}

// GetDialer возвращает базовый kafka.Dialer
func (d *Dialer) GetDialer() *kafka.Dialer {
	return d.dialer
}

// Close закрывает соединение с Kafka
func (d *Dialer) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	d.logger.Info("Kafka dialer closed")
	return nil
}

// GetBrokers возвращает список брокеров Kafka
func (d *Dialer) GetBrokers() []string {
	return d.cfg.Brokers
}
