---
name: "env_config"
description: "Паттерн конфигурации через env: envconfig-теги, InitConfig, .env файл"
---
# Environment Configuration

## Config Struct Pattern
```go
type Config struct {
    Host           string        `envconfig:"SERVICE_HOST"    default:"localhost"`
    Port           int           `envconfig:"SERVICE_PORT"    default:"5432"`
    Password       string        `envconfig:"SERVICE_PASSWORD" required:"true"`
    Timeout        time.Duration `envconfig:"SERVICE_TIMEOUT"  default:"10s"`
    MaxConnections int           `envconfig:"SERVICE_MAX_CONNS" default:"10"`
}
```

Tag reference:
- `envconfig:"VAR_NAME"` — env variable name
- `required:"true"` — fail if not set
- `default:"value"` — fallback value

## Loading Config
```go
var cfg Config
if err := env.InitConfig(&cfg); err != nil {
    // handle error
}
```

`env.InitConfig` loads `.env` file from the project root (via `godotenv`) and then parses env variables via `envconfig`.

## .env File Example
```bash
# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=secret
POSTGRES_DB=mydb
POSTGRES_SSLMODE=disable
POSTGRES_CONNECT_TIMEOUT=5
POSTGRES_MAX_OPEN_CONNS=10
POSTGRES_MAX_IDLE_CONNS=5
POSTGRES_CONN_MAX_LIFETIME=30m
POSTGRES_CONN_MAX_IDLE_TIME=10m
POSTGRES_QUERY_TIMEOUT=10s

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_DEFAULT_QUEUE=default_queue

# S3 Storage (MinIO/Yandex Cloud/AWS S3)
S3_ENDPOINT=localhost:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_REGION=us-east-1
S3_BUCKET=my-bucket
S3_SECURE=false
S3_TIMEOUT=30

# Logger
LOG_PROVIDER=std_json
LOG_LEVEL=info
```
