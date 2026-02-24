package kv

import (
	"context"
	"time"
)

// Store определяет интерфейс key-value хранилища
type Store interface {
	// Базовые операции
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
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
	HSet(ctx context.Context, key, field string, value any) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error

	// List операции
	LPush(ctx context.Context, key string, values ...any) error
	RPush(ctx context.Context, key string, values ...any) error
	LPop(ctx context.Context, key string) (string, error)
	RPop(ctx context.Context, key string) (string, error)
	LLen(ctx context.Context, key string) (int64, error)

	// Set операции
	SAdd(ctx context.Context, key string, members ...any) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key string, member any) (bool, error)
	SRem(ctx context.Context, key string, members ...any) error

	// Подключение
	Ping(ctx context.Context) error
	Close() error
}
