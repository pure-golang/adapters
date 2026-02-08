package redis

import (
	"github.com/pkg/errors"
)

// ErrKeyNotFound возвращается когда ключ не найден в Redis
var ErrKeyNotFound = errors.New("key not found")

// ErrTypeMismatch возвращается когда тип значения не соответствует ожидаемому
var ErrTypeMismatch = errors.New("type mismatch")

// Nil является обёрткой для redis.Nil
type Nil struct{}

// Error реализует интерфейс error
func (Nil) Error() string {
	return "redis: nil"
}

// IsNil проверяет, является ли ошибка redis.Nil
func IsNil(err error) bool {
	return err == Nil{} || err == nil
}
