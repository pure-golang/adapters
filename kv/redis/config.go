package redis

import (
	"context"
	"time"
)

// Config содержит конфигурацию для подключения к Redis
type Config struct {
	Addr            string        // Адрес Redis сервера (хост:порт)
	Password        string        // Пароль для подключения
	DB              int           // Номер базы данных
	MaxRetries      int           // Максимальное количество попыток повтора
	MinRetryBackoff time.Duration // Минимальная задержка между повторами
	MaxRetryBackoff time.Duration // Максимальная задержка между повторами
	DialTimeout     time.Duration // Таймаут установки соединения
	ReadTimeout     time.Duration // Таймаут чтения
	WriteTimeout    time.Duration // Таймаут записи
	PoolSize        int           // Размер пула соединений
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
