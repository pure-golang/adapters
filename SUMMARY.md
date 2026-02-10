# Технический обзор проекта adapters

## Общая информация

**Название проекта:** adapters  
**Модуль:** `github.com/pure-golang/adapters`  
**Версия Go:** 1.24.12  
**Назначение:** Go-библиотека, предоставляющая адаптеры и инфраструктуру для общих сервисов

### Архитектурные принципы

Проект следует **двухуровневой структуре каталогов**:
- **Первый уровень** — поставляемая услуга (интерфейс)
- **Второй уровень** — поставщик услуги (реализация)

Примеры:
- `queue/rabbitmq` — адаптер очереди сообщений RabbitMQ
- `db/pg/sqlx` — адаптер PostgreSQL с использованием библиотеки sqlx
- `db/pg/pgx` — адаптер PostgreSQL с использованием библиотеки pgx v5
- `logger/stdjson` — JSON-логгер для production

### Уровни архитектуры

**L0 (Мониторинг):**
- Logger — структурированное логирование
- Tracing — распределённая трассировка (OpenTelemetry)
- Metrics — метрики (Prometheus)

**L1 (Драйверы сервисов):**
- PostgreSQL (sqlx и pgx реализации)
- RabbitMQ — брокер сообщений
- Kafka — брокер сообщений
- gRPC — RPC-фреймворк
- HTTP — HTTP-сервер
- Redis — key-value хранилище
- MinIO/S3 — объектное хранилище
- SMTP — отправка электронной почты

---

## Модульная структура

### 1. Logger (Логирование)

**Пакет:** `logger/`  
**Интерфейс:** `*slog.Logger` (стандартная библиотека Go 1.21+)

#### Реализации

| Реализация | Назначение | Пакет |
|------------|------------|-------|
| `ProviderStdJson` | Структурированный JSON для production | `logger/stdjson` |
| `ProviderDevSlog` | Pretty-printed для разработки | `logger/devslog` |
| `ProviderNoop` | No-op для тестирования | `logger/noop` |

#### Уровни логирования

- `INFO` — информационные сообщения
- `ERROR` — ошибки
- `WARN` — предупреждения
- `DEBUG` — отладочная информация

#### Возможности

- Хранение логгера в контексте: `logger.NewContext(ctx, log)`
- Извлечение из контекста: `logger.FromContext(ctx)`
- Автоматическое извлечение stack trace из ошибок `pkg/errors`
- Интеграция с OpenTelemetry error handler

#### Конфигурация

```go
type Config struct {
    Provider Provider `envconfig:"LOG_PROVIDER" default:"std_json"`
    Level    Level    `envconfig:"LOG_LEVEL" default:"info"`
}
```

---

### 2. Database (Базы данных)

#### 2.1 PostgreSQL (sqlx)

**Пакет:** `db/pg/sqlx/`  
**Драйвер:** `github.com/lib/pq`

##### Конфигурация

```go
type Config struct {
    Host            string        `envconfig:"POSTGRES_HOST" required:"true"`
    Port            int           `envconfig:"POSTGRES_PORT" default:"5432"`
    User            string        `envconfig:"POSTGRES_USER" required:"true"`
    Password        string        `envconfig:"POSTGRES_PASSWORD" required:"true"`
    Database        string        `envconfig:"POSTGRES_DB" required:"true"`
    SSLMode         string        `envconfig:"POSTGRES_SSLMODE" default:"disable"`
    ConnectTimeout  int           `envconfig:"POSTGRES_CONNECT_TIMEOUT" default:"5"`
    MaxOpenConns    int           `envconfig:"POSTGRES_MAX_OPEN_CONNS" default:"10"`
    MaxIdleConns    int           `envconfig:"POSTGRES_MAX_IDLE_CONNS" default:"5"`
    ConnMaxLifetime time.Duration `envconfig:"POSTGRES_CONN_MAX_LIFETIME" default:"30m"`
    ConnMaxIdleTime time.Duration `envconfig:"POSTGRES_CONN_MAX_IDLE_TIME" default:"10m"`
    QueryTimeout    time.Duration `envconfig:"POSTGRES_QUERY_TIMEOUT" default:"10s"`
}
```

##### Основные возможности

- **Connection:** `Connect(ctx, cfg)` — создание соединения
- **Транзакции:** `RunTx(ctx, opts, fn)` — выполнение транзакции с автоматическим rollback
- **Уровни изоляции:** поддержка всех стандартных уровней SQL
- **Named queries:** поддержка именованных запросов через sqlx
- **OpenTelemetry tracing:** автоматическое создание спанов для всех операций
- **Query timeouts:** применение таймаутов через контекст

##### Обработка ошибок

Вспомогательные функции для проверки ограничений базы данных:
- `IsUniqueViolation(err)` — нарушение уникальности
- `IsForeignKeyViolation(err)` — нарушение внешнего ключа
- `IsCheckViolation(err)` — нарушение CHECK-ограничения
- `IsNotNullViolation(err)` — нарушение NOT NULL
- `IsConstraintViolation(err)` — любое ограничение

#### 2.2 PostgreSQL (pgx)

**Пакет:** `db/pg/pgx/`  
**Драйвер:** `github.com/jackc/pgx/v5` (современный драйвер с connection pooling)

##### Конфигурация

```go
type Config struct {
    User            string `envconfig:"POSTGRES_USER" required:"true"`
    Password        string `envconfig:"POSTGRES_PASSWORD" required:"true"`
    Host            string `envconfig:"POSTGRES_HOST" required:"true"`
    Port            int    `envconfig:"POSTGRES_PORT" default:"5432"`
    Name            string `envconfig:"POSTGRES_DB_NAME" required:"true"`
    CertPath        string `envconfig:"POSTGRES_SSL_CERT_PATH"`
    MaxOpenConns    int32  `envconfig:"POSTGRES_MAX_OPEN_CONNECTIONS" default:"20"`
    MaxConnLifeTime int32  `envconfig:"POSTGRES_MAX_CONNECTIONS_LIFETIME" default:"5"`
    MaxConnIdleTime int32  `envconfig:"POSTGRES_MAX_CONNECTIONS_IDLE_TIME" default:"5"`
    TraceLogLevel   string `envconfig:"POSTGRES_TRACE_LOG_LEVEL" default:"error"`
}
```

##### Особенности

- **Connection pooling:** встроенный пул соединений pgxpool
- **OpenTelemetry integration:** через `github.com/exaring/otelpgx`
- **Multi-tracer support:** поддержка нескольких трейсеров одновременно
- **Health checks:** периодическая проверка соединений (20s)
- **SSL/TLS:** поддержка сертификатов для защищённых соединений

---

### 3. Queue (Очереди сообщений)

#### 3.1 RabbitMQ

**Пакет:** `queue/rabbitmq/`  
**Клиент:** `github.com/rabbitmq/amqp091-go`

##### Конфигурация

```go
type Config struct {
    URL   string `envconfig:"RABBITMQ_URL" required:"true"`
    Queue string `envconfig:"RABBITMQ_DEFAULT_QUEUE"`
}

type PublisherConfig struct {
    Exchange     string
    RoutingKey   string
    DeliveryMode DeliveryMode (Transient/Persistent)
    Encoder      queue.Encoder
    MessageTTL   time.Duration
}
```

##### Компоненты

**Publisher:**
- Публикация сообщений в exchange с routing key
- Поддержка TTL сообщений
- OpenTelemetry tracing с propagation
- Автоматическое переподключение при разрыве соединения

**Subscriber:**
- Потребление сообщений из очереди
- Retry политики: `ConstantRetryPolicy`, `IntervalRetryPolicy`
- Prefetch control для управления нагрузкой
- Ack/Nack механизмы

**Encoders:**
- `JSON` — кодирование в JSON
- `Text` — кодирование в текст

##### Retry политики

- **ConstantRetryPolicy:** фиксированная задержка между попытками
- **IntervalRetryPolicy:** увеличивающаяся задержка

#### 3.2 Kafka

**Пакет:** `queue/kafka/`  
**Клиент:** `github.com/segmentio/kafka-go`

##### Конфигурация

```go
type Config struct {
    Brokers []string `envconfig:"KAFKA_BROKERS" required:"true"`
    GroupID string   `envconfig:"KAFKA_GROUP_ID"`
}

type PublisherConfig struct {
    Balancer kafka.Balancer (LeastBytes, RoundRobin, etc.)
    Encoder  queue.Encoder
}
```

##### Компоненты

**Publisher:**
- Публикация сообщений в Kafka topics
- Балансировка между партициями
- OpenTelemetry tracing
- Кэширование writers для разных topics

**Subscriber:**
- Потребление сообщений из Kafka
- Group-based consumption
- Prefetch control
- Retry механизм с backoff

**Encoders:**
- `JSON` — кодирование в JSON
- `Text` — кодирование в текст

---

### 4. Key-Value Storage (Redis)

**Пакет:** `kv/`  
**Клиент:** `github.com/redis/go-redis/v9`

##### Конфигурация

```go
type Config struct {
    Provider         Provider `envconfig:"KV_PROVIDER" default:"noop"`
    RedisAddr        string   `envconfig:"REDIS_ADDR" default:"localhost:6379"`
    RedisPassword    string   `envconfig:"REDIS_PASSWORD"`
    RedisDB          int      `envconfig:"REDIS_DB" default:"0"`
    RedisMaxRetries  int      `envconfig:"REDIS_MAX_RETRIES" default:"3"`
    RedisDialTimeout time.Duration `envconfig:"REDIS_DIAL_TIMEOUT" default:"5s"`
    RedisReadTimeout time.Duration `envconfig:"REDIS_READ_TIMEOUT" default:"3s"`
    RedisWriteTimeout time.Duration `envconfig:"REDIS_WRITE_TIMEOUT" default:"3s"`
    RedisPoolSize    int      `envconfig:"REDIS_POOL_SIZE" default:"10"`
}
```

##### Интерфейс Store

```go
type Store interface {
    // Базовые операции
    Get(ctx, key) (string, error)
    Set(ctx, key, value, expiration) error
    Delete(ctx, keys...) error
    Exists(ctx, keys...) (int64, error)
    
    // Операции с счётчиками
    Incr(ctx, key) (int64, error)
    Decr(ctx, key) (int64, error)
    
    // TTL операции
    Expire(ctx, key, expiration) error
    TTL(ctx, key) (time.Duration, error)
    
    // Hash операции
    HGet(ctx, key, field) (string, error)
    HSet(ctx, key, field, value) error
    HGetAll(ctx, key) (map[string]string, error)
    HDel(ctx, key, fields...) error
    
    // List операции
    LPush(ctx, key, values...) error
    RPush(ctx, key, values...) error
    LPop(ctx, key) (string, error)
    RPop(ctx, key) (string, error)
    LLen(ctx, key) (int64, error)
    
    // Set операции
    SAdd(ctx, key, members...) error
    SMembers(ctx, key) ([]string, error)
    SIsMember(ctx, key, member) (bool, error)
    SRem(ctx, key, members...) error
    
    // Подключение
    Ping(ctx) error
    Close() error
}
```

##### Возможности

- OpenTelemetry tracing для всех операций
- Connection pooling
- Retry механизм
- Поддержка всех основных типов данных Redis

---

### 5. Object Storage (S3/MinIO)

**Пакет:** `storage/`  
**Клиент:** `github.com/minio/minio-go/v7`

##### Конфигурация

```go
type Config struct {
    Endpoint           string `envconfig:"S3_ENDPOINT"`
    AccessKey          string `envconfig:"S3_ACCESS_KEY" required:"true"`
    SecretKey          string `envconfig:"S3_SECRET_KEY" required:"true"`
    Region             string `envconfig:"S3_REGION" default:"us-east-1"`
    DefaultBucket      string `envconfig:"S3_BUCKET"`
    Secure             bool   `envconfig:"S3_SECURE" default:"true"`
    Timeout            int    `envconfig:"S3_TIMEOUT" default:"30"`
    InsecureSkipVerify bool   `envconfig:"S3_INSECURE_SKIP_VERIFY" default:"false"`
}
```

##### Интерфейс Storage

```go
type Storage interface {
    // Базовые операции
    Put(ctx, bucket, key, reader, opts) error
    Get(ctx, bucket, key) (io.ReadCloser, *ObjectInfo, error)
    Delete(ctx, bucket, key) error
    Exists(ctx, bucket, key) (bool, error)
    List(ctx, bucket, opts) (*ListResult, error)
    
    // Presigned URLs
    GetPresignedURL(ctx, bucket, key, opts) (string, error)
    
    // Multipart upload
    CreateMultipartUpload(ctx, bucket, key, opts) (*MultipartUpload, error)
    UploadPart(ctx, bucket, key, uploadID, partNumber, reader) (*UploadedPart, error)
    CompleteMultipartUpload(ctx, bucket, key, uploadID, opts) (*ObjectInfo, error)
    AbortMultipartUpload(ctx, bucket, key, uploadID) error
    ListMultipartUploads(ctx, bucket) ([]MultipartUpload, error)
    
    io.Closer
}
```

##### Поддерживаемые провайдеры

- MinIO (локальный S3-совместимый storage)
- Yandex Cloud Storage (default endpoint: `storage.yandexcloud.net`)
- AWS S3
- Любые S3-совместимые хранилища

##### Возможности

- OpenTelemetry tracing
- Multipart upload для больших файлов
- Presigned URLs для прямого доступа
- Metadata поддержка
- TLS/SSL с возможностью skip verify

---

### 6. gRPC Server

**Пакет:** `grpc/`  
**Фреймворк:** `google.golang.org/grpc`

#### 6.1 Стандартный сервер

**Пакет:** `grpc/std/`

##### Конфигурация

```go
type Config struct {
    Host          string `envconfig:"GRPC_HOST"`
    Port          int    `envconfig:"GRPC_PORT" required:"true"`
    TLSCertPath   string `envconfig:"GRPC_TLS_CERT_PATH"`
    TLSKeyPath    string `envconfig:"GRPC_TLS_KEY_PATH"`
    EnableReflect bool   `envconfig:"GRPC_ENABLE_REFLECTION" default:"true"`
}
```

##### Возможности

- TLS/SSL поддержка
- gRPC Reflection API (для отладки)
- Graceful shutdown (15s timeout)
- Keepalive параметры
- Custom interceptors через `ServerOption`

#### 6.2 Middleware

**Пакет:** `grpc/middleware/`

##### Доступные интерцепторы

| Интерцептор | Назначение |
|-------------|------------|
| `Logging` | Структурированное логирование запросов |
| `Metrics` | Сбор Prometheus метрик |
| `Tracing` | OpenTelemetry трассировка |
| `Monitoring` | Комбинированный мониторинг |
| `Recovery` | Восстановление после паник |

##### Метрики

- `grpc.server.requests_total` — счётчик запросов
- `grpc.server.duration_ms` — гистограмма длительности
- `grpc.server.request_size_bytes` — размер запросов
- `grpc.server.response_size_bytes` — размер ответов

##### Настройка

```go
monitoringOptions := middleware.DefaultMonitoringOptions(logger)
unaryInterceptors, streamInterceptors, serverOptions := middleware.SetupMonitoring(
    context.Background(), 
    monitoringOptions,
)
```

#### 6.3 Обработка ошибок

**Пакет:** `grpc/errors/`

Автоматическое преобразование ошибок Go в gRPC статусы:
- `FromError(err)` — автоопределение типа ошибки
- `NewError(codes, message)` — создание ошибки с кодом

---

### 7. HTTP Server

**Пакет:** `httpserver/`

#### 7.1 Стандартный сервер

**Пакет:** `httpserver/std/`

##### Конфигурация

```go
type Config struct {
    Host        string `envconfig:"WEBSERVER_HOST"`
    Port        int    `envconfig:"WEBSERVER_PORT" required:"true"`
    TLSCertPath string `envconfig:"WEBSERVER_TLS_CERT_PATH"`
    TLSKeyPath  string `envconfig:"WEBSERVER_TLS_KEY_PATH"`
    ReadTimeout string `envconfig:"WEBSERVER_READ_TIMEOUT" default:"30"`
}
```

##### Возможности

- TLS/SSL поддержка
- Graceful shutdown (15s timeout)
- Protection от Slowloris атак (ReadHeaderTimeout: 10s)
- Custom error logging

#### 7.2 Middleware

**Пакет:** `httpserver/middleware/`

| Интерцептор | Назначение |
|-------------|------------|
| `Monitoring` | Мониторинг запросов |
| `Recovery` | Восстановление после паник |

---

### 8. Mail (SMTP)

**Пакет:** `mail/`

#### 8.1 SMTP Sender

**Пакет:** `mail/smtp/`

##### Конфигурация

```go
type Config struct {
    Host     string `envconfig:"SMTP_HOST" required:"true"`
    Port     int    `envconfig:"SMTP_PORT" required:"true"`
    Username string `envconfig:"SMTP_USERNAME"`
    Password string `envconfig:"SMTP_PASSWORD"`
    From     string `envconfig:"SMTP_FROM"`
    TLS      bool   `envconfig:"SMTP_TLS" default:"true"`
    Insecure bool   `envconfig:"SMTP_INSECURE" default:"false"`
}
```

##### Интерфейс Email

```go
type Email struct {
    // Envelope
    From    Address
    To      []Address
    Cc      []Address
    Bcc     []Address
    Subject string
    
    // Headers
    Headers map[string]string
    
    // Body
    Body string // Plain text
    HTML string // HTML (опционально)
}
```

##### Возможности

- TLS/STARTTLS поддержка
- Multipart messages (plain text + HTML)
- Custom headers
- OpenTelemetry tracing
- Thread-safe операции

---

### 9. Metrics (Prometheus)

**Пакет:** `metrics/`

##### Конфигурация

```go
type Config struct {
    Host                  string `envconfig:"METRICS_HOST" required:"true"`
    Port                  int    `envconfig:"METRICS_PORT" required:"true"`
    HttpServerReadTimeout  int    `envconfig:"METRICS_READ_TIMEOUT" default:"30"`
}
```

##### Возможности

- HTTP endpoint `/metrics` для Prometheus
- Runtime metrics (через `go.opentelemetry.io/contrib/instrumentation/runtime`)
- Custom metrics поддержка
- Graceful shutdown

---

### 10. Tracing (OpenTelemetry)

**Пакет:** `tracing/`

#### 10.1 Jaeger Provider

**Пакет:** `tracing/jaeger/`

##### Конфигурация

```go
type Config struct {
    EndPoint    string `envconfig:"TRACING_ENDPOINT" required:"true"`
    ServiceName string `envconfig:"SERVICE_NAME" required:"true"`
    AppVersion  string `envconfig:"APP_VERSION" required:"true"`
}
```

##### Возможности

- OTLP exporter (HTTP)
- Resource attributes (service name, version)
- Batch sampling (AlwaysSample)
- Graceful shutdown с ForceFlush

---

## Общие паттерны и конвенции

### Интерфейсы

#### Provider
Базовый интерфейс для компонентов, которые могут запускаться и закрываться:

```go
type Provider interface {
    Start() error
    io.Closer
}
```

#### Runner
Интерфейс для компонентов, которые работают indefinitely (в горутинах):

```go
type Runner interface {
    Run()
}
```

#### RunableProvider
Комбинирует Provider и Runner:

```go
type RunableProvider interface {
    Provider
    Runner
}
```

### Обработка ошибок

- Использование `github.com/pkg/errors` для заворачивания ошибок с контекстом
- `errors.Wrap(err, "context")` — добавление контекста к ошибке
- `errors.As(err, &target)` — тип assertion для wrapped errors
- Автоматическое извлечение stack trace из ошибок в логгере

### Использование Context

- Все операции, которые могут блокироваться, принимают `context.Context` первым аргументом
- Query timeouts применяются через контекст
- Tracing propagation через контекст
- Logger propagation через контекст

### OpenTelemetry Tracing

- Инициализация трейсера как пакетной переменной
- Именование спанов: `packageName.operation` (например, `sqlx.Get`, `kafka.Publish`)
- Стандартные атрибуты для каждого типа операции
- Propagation trace context через заголовки сообщений

### Конфигурация

- Использование `github.com/kelseyhightower/envconfig` для парсинга переменных окружения
- Теги struct: `` `envconfig:"VAR_NAME" required:"true"` ``
- Default значения: `` `envconfig:"VAR_NAME" default:"value"` ``
- Загрузка из `.env` файла через `env.InitConfig(&cfg)`

### Тестирование

#### Unit тесты
- Стандартные `*_test.go` файлы
- Использование `github.com/stretchr/testify` для assertions

#### Интеграционные тесты
- Использование `github.com/testcontainers/testcontainers-go` для Docker контейнеров
- Поддержка флага `-short` для пропуска Docker-тестов
- Автоматическая очистка контейнеров

#### Покрытие тестами

Для адаптеров баз данных и очередей сообщений:
- Тест подключения и установления соединения
- Тест базовых операций (CRUD)
- Тест обработки ошибок
- Тест транзакций (commit/rollback)
- Тест уровней изоляции транзакций
- Тест обработки таймаутов и отмены контекста
- Тест конкурентных операций

---

## Зависимости проекта

### Основные библиотеки

| Библиотека | Версия | Назначение |
|------------|--------|-----------|
| `github.com/jmoiron/sqlx` | v1.4.0 | Database operations |
| `github.com/jackc/pgx/v5` | v5.7.6 | Modern PostgreSQL driver |
| `github.com/rabbitmq/amqp091-go` | v1.10.0 | RabbitMQ client |
| `github.com/segmentio/kafka-go` | v0.4.49 | Kafka client |
| `github.com/redis/go-redis/v9` | v9.17.2 | Redis client |
| `github.com/minio/minio-go/v7` | v7.0.97 | S3-compatible storage |
| `google.golang.org/grpc` | v1.67.1 | gRPC framework |

### Observability

| Библиотека | Версия | Назначение |
|------------|--------|-----------|
| `go.opentelemetry.io/otel` | v1.35.0 | OpenTelemetry tracing |
| `go.opentelemetry.io/contrib` | v0.49.0 | OpenTelemetry integrations |
| `github.com/prometheus/client_golang` | v1.20.5 | Prometheus metrics |
| `log/slog` | stdlib | Structured logging (Go 1.21+) |
| `github.com/golang-cz/devslog` | v0.0.11 | Pretty-printed logger |

### Тестирование

| Библиотека | Версия | Назначение |
|------------|--------|-----------|
| `github.com/stretchr/testify` | v1.11.1 | Assertions and test suites |
| `github.com/testcontainers/testcontainers-go` | v0.40.0 | Docker integration testing |
| `github.com/ory/dockertest/v3` | v3.10.0 | Docker integration testing (legacy) |

### Конфигурация

| Библиотека | Версия | Назначение |
|------------|--------|-----------|
| `github.com/joho/godotenv` | v1.5.1 | Load .env files |
| `github.com/kelseyhightower/envconfig` | v1.4.0 | Environment variable parsing |
| `github.com/pkg/errors` | v0.9.1 | Error wrapping and context |

---

## Команды для работы с проектом

### Тестирование

```bash
# Запуск всех тестов
go test .

# Пропуск интеграционных тестов (Docker-based)
go test -short .

# Запуск тестов для конкретного пакета
go test ./db/pg/sqlx
go test ./queue/rabbitmq
go test ./queue/kafka
```

### Сборка

```bash
# Сборка модуля
go build ./...

# Проверка зависимостей
go mod tidy
go mod verify
```

### Зависимости

```bash
# Загрузка зависимостей
go mod download

# Обновление зависимостей
go get -u ./...
```

### Docker для интеграционных тестов

Интеграционные тесты используют `testcontainers` для запуска контейнеров:
- PostgreSQL: `postgres:15`
- RabbitMQ: `rabbitmq:management-alpine`
- Kafka: `confluentinc/cp-kafka`
- MinIO: `minio/minio`

**Docker должен быть запущен** для прохождения интеграционных тестов.

---

## Особенности и важные моменты

### 1. Русскоязычные комментарии

**Все комментарии и документация на русском языке.** Это намеренное решение — не переводить на английский, если не требуется явно.

### 2. Query Timeouts

- Query timeouts применяются через контекст (`WithTimeout()`)
- Default timeout настраивается в `Config.QueryTimeout`
- SQL запросы включают таймаут автоматически через wrapper

### 3. Transaction Rollback

- Транзакции используют defer-based rollback pattern
- `RunTx()` автоматически откатывает при ошибке или panic
- Ручной `Rollback()` должен проверять `sql.ErrTxDone` (уже committed/rolled back)

### 4. Error Context

- Всегда заворачивайте ошибки с контекстом через `errors.Wrap()`
- Используйте `errors.As()` для type assertion, а не direct type casting
- Логгеры автоматически извлекают и логируют stack traces из `pkg/errors` ошибок

### 5. Database Drivers

Два PostgreSQL адаптера доступны:
- `sqlx`: Использует `lib/pq` driver, традиционный подход
- `pgx`: Использует `jackc/pgx/v5`, более современный с connection pooling

Выбор зависит от потребностей проекта (pgx рекомендуется для новых проектов).

### 6. Требования к интеграционным тестам

- **Docker должен быть запущен** для интеграционных тестов
- Используйте флаг `-short` для пропуска интеграционных тестов в CI/CD, когда Docker недоступен
- Интеграционные тесты автоматически очищают контейнеры

### 7. Logger Context Propagation

- Логгеры можно хранить и извлекать из контекста:
  ```go
  ctx = logger.NewContext(ctx, customLogger)
  log = logger.FromContext(ctx)
  ```
- Если нет логгера в контексте, возвращается `slog.Default()`

### 8. Middleware Order

Для gRPC/HTTP серверов порядок middleware важен:
1. Recovery (first) — catch panics
2. Tracing — create spans
3. Logging — log requests
4. Metrics — collect metrics
5. Application handler (last)

---

## Definition of Done (DOD)

При добавлении новых адаптеров должны быть выполнены следующие критерии:

### Требования к коду

- Код соответствует конвенциям проекта (см. `AGENTS.md`)
- Структура `Config` с тегами `envconfig`
- Реализация интерфейса `Provider` (или `RunableProvider`)
- Обработка ошибок с заворачиванием в контекст через `errors.Wrap()`
- Использование `context.Context` первым аргументом
- Реализация `defer` для корректной очистки ресурсов

### Требования к документации

- Наличие `README.md` в директории адаптера
- Все комментарии и документация на русском языке
- Описание всех параметров конфигурации
- Примеры базовых операций

### Требования к тестированию

- **Unit-тесты** для основной бизнес-логики
- **Интеграционные тесты** с использованием `github.com/testcontainers/testcontainers-go`
- Поддержка флага `-short` для пропуска Docker-тестов
- Автоматический запуск и остановка контейнеров
- Ожидание готовности сервиса перед выполнением тестов

### Требования к наблюдаемости

- OpenTelemetry tracing для всех операций
- Интеграция со структурированным логированием (`log/slog`)
- Метрики (если применимо)

### Дополнительные требования

- `go build ./...` выполняется без ошибок
- `go test ./...` выполняется без ошибок
- `go vet ./...` не выдаёт предупреждений
- `go mod tidy` не изменяет `go.mod` и `go.sum`
- Отсутствие `TODO` в коде адаптера
- Добавление описания нового адаптера в `AGENTS.md` при введении новых паттернов

---

## Примеры использования

### PostgreSQL (sqlx)

```go
cfg := sqlx.Config{
    Host:           "localhost",
    Port:           5432,
    User:           "postgres",
    Password:       "secret",
    Database:       "mydb",
    SSLMode:        "disable",
    ConnectTimeout: 5,
    QueryTimeout:   10 * time.Second,
}

db, err := sqlx.Connect(context.Background(), cfg)
defer db.Close()

// Транзакция
err = db.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
    _, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", 100, 1)
    if err != nil {
        return err  // Auto rollback
    }
    return nil  // Commit
})
```

### RabbitMQ

```go
cfg := rabbitmq.Config{
    URL:   "amqp://guest:guest@localhost:5672/",
    Queue: "default_queue",
}

dialer := rabbitmq.NewDialer(cfg, nil)
defer dialer.Close()

pub := rabbitmq.NewPublisher(dialer, rabbitmq.PublisherConfig{
    Exchange:     "my-exchange",
    RoutingKey:   "my-key",
    DeliveryMode: rabbitmq.Persistent,
    Encoder:      encoders.JSON{},
})

msg := queue.Message{
    Topic: "my-topic",
    Body:  map[string]string{"key": "value"},
}
err := pub.Publish(ctx, msg)
```

### Redis

```go
cfg := kv.Config{
    Provider:      kv.ProviderRedis,
    RedisAddr:    "localhost:6379",
    RedisPassword: "",
    RedisDB:      0,
}

store, err := kv.NewDefault()
defer store.Close()

// Базовые операции
err := store.Set(ctx, "key", "value", 10*time.Minute)
val, err := store.Get(ctx, "key")

// Hash операции
err := store.HSet(ctx, "user:1", "name", "John")
name, err := store.HGet(ctx, "user:1", "name")
```

### gRPC Server

```go
config := std.Config{
    Host:          "",
    Port:          50051,
    EnableReflect: true,
}

server := std.NewDefault(config, func(grpcServer *grpc.Server) {
    pb.RegisterMyServiceServer(grpcServer, &myServiceImpl{})
})

server.Run()
defer server.Close()
```

---

## Заключение

Проект `adapters` представляет собой комплексную библиотеку адаптеров для Go, предоставляющую унифицированный интерфейс для работы с различными сервисами и инфраструктурными компонентами. Основные преимущества:

1. **Единообразие:** Все адаптеры следуют общим паттернам и конвенциям
2. **Наблюдаемость:** Встроенная поддержка OpenTelemetry, Prometheus и структурированного логирования
3. **Надёжность:** Graceful shutdown, retry механизмы, connection pooling
4. **Тестируемость:** Комплексное покрытие тестами, включая интеграционные с Docker
5. **Гибкость:** Поддержка различных реализаций (sqlx/pgx, RabbitMQ/Kafka, etc.)
6. **Современность:** Использование последних версий Go и библиотек

Проект идеально подходит для построения микросервисной архитектуры с едиными стандартами мониторинга, логирования и трассировки.
