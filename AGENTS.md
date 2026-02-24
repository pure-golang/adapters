# AGENTS.md

This document defines **strategic context** for agents working in the **adapters** codebase. Tactical instructions are modularized into `.skills/`. Always consult the relevant skill before acting.

## Project Overview

Go library (`github.com/pure-golang/adapters`) providing adapters and infrastructure for:

- **L0 (Monitoring)**: Logger, Tracing, Metrics
- **L1 (Service Drivers)**: PostgreSQL (sqlx/pgx), RabbitMQ, Kafka, gRPC, HTTP server, CLI Executor, S3-compatible Storage (MinIO, Yandex Cloud, AWS S3)

**Two-level directory structure**: `{adapter_type}/{provider}`

Examples: `queue/rabbitmq`, `db/pg/sqlx`, `db/pg/pgx`, `logger/stdjson`

## Architecture Principles

### Package Structure
- Each adapter: `Config` struct (envconfig tags) + `New(cfg Config)` constructor + implements `Provider` or `RunableProvider`
- `doc.go` is the package contract — **read it first** when working in any package directory

### Constructor Rules
- **`New` returns a single value — never an error**: `func New(cfg Config) *Adapter`
- Constructors only store config and initialize lightweight fields (no I/O, no connections)
- Connection/initialization errors are deferred to `Start()` or `Connect()` methods

### Core Interfaces
```go
type Provider interface { Start() error; io.Closer }
type Runner interface { Run() }
type RunableProvider interface { Provider; Runner }
// Also: Publisher, Subscriber, Executor
```

### Layering
- L0 (monitoring) has no dependencies on L1
- L1 adapters depend on L0 for observability
- No circular imports between providers

## Global Conventions

- **All comments and documentation: in Russian.** Do not translate unless explicitly requested.
- **Log messages and error messages: English, lowercase first letter.**
- Always wrap errors with context: `errors.Wrap(err, "failed to ...")`
- Always use `errors.As()` for type assertion, never direct cast
- Context is the first parameter for blocking/cancellable operations
- Never ignore errors (`_ = err` is forbidden)
- Prefer `defer` for cleanup operations
- Struct fields: PascalCase. Private fields: lowercase (unexported).
- Return errors as the last return value

## Common Dependencies

### Core
- `github.com/jmoiron/sqlx` — PostgreSQL (sqlx adapter)
- `github.com/jackc/pgx/v5` — PostgreSQL (pgx adapter, recommended for new projects)
- `github.com/rabbitmq/amqp091-go` — RabbitMQ
- `google.golang.org/grpc` — gRPC

### Observability
- `go.opentelemetry.io/otel/*` — OpenTelemetry tracing
- `go.opentelemetry.io/contrib` — gRPC/runtime integrations
- `github.com/prometheus/client_golang` — Prometheus metrics
- `log/slog` — structured logging (Go 1.21+)
- `github.com/golang-cz/devslog` — pretty-printed dev logger

### Testing
- `github.com/stretchr/testify` — assertions and test suites
- `github.com/testcontainers/testcontainers-go` — Docker integration testing

### Configuration
- `github.com/joho/godotenv` — load `.env` files
- `github.com/kelseyhightower/envconfig` — env variable parsing
- `github.com/pkg/errors` — error wrapping with stack traces

### Storage
- `github.com/minio/minio-go/v7` — S3-compatible client

