---
name: "integration_testing"
description: "Паттерн интеграционных тестов с testcontainers-go: setup, teardown, skip-флаг, context timeout"
modes: [Code, Debug]
---
# Skill: Integration Testing

## Tactical Instructions

### testcontainers-go Suite Pattern
```go
type MySuite struct {
    suite.Suite
    container testcontainers.Container
    dsn       string
}

func TestMySuite(t *testing.T) {
    if testing.Short() {
        t.Skip("integration test")
    }
    suite.Run(t, new(MySuite))
}

func (s *MySuite) SetupSuite() {
    ctx := context.Background()
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "postgres:15-alpine",
            ExposedPorts: []string{"5432/tcp"},
            Env: map[string]string{
                "POSTGRES_USER":     "postgres",
                "POSTGRES_PASSWORD": "secret",
                "POSTGRES_DB":       "testdb",
            },
            WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
            AutoRemove: true,
        },
        Started: true,
    })
    s.Require().NoError(err)
    s.container = container

    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "5432")
    s.dsn = fmt.Sprintf("postgres://postgres:secret@%s:%s/testdb?sslmode=disable", host, port.Port())
}

func (s *MySuite) TearDownSuite() {
    if s.container != nil {
        s.Require().NoError(s.container.Terminate(context.Background()))
    }
}
```

### RabbitMQ Container
```go
ContainerRequest: testcontainers.ContainerRequest{
    Image:        "rabbitmq:management-alpine",
    ExposedPorts: []string{"5672/tcp"},
    WaitingFor:   wait.ForListeningPort("5672/tcp"),
    AutoRemove:   true,
},
```

### Skip Marker (Required for all integration tests)
```go
func TestMyIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("integration test")
    }
    // ...
}
```

### Context Timeout in Integration Tests
Use `context.WithTimeout` instead of plain `context.Background()`:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

### Readiness Strategies
```go
// Wait for log message
wait.ForLog("ready to accept connections").WithOccurrence(2)

// Wait for port
wait.ForListeningPort("5432/tcp")
```
