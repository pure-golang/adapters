package noop

import (
	"context"
	"time"
)

// Store представляет no-op реализацию key-value хранилища для тестов
type Store struct{}

// NewStore создаёт новый no-op Store
func NewStore() *Store {
	return &Store{}
}

// Get возвращает пустую строку без выполнения операций
func (s *Store) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

// Set не выполняет операций
func (s *Store) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

// Delete не выполняет операций
func (s *Store) Delete(ctx context.Context, keys ...string) error {
	return nil
}

// Exists возвращает 0 без выполнения операций
func (s *Store) Exists(ctx context.Context, keys ...string) (int64, error) {
	return 0, nil
}

// Incr возвращает 0 без выполнения операций
func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

// Decr возвращает 0 без выполнения операций
func (s *Store) Decr(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

// Expire не выполняет операций
func (s *Store) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}

// TTL возвращает 0 без выполнения операций
func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

// HGet возвращает пустую строку без выполнения операций
func (s *Store) HGet(ctx context.Context, key, field string) (string, error) {
	return "", nil
}

// HSet не выполняет операций
func (s *Store) HSet(ctx context.Context, key, field string, value interface{}) error {
	return nil
}

// HGetAll возвращает пустую map без выполнения операций
func (s *Store) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return make(map[string]string), nil
}

// HDel не выполняет операций
func (s *Store) HDel(ctx context.Context, key string, fields ...string) error {
	return nil
}

// LPush не выполняет операций
func (s *Store) LPush(ctx context.Context, key string, values ...interface{}) error {
	return nil
}

// RPush не выполняет операций
func (s *Store) RPush(ctx context.Context, key string, values ...interface{}) error {
	return nil
}

// LPop возвращает пустую строку без выполнения операций
func (s *Store) LPop(ctx context.Context, key string) (string, error) {
	return "", nil
}

// RPop возвращает пустую строку без выполнения операций
func (s *Store) RPop(ctx context.Context, key string) (string, error) {
	return "", nil
}

// LLen возвращает 0 без выполнения операций
func (s *Store) LLen(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

// SAdd не выполняет операций
func (s *Store) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return nil
}

// SMembers возвращает пустой slice без выполнения операций
func (s *Store) SMembers(ctx context.Context, key string) ([]string, error) {
	return []string{}, nil
}

// SIsMember возвращает false без выполнения операций
func (s *Store) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return false, nil
}

// SRem не выполняет операций
func (s *Store) SRem(ctx context.Context, key string, members ...interface{}) error {
	return nil
}

// Ping не выполняет операций
func (s *Store) Ping(ctx context.Context) error {
	return nil
}

// Close не выполняет операций
func (s *Store) Close() error {
	return nil
}
