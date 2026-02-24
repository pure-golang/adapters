package redis

import (
	"context"
	"time"
)

// Config содержит конфигурацию для подключения к Redis
type Config struct {
	Addr            string        `envconfig:"REDIS_ADDR" default:"localhost:6379"`     // Адрес Redis сервера (хост:порт)
	Password        string        `envconfig:"REDIS_PASSWORD"`                          // Пароль для подключения
	DB              int           `envconfig:"REDIS_DB" default:"0"`                    // Номер базы данных
	MaxRetries      int           `envconfig:"REDIS_MAX_RETRIES" default:"3"`           // Максимальное количество попыток повтора
	MinRetryBackoff time.Duration `envconfig:"REDIS_MIN_RETRY_BACKOFF" default:"8ms"`   // Минимальная задержка между повторами
	MaxRetryBackoff time.Duration `envconfig:"REDIS_MAX_RETRY_BACKOFF" default:"512ms"` // Максимальная задержка между повторами
	DialTimeout     time.Duration `envconfig:"REDIS_DIAL_TIMEOUT" default:"5s"`         // Таймаут установки соединения
	ReadTimeout     time.Duration `envconfig:"REDIS_READ_TIMEOUT" default:"3s"`         // Таймаут чтения
	WriteTimeout    time.Duration `envconfig:"REDIS_WRITE_TIMEOUT" default:"3s"`        // Таймаут записи
	PoolSize        int           `envconfig:"REDIS_POOL_SIZE" default:"10"`            // Размер пула соединений
}

// NewDefault создаёт Config с значениями по умолчанию
func NewDefault(cfg Config) (*Client, error) {
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.MinRetryBackoff == 0 {
		cfg.MinRetryBackoff = 8 * time.Millisecond
	}
	if cfg.MaxRetryBackoff == 0 {
		cfg.MaxRetryBackoff = 512 * time.Millisecond
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 3 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 3 * time.Second
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 10
	}

	return Connect(context.Background(), cfg)
}
