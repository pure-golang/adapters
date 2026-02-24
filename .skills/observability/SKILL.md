---
name: "observability"
description: "Трейсинг (OpenTelemetry), логирование (slog), метрики (Prometheus), порядок middleware"
---
# Observability

## Tracing (OpenTelemetry)

Initialize tracer as package variable:
```go
var tracer = otel.Tracer("github.com/pure-golang/adapters/db/pg/sqlx")
```

Span naming pattern: `packageName.operation`
```go
// Examples:
// sqlx.Get, sqlx.tx.Exec
// S3.GetFileHeader, S3.Put, S3.Get
// rabbitmq.Publish, rabbitmq.Subscribe
```

Standard span attributes for database operations:
```go
span.SetAttributes(
    attribute.String("db.system", "postgresql"),
    attribute.String("db.operation", "Get"),       // Get, Exec, Select, etc.
    attribute.String("db.statement", sqlQuery),
    attribute.Bool("db.transaction", isInTx),
)
```

## Logging (slog)

Three logger implementations:
| Package | Use case |
|---------|----------|
| `logger/stdjson` | Production — structured JSON |
| `logger/devslog` | Development — pretty-printed |
| `logger/noop` | Tests — discard all output |

```go
// Initialize default logger (main.go)
logger.InitDefault(stdjson.New(lcfg))

// Logger from context
ctx = logger.NewContext(ctx, customLogger)
log := logger.FromContext(ctx)  // returns slog.Default() if not set

// Log levels
slog.Info("message", "key", value)
slog.Error("message", "err", err)
slog.Warn("message", "key", value)
slog.Debug("message", "key", value)
```

Error logging automatically includes stack traces for errors implementing `StackTrace()` (from `github.com/pkg/errors`).

## Metrics (Prometheus)

Prometheus metrics are integrated via `metrics/prometheus.go`.
Standard Go runtime metrics are exposed automatically when the adapter is initialized.

## Middleware Order (gRPC / HTTP)

Apply middleware in this order (outermost first):
1. **Recovery** — catch panics before anything else
2. **Tracing** — create spans
3. **Logging** — log requests (has span context available)
4. **Metrics** — collect metrics
5. **Application handler** (innermost)

```go
// gRPC example
grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        recovery.UnaryServerInterceptor(),
        tracing.UnaryServerInterceptor(),
        logging.UnaryServerInterceptor(),
        metrics.UnaryServerInterceptor(),
    ),
)
```
