# Redis Adapter

Адаптер для работы с Redis как key-value хранилищем.

## Установка

```bash
go get github.com/pure-golang/adapters/kv/redis
```

## Конфигурация

Конфигурация выполняется через переменные окружения с использованием `envconfig`:

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `KV_PROVIDER` | Тип провайдера (должен быть `redis`) | `noop` |
| `REDIS_ADDR` | Адрес Redis сервера (хост:порт) | `localhost:6379` |
| `REDIS_PASSWORD` | Пароль для подключения | - |
| `REDIS_DB` | Номер базы данных | `0` |
| `REDIS_MAX_RETRIES` | Максимальное количество попыток повтора | `3` |
| `REDIS_DIAL_TIMEOUT` | Таймаут установки соединения | `5s` |
| `REDIS_READ_TIMEOUT` | Таймаут чтения | `3s` |
| `REDIS_WRITE_TIMEOUT` | Таймаут записи | `3s` |
| `REDIS_POOL_SIZE` | Размер пула соединений | `10` |

## Использование

### Создание клиента

```go
import (
    "context"
    "github.com/pure-golang/adapters/kv"
    "github.com/pure-golang/adapters/kv/redis"
)

// Через общий интерфейс kv (конфигурация из переменных окружения)
store, err := kv.NewDefault()
// или
store, err := kv.InitDefault()

// Или напрямую с явной конфигурацией
client, err := redis.Connect(context.Background(), redis.Config{
    Addr: "localhost:6379",
    DB:   0,
})
```

### Базовые операции

```go
ctx := context.Background()

// Установка значения
err := client.Set(ctx, "mykey", "myvalue", 0)

// Получение значения
value, err := client.Get(ctx, "mykey")

// Удаление ключей
err := client.Delete(ctx, "key1", "key2")

// Проверка существования
count, err := client.Exists(ctx, "key1", "key2")
```

### Операции с TTL

```go
// Установка с TTL (1 час)
err := client.Set(ctx, "session:123", "data", time.Hour)

// Установка TTL для существующего ключа
err := client.Expire(ctx, "session:123", 30*time.Minute)

// Получение оставшегося времени жизни
ttl, err := client.TTL(ctx, "session:123")
```

### Операции с счётчиками

```go
// Инкремент
counter, err := client.Incr(ctx, "counter")

// Декремент
counter, err := client.Decr(ctx, "counter")
```

### Hash операции

```go
// Установка поля в хеше
err := client.HSet(ctx, "user:123", "name", "John")

// Получение поля из хеша
name, err := client.HGet(ctx, "user:123", "name")

// Получение всех полей хеша
fields, err := client.HGetAll(ctx, "user:123")

// Удаление полей из хеша
err := client.HDel(ctx, "user:123", "name", "email")
```

### List операции

```go
// Добавление в начало списка
err := client.LPush(ctx, "queue", "task1", "task2")

// Добавление в конец списка
err := client.RPush(ctx, "queue", "task3")

// Получение из начала
task, err := client.LPop(ctx, "queue")

// Получение из конца
task, err := client.RPop(ctx, "queue")

// Длина списка
length, err := client.LLen(ctx, "queue")
```

### Set операции

```go
// Добавление членов в множество
err := client.SAdd(ctx, "tags", "golang", "redis")

// Получение всех членов
tags, err := client.SMembers(ctx, "tags")

// Проверка членства
isMember, err := client.SIsMember(ctx, "tags", "golang")

// Удаление членов
err := client.SRem(ctx, "tags", "golang")
```

### Закрытие подключения

```go
defer client.Close()
```

## Обработка ошибок

Адаптер возвращает ошибку `redis.ErrKeyNotFound` при попытке получить несуществующий ключ:

```go
value, err := client.Get(ctx, "nonexistent")
if err == redis.ErrKeyNotFound {
    // Ключ не найден
} else if err != nil {
    // Другая ошибка
}
```

## Интеграционные тесты

Для запуска интеграционных тестов требуется Docker:

```bash
# Все тесты
go test ./kv/redis/... -v

# С пропуском интеграционных тестов
go test ./kv/redis/... -short
```

## Observability

Адаптер поддерживает OpenTelemetry tracing для всех операций и структурированное логирование через `log/slog`.
