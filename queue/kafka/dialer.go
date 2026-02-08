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

// DialerOptions содержит опции для создания Dialer
type DialerOptions struct {
	Logger *slog.Logger
}

// NewDialer создает новый Dialer для работы с Kafka
func NewDialer(cfg Config, options *DialerOptions) *Dialer {
	if options == nil {
		options = new(DialerOptions)
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	options.Logger = options.Logger.WithGroup("kafka")

	return &Dialer{
		cfg:    cfg,
		logger: options.Logger,
		dialer: &kafka.Dialer{
			Timeout:   10 * time.Second,
			DualStack: true,
		},
	}
}

// NewDefaultDialer создает Dialer с параметрами по умолчанию
func NewDefaultDialer(brokers []string) *Dialer {
	return NewDialer(Config{Brokers: brokers}, nil)
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
