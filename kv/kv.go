package kv

import (
	"context"
	"time"

	"github.com/pure-golang/adapters/env"
	"github.com/pure-golang/adapters/kv/noop"
	"github.com/pure-golang/adapters/kv/redis"
	"github.com/pkg/errors"
)

// Provider определяет тип key-value хранилища
type Provider string

const (
	ProviderRedis Provider = "redis" // Redis хранилище
	ProviderNoop  Provider = "noop"  // No-op реализация для тестов
)

// Config содержит конфигурацию для key-value хранилища
type Config struct {
	Provider Provider `envconfig:"KV_PROVIDER" default:"noop"`
	// Redis конфигурация (используется когда ProviderRedis)
	RedisAddr         string        `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword     string        `envconfig:"REDIS_PASSWORD"`
	RedisDB           int           `envconfig:"REDIS_DB" default:"0"`
	RedisMaxRetries   int           `envconfig:"REDIS_MAX_RETRIES" default:"3"`
	RedisDialTimeout  time.Duration `envconfig:"REDIS_DIAL_TIMEOUT" default:"5s"`
	RedisReadTimeout  time.Duration `envconfig:"REDIS_READ_TIMEOUT" default:"3s"`
	RedisWriteTimeout time.Duration `envconfig:"REDIS_WRITE_TIMEOUT" default:"3s"`
	RedisPoolSize     int           `envconfig:"REDIS_POOL_SIZE" default:"10"`
}

// Store определяет интерфейс key-value хранилища
type Store interface {
	// Базовые операции
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)

	// Операции с счётчиками
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)

	// TTL операции
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Hash операции
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key, field string, value interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error

	// List операции
	LPush(ctx context.Context, key string, values ...interface{}) error
	RPush(ctx context.Context, key string, values ...interface{}) error
	LPop(ctx context.Context, key string) (string, error)
	RPop(ctx context.Context, key string) (string, error)
	LLen(ctx context.Context, key string) (int64, error)

	// Set операции
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SRem(ctx context.Context, key string, members ...interface{}) error

	// Подключение
	Ping(ctx context.Context) error
	Close() error
}

// NewDefault создаёт инстанс Store, читая конфигурацию из переменных окружения
func NewDefault() (Store, error) {
	var cfg Config
	if err := env.InitConfig(&cfg); err != nil {
		return nil, errors.Wrap(err, "failed to init config")
	}

	switch cfg.Provider {
	case ProviderRedis:
		redisCfg := redis.Config{
			Addr:         cfg.RedisAddr,
			Password:     cfg.RedisPassword,
			DB:           cfg.RedisDB,
			MaxRetries:   cfg.RedisMaxRetries,
			DialTimeout:  cfg.RedisDialTimeout,
			ReadTimeout:  cfg.RedisReadTimeout,
			WriteTimeout: cfg.RedisWriteTimeout,
			PoolSize:     cfg.RedisPoolSize,
		}
		return redis.NewDefault(redisCfg)
	case ProviderNoop:
		return noop.NewStore(), nil
	default:
		return nil, errors.Errorf("unknown kv provider: %s", cfg.Provider)
	}
}

// InitDefault создаёт и возвращает Store (для использования в main)
func InitDefault() (Store, error) {
	return NewDefault()
}
