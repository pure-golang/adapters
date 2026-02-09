# AGENTS.md

This document provides essential information for agents working in the **adapters** codebase - a Go library providing adapters and infrastructure for common services (PostgreSQL, RabbitMQ, gRPC, HTTP, logging, tracing, metrics).

## Project Overview

This is a Go library (module: `github.com/pure-golang/adapters`) that provides adapters and infrastructure for:
- **L0 (Monitoring)**: Logger, Tracing, Metrics
- **L1 (Service Drivers)**: PostgreSQL (sqlx and pgx implementations), RabbitMQ, gRPC, HTTP servers, CLI Executor

The project follows a **two-level directory structure**: first level = service/interface, second level = provider/implementation.
- Example: `queue/rabbitmq`, `db/pg/sqlx`, `db/pg/pgx`, `logger/stdjson`

## Essential Commands

### Testing
```bash
# Run all tests
go test .

# Skip integration tests (Docker-based)
go test -short .

# Run tests for a specific package
go test ./db/pg/sqlx
go test ./queue/rabbitmq
go test ./queue/kafka
```

### Build
```bash
# Build the module
go build ./...

# Verify dependencies
go mod tidy
go mod verify
```

### Dependencies
```bash
# Download dependencies
go mod download

# Update dependencies
go get -u ./...
```

### Docker for Integration Tests
Integration tests use `dockertest` to spin up PostgreSQL and RabbitMQ containers:
- PostgreSQL: `postgres:15`
- RabbitMQ: `rabbitmq:management-alpine`

**Docker must be running** for integration tests to pass.

## Code Organization

### Directory Structure
```
adapters/
├── db/
│   └── pg/
│       ├── sqlx/          # PostgreSQL adapter using sqlx library
│       └── pgx/           # PostgreSQL adapter using pgx v5 library
├── queue/
│   ├── rabbitmq/          # RabbitMQ adapter
│   │   └── encoders/      # Message encoders (JSON, text)
│   └── kafka/             # Kafka adapter
│       └── encoders/      # Message encoders (JSON, text)
├── grpc/
│   ├── middleware/        # gRPC middleware (logging, metrics, tracing, monitoring)
│   └── std/               # Standard gRPC server implementation
├── httpserver/
│   ├── middleware/        # HTTP middleware (monitoring, recovery)
│   └── std/               # Standard HTTP server implementation
├── executor/
│   └── cli/               # CLI executor for running external commands
├── logger/
│   ├── stdjson/           # Structured JSON logger for production
│   ├── devslog/           # Pretty-printed logger for development
│   └── noop/              # No-op logger for testing
├── tracing/
│   └── jaeger/            # OpenTelemetry Jaeger tracer
├── metrics/
│   └── prometheus.go      # Prometheus metrics integration
└── env/                   # Environment configuration utilities
```

### Package Structure
- Each adapter has its own `Config` struct
- Configs use `envconfig` tags: `` `envconfig:"VARIABLE_NAME"` ``
- Most providers implement both `Provider` and `io.Closer` interfaces
- Servers (HTTP/gRPC) implement `Provider` with `Start()` and `io.Closer` with `Close()`

## Code Conventions

### Error Handling
- Use `github.com/pkg/errors` for wrapping errors with context:
  ```go
  return errors.Wrap(err, "failed to connect to PostgreSQL")
  ```
- Use `errors.As()` for type assertion of wrapped errors
- Database adapters provide helper functions for constraint violations:
  ```go
  // PostgreSQL constraint checks
  IsUniqueViolation(err)
  IsForeignKeyViolation(err)
  IsCheckViolation(err)
  IsNotNullViolation(err)
  IsConstraintViolation(err)
  ```

### Context Usage
- All database operations accept `context.Context`
- Use `WithTimeout()` wrapper for query timeouts
- Context is passed through for tracing and cancellation

### Tracing
- Use OpenTelemetry for distributed tracing
- Tracers are initialized as package variables:
  ```go
  var tracer = otel.Tracer("github.com/pure-golang/adapters/db/pg/sqlx")
  ```
- Span naming pattern: `packageName.operation` (e.g., `sqlx.Get`, `sqlx.tx.Exec`)
- Common span attributes:
  - `db.system`: "postgresql"
  - `db.operation`: "Get", "Exec", "Select", etc.
  - `db.statement`: The SQL query
  - `db.transaction`: boolean for transaction spans

### Logging
- Use `log/slog` (standard library structured logging)
- Three logger implementations:
  - `ProviderStdJson`: Structured JSON for production
  - `ProviderDevSlog`: Pretty-printed for development
  - `ProviderNoop`: No-op for testing
- Log levels: INFO, ERROR, WARN, DEBUG
- Error logging automatically includes stack traces for errors implementing `StackTrace()`

### Environment Configuration
- Config structs use `envconfig` struct tags
- Load environment from `.env` file using `env.InitConfig()`:
  ```go
  var cfg Config
  if err := env.InitConfig(&cfg); err != nil {
      // handle error
  }
  ```
- Required fields: `` `envconfig:"VAR_NAME" required:"true"` ``
- Default values: `` `envconfig:"VAR_NAME" default:"value"` ``

### Database Patterns

#### Connection (sqlx)
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
```

#### Transactions (sqlx)
```go
err := db.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
    _, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", 100, 1)
    if err != nil {
        return err  // Auto rollback on error
    }

    _, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", 100, 2)
    return err  // Commit on nil, rollback on error
})
```

#### Transaction Isolation Levels
```go
opts := &sqlx.TxOptions{
    Isolation: sql.LevelRepeatableRead,
    ReadOnly:  false,
}
err := db.RunTx(ctx, opts, func(ctx context.Context, tx *sqlx.Tx) error {
    // operations
    return nil
})
```

#### Named Queries (sqlx)
```go
type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
    Age  int    `db:"age"`
}

user := User{Name: "John", Age: 30}
result, err := db.NamedExec(ctx,
    "INSERT INTO users (name, age) VALUES (:name, :age)",
    user)
```

### Interface Patterns
- **Provider**: Base interface for components that can start and be closed
- **Runner**: Interface for components that run indefinitely (in goroutines)
- **RunableProvider**: Combines Provider and Runner
- **Publisher/Subscriber**: Queue messaging pattern
- **Executor**: Interface for executing external CLI commands

```go
type Provider interface {
    Start() error
    io.Closer
}

type Runner interface {
    Run()
}

type RunableProvider interface {
    Provider
    Runner
}
```

## Testing

### Test Organization
- Unit tests: Standard `*_test.go` files
- Integration tests: Use `dockertest` for spinning up real services
- Test suites: Use `testify/suite` for complex test setups

### Integration Tests with Docker
Integration tests use `github.com/ory/dockertest` to:
1. Start PostgreSQL or RabbitMQ containers
2. Wait for services to be ready
3. Run tests against real services
4. Clean up containers on completion

**Key pattern**: Use `testing.Short()` flag to skip integration tests:
```go
func skipShort(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
}

func TestMain(m *testing.M) {
    if testing.Short() {
        fmt.Println("Skipping integration tests in short mode")
        os.Exit(0)
    }
    // Setup Docker containers...
}
```

### Test Assertions
- Use `github.com/stretchr/testify` for assertions
- `assert.NoError(t, err)`: Fail immediately on error
- `require.NoError(t, err)`: Continue test, fail at end
- `assert.ErrorIs(t, err, expectedErr)`: Check for specific error

### Common Test Patterns
```go
func TestConnection_Exec(t *testing.T) {
    skipShort(t)
    ctx := context.Background()

    // Setup: Create table
    _, err := testDB.Exec(ctx, `CREATE TABLE test_users (id SERIAL PRIMARY KEY, name TEXT)`)
    require.NoError(t, err)

    // Execute
    result, err := testDB.Exec(ctx, `INSERT INTO test_users (name) VALUES ($1)`, "John")
    require.NoError(t, err)

    // Verify
    affected, err := result.RowsAffected()
    require.NoError(t, err)
    require.Equal(t, int64(1), affected)
}
```

## Important Gotchas

### 1. Russian Language Comments
**All comments and documentation are in Russian.** This is intentional - do not translate to English unless specifically requested.

### 2. Query Timeouts
- Query timeouts are applied via context wrapping (`WithTimeout()`)
- Default timeout is configured in `Config.QueryTimeout`
- SQL queries include the timeout automatically through the wrapper

### 3. Transaction Rollback
- Transactions use defer-based rollback pattern
- `RunTx()` automatically rolls back on error or panic
- Manual `Rollback()` must check for `sql.ErrTxDone` (already committed/rolled back)

### 4. Error Context
- Always wrap errors with context using `errors.Wrap()`
- Use `errors.As()` for type assertion, not direct type casting
- Loggers automatically extract and log stack traces from `pkg/errors` errors

### 5. Database Drivers
- Two PostgreSQL adapters available:
  - `sqlx`: Uses `lib/pq` driver, traditional approach
  - `pgx`: Uses `jackc/pgx/v5`, more modern with connection pooling
- Choose based on project needs (pgx recommended for new projects)

### 6. Integration Test Requirements
- **Docker must be running** for integration tests
- Use `-short` flag to skip integration tests in CI/CD when Docker unavailable
- Integration tests automatically clean up containers with `AutoRemove: true`

### 7. Logger Context Propagation
- Loggers can be stored in and retrieved from context:
  ```go
  ctx = logger.NewContext(ctx, customLogger)
  log := logger.FromContext(ctx)
  ```
- If no logger in context, returns `slog.Default()`

### 8. Middleware Order
For gRPC/HTTP servers, middleware order matters:
1. Recovery (first) - catch panics
2. Tracing - create spans
3. Logging - log requests
4. Metrics - collect metrics
5. Application handler (last)

## Common Dependencies

### Core Libraries
- `github.com/jmoiron/sqlx`: Database operations for PostgreSQL
- `github.com/jackc/pgx/v5`: Modern PostgreSQL driver with connection pooling
- `github.com/rabbitmq/amqp091-go`: RabbitMQ client
- `google.golang.org/grpc`: gRPC framework
- `github.com/gorilla/mux`: HTTP router
- `os/exec`: Standard library for executing external commands (CLI executor)

### Observability
- `go.opentelemetry.io/otel/*`: OpenTelemetry tracing
- `go.opentelemetry.io/contrib`: OpenTelemetry integrations (grpc, runtime)
- `github.com/prometheus/client_golang`: Prometheus metrics
- `log/slog`: Structured logging (Go 1.21+)
- `github.com/golang-cz/devslog`: Pretty-printed development logger

### Testing
- `github.com/stretchr/testify`: Assertions and test suites
- `github.com/ory/dockertest/v3`: Docker integration testing
- `github.com/lib/pq`: PostgreSQL driver (for sqlx adapter)

### Configuration
- `github.com/joho/godotenv`: Load .env files
- `github.com/kelseyhightower/envconfig`: Environment variable parsing
- `github.com/pkg/errors`: Error wrapping and context

## Code Style Notes

- Comments and documentation are in Russian
- Struct fields use PascalCase
- Private fields use lowercase (not exported)
- Interface methods use descriptive names (e.g., `Connect`, `Publish`, `Listen`)
- Return errors as the last return value
- Use context as the first parameter for operations that may block or be cancelled
- Prefer `defer` for cleanup operations
- Always handle errors explicitly (never ignore with `_`)

## Configuration Example (.env)

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

# Logger
LOG_PROVIDER=std_json
LOG_LEVEL=info
```

## Adding New Adapters

When adding a new adapter:

1. Follow the directory structure: `adapter_type/adapter_name/`
2. Create a `Config` struct with `envconfig` tags
3. Implement `Provider` interface (or `RunableProvider` if it runs indefinitely)
4. Add OpenTelemetry tracing spans for operations
5. Add error wrapping with context
6. Create a `README.md` with usage examples (in Russian)
7. Add unit tests for core functionality
8. Add integration tests with `dockertest` (if external service)
9. Add to this AGENTS.md if new patterns introduced

## Troubleshooting

### Integration Tests Fail
- Check if Docker is running: `docker ps`
- Check port availability: `lsof -i :5432` (PostgreSQL), `lsof -i :5672` (RabbitMQ)
- Run with `-v` flag to see test output

### Context Deadline Exceeded
- Query timeout may be too short (check `Config.QueryTimeout`)
- Increase timeout in config or use longer-lived context

### Database Connection Errors
- Verify PostgreSQL is running and accessible
- Check connection string format (DSN)
- Verify credentials and database name
- Check firewall rules for port 5432

### RabbitMQ Connection Errors
- Verify RabbitMQ is running and accessible
- Check connection URL format: `amqp://user:pass@host:port/`
- Verify user permissions and queue existence
- Check firewall rules for port 5672

## Queue Patterns (Kafka)
```go
// Create Kafka dialer
cfg := kafka.Config{
    Brokers: []string{"localhost:9092"},
    GroupID: "my-consumer-group",
}
dialer := kafka.NewDialer(cfg, nil)

// Create publisher
pub := kafka.NewPublisher(dialer, kafka.PublisherConfig{
    Encoder:  encoders.JSON{},
    Balancer: &kafka.LeastBytes{},
})

// Publish message
msg := queue.Message{
    Topic: "my-topic",
    Body:  map[string]string{"key": "value"},
}
err := pub.Publish(ctx, msg)

// Create subscriber
sub := kafka.NewSubscriber(dialer, "my-topic", kafka.SubscriberConfig{
    Name:         "my-subscriber",
    PrefetchCount: 1,
    MaxTryNum:     3,
    Backoff:       5 * time.Second,
})

// Consume messages
go sub.Listen(func(ctx context.Context, msg queue.Delivery) (bool, error) {
    // Process message
    fmt.Println("Received:", string(msg.Body))
    return false, nil // false = success, true = retry
})
```

## Executor Patterns (CLI)
```go
// Create CLI executor
cfg := cli.Config{
    Command: "ffmpeg",
}
executor := cli.New(cfg)
defer executor.Close()

// Execute command
ctx := context.Background()
output, err := executor.Execute(ctx,
    "-i", "input.mp4",
    "-c:v", "libx264",
    "-c:a", "aac",
    "-y", "output.mp4",
)
if err != nil {
    log.Fatal(err)
}
```
