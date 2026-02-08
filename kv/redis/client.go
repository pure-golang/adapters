package redis

import (
	"context"
	"time"

	"log/slog"

	"github.com/pkg/errors"
	rclient "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/codes"
)

// Client представляет клиент Redis
type Client struct {
	*rclient.Client
	cfg    Config
	logger *slog.Logger
}

// Connect создаёт новое подключение к Redis
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	logger := newLogger(nil)
	logger.Debug("connecting to redis", "addr", cfg.Addr)

	rdb := rclient.NewClient(&rclient.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolSize:        cfg.PoolSize,
	})

	client := &Client{
		Client: rdb,
		cfg:    cfg,
		logger: logger,
	}

	// Проверяем подключение
	if err := client.Ping(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to ping redis")
	}

	logger.Info("connected to redis", "addr", cfg.Addr)
	return client, nil
}

// Close закрывает подключение к Redis
func (c *Client) Close() error {
	_, span := startSpan(context.Background(), "Close", "", c.cfg.DB)
	defer span.End()

	if c.Client == nil {
		return nil
	}

	if err := c.Client.Close(); err != nil {
		// Игнорируем ошибку закрытия уже закрытого соединения
		if err.Error() != "redis: client is closed" {
			recordError(span, err)
			return errors.Wrap(err, "failed to close redis connection")
		}
	}

	// Помечаем клиента как закрытого
	c.Client = nil
	c.logger.Debug("redis connection closed")
	return nil
}

// Ping проверяет подключение к Redis
func (c *Client) Ping(ctx context.Context) error {
	ctx, span := startSpan(ctx, "Ping", "", c.cfg.DB)
	defer span.End()

	if err := c.Client.Ping(ctx).Err(); err != nil {
		recordError(span, err)
		return errors.Wrap(err, "failed to ping redis")
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// Get получает значение по ключу
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	ctx, span := startSpan(ctx, "Get", key, c.cfg.DB)
	defer span.End()

	val, err := c.Client.Get(ctx, key).Result()
	if err != nil {
		if err == rclient.Nil {
			recordError(span, ErrKeyNotFound)
			return "", ErrKeyNotFound
		}
		recordError(span, err)
		return "", errors.Wrapf(err, "failed to get key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return val, nil
}

// Set устанавливает значение по ключу с опциональным TTL
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	ctx, span := startSpan(ctx, "Set", key, c.cfg.DB)
	defer span.End()

	if err := c.Client.Set(ctx, key, value, expiration).Err(); err != nil {
		recordError(span, err)
		return errors.Wrapf(err, "failed to set key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// Delete удаляет ключи
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	ctx, span := startSpan(ctx, "Delete", "", c.cfg.DB)
	defer span.End()

	if len(keys) == 0 {
		return nil
	}

	if err := c.Client.Del(ctx, keys...).Err(); err != nil {
		recordError(span, err)
		return errors.Wrap(err, "failed to delete keys")
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// Exists проверяет существование ключей
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	ctx, span := startSpan(ctx, "Exists", "", c.cfg.DB)
	defer span.End()

	if len(keys) == 0 {
		return 0, nil
	}

	count, err := c.Client.Exists(ctx, keys...).Result()
	if err != nil {
		recordError(span, err)
		return 0, errors.Wrap(err, "failed to check keys existence")
	}

	span.SetStatus(codes.Ok, "")
	return count, nil
}

// Incr инкрементирует значение ключа на 1
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	ctx, span := startSpan(ctx, "Incr", key, c.cfg.DB)
	defer span.End()

	val, err := c.Client.Incr(ctx, key).Result()
	if err != nil {
		recordError(span, err)
		return 0, errors.Wrapf(err, "failed to increment key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return val, nil
}

// Decr декрементирует значение ключа на 1
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	ctx, span := startSpan(ctx, "Decr", key, c.cfg.DB)
	defer span.End()

	val, err := c.Client.Decr(ctx, key).Result()
	if err != nil {
		recordError(span, err)
		return 0, errors.Wrapf(err, "failed to decrement key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return val, nil
}

// Expire устанавливает TTL для ключа
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	ctx, span := startSpan(ctx, "Expire", key, c.cfg.DB)
	defer span.End()

	if err := c.Client.Expire(ctx, key, expiration).Err(); err != nil {
		recordError(span, err)
		return errors.Wrapf(err, "failed to set expiration for key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// TTL получает оставшееся время жизни ключа
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	ctx, span := startSpan(ctx, "TTL", key, c.cfg.DB)
	defer span.End()

	ttl, err := c.Client.TTL(ctx, key).Result()
	if err != nil {
		recordError(span, err)
		return 0, errors.Wrapf(err, "failed to get ttl for key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return ttl, nil
}

// HGet получает значение поля из хеша
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	ctx, span := startSpan(ctx, "HGet", key, c.cfg.DB)
	defer span.End()

	val, err := c.Client.HGet(ctx, key, field).Result()
	if err != nil {
		if err == rclient.Nil {
			recordError(span, ErrKeyNotFound)
			return "", ErrKeyNotFound
		}
		recordError(span, err)
		return "", errors.Wrapf(err, "failed to get hash field %q from key %q", field, key)
	}

	span.SetStatus(codes.Ok, "")
	return val, nil
}

// HSet устанавливает значение поля в хеше
func (c *Client) HSet(ctx context.Context, key, field string, value interface{}) error {
	ctx, span := startSpan(ctx, "HSet", key, c.cfg.DB)
	defer span.End()

	if err := c.Client.HSet(ctx, key, field, value).Err(); err != nil {
		recordError(span, err)
		return errors.Wrapf(err, "failed to set hash field %q in key %q", field, key)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// HGetAll получает все поля и значения из хеша
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	ctx, span := startSpan(ctx, "HGetAll", key, c.cfg.DB)
	defer span.End()

	val, err := c.Client.HGetAll(ctx, key).Result()
	if err != nil {
		recordError(span, err)
		return nil, errors.Wrapf(err, "failed to get all hash fields from key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return val, nil
}

// HDel удаляет поля из хеша
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	ctx, span := startSpan(ctx, "HDel", key, c.cfg.DB)
	defer span.End()

	if len(fields) == 0 {
		return nil
	}

	if err := c.Client.HDel(ctx, key, fields...).Err(); err != nil {
		recordError(span, err)
		return errors.Wrapf(err, "failed to delete hash fields from key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// LPush добавляет значения в начало списка
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	ctx, span := startSpan(ctx, "LPush", key, c.cfg.DB)
	defer span.End()

	if len(values) == 0 {
		return nil
	}

	if err := c.Client.LPush(ctx, key, values...).Err(); err != nil {
		recordError(span, err)
		return errors.Wrapf(err, "failed to lpush to key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// RPush добавляет значения в конец списка
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	ctx, span := startSpan(ctx, "RPush", key, c.cfg.DB)
	defer span.End()

	if len(values) == 0 {
		return nil
	}

	if err := c.Client.RPush(ctx, key, values...).Err(); err != nil {
		recordError(span, err)
		return errors.Wrapf(err, "failed to rpush to key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// LPop получает и удаляет значение из начала списка
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	ctx, span := startSpan(ctx, "LPop", key, c.cfg.DB)
	defer span.End()

	val, err := c.Client.LPop(ctx, key).Result()
	if err != nil {
		if err == rclient.Nil {
			recordError(span, ErrKeyNotFound)
			return "", ErrKeyNotFound
		}
		recordError(span, err)
		return "", errors.Wrapf(err, "failed to lpop from key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return val, nil
}

// RPop получает и удаляет значение из конца списка
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	ctx, span := startSpan(ctx, "RPop", key, c.cfg.DB)
	defer span.End()

	val, err := c.Client.RPop(ctx, key).Result()
	if err != nil {
		if err == rclient.Nil {
			recordError(span, ErrKeyNotFound)
			return "", ErrKeyNotFound
		}
		recordError(span, err)
		return "", errors.Wrapf(err, "failed to rpop from key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return val, nil
}

// LLen получает длину списка
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	ctx, span := startSpan(ctx, "LLen", key, c.cfg.DB)
	defer span.End()

	length, err := c.Client.LLen(ctx, key).Result()
	if err != nil {
		recordError(span, err)
		return 0, errors.Wrapf(err, "failed to get length of list %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return length, nil
}

// SAdd добавляет члены в множество
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	ctx, span := startSpan(ctx, "SAdd", key, c.cfg.DB)
	defer span.End()

	if len(members) == 0 {
		return nil
	}

	if err := c.Client.SAdd(ctx, key, members...).Err(); err != nil {
		recordError(span, err)
		return errors.Wrapf(err, "failed to sadd to key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// SMembers получает все члены множества
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	ctx, span := startSpan(ctx, "SMembers", key, c.cfg.DB)
	defer span.End()

	members, err := c.Client.SMembers(ctx, key).Result()
	if err != nil {
		recordError(span, err)
		return nil, errors.Wrapf(err, "failed to smembers from key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return members, nil
}

// SIsMember проверяет наличие элемента в множестве
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	ctx, span := startSpan(ctx, "SIsMember", key, c.cfg.DB)
	defer span.End()

	isMember, err := c.Client.SIsMember(ctx, key, member).Result()
	if err != nil {
		recordError(span, err)
		return false, errors.Wrapf(err, "failed to check membership in key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return isMember, nil
}

// SRem удаляет члены из множества
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	ctx, span := startSpan(ctx, "SRem", key, c.cfg.DB)
	defer span.End()

	if len(members) == 0 {
		return nil
	}

	if err := c.Client.SRem(ctx, key, members...).Err(); err != nil {
		recordError(span, err)
		return errors.Wrapf(err, "failed to srem from key %q", key)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}
