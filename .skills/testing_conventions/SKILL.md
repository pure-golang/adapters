---
name: "testing_conventions"
description: "Соглашения по тестам: маркеры unit/integration, AAA-паттерн, assertions, t.Parallel, t.Cleanup"
---
# Testing Conventions

## Test Type Markers

**Unit tests** — use `t.Parallel()` at the beginning:
```go
func TestSomething(t *testing.T) {
    t.Parallel()
    // no external services required
}
```

**Integration tests** — use `testing.Short()` check:
```go
func TestSomethingIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("integration test")
    }
    // requires Docker/DB/RabbitMQ
}
```

## AAA Pattern (Arrange, Act, Assert)
```go
func TestConnection_Exec(t *testing.T) {
    if testing.Short() {
        t.Skip("integration test")
    }
    ctx := context.Background()

    // Arrange: Create table
    _, err := testDB.Exec(ctx, `CREATE TABLE test_users (id SERIAL PRIMARY KEY, name TEXT)`)
    require.NoError(t, err)

    // Act
    result, err := testDB.Exec(ctx, `INSERT INTO test_users (name) VALUES ($1)`, "John")
    require.NoError(t, err)

    // Assert
    affected, err := result.RowsAffected()
    require.NoError(t, err)
    require.Equal(t, int64(1), affected)
}
```

## Assertions (testify)
```go
// Fail immediately on error
require.NoError(t, err)
require.Equal(t, expected, actual)

// Continue test, fail at end
assert.NoError(t, err)
assert.Equal(t, expected, actual)

// Check for specific error
assert.ErrorIs(t, err, expectedErr)
```

**Rule**: Use `require` for setup/preconditions. Use `assert` for multiple independent checks.

## Resource Cleanup
Prefer `t.Cleanup()` over `defer` in tests:
```go
func TestDB(t *testing.T) {
    db, err := connect()
    require.NoError(t, err)
    t.Cleanup(func() { db.Close() })
    // ...
}
```

## Test Organization

### Structure
- **Unit tests**: `*_test.go` files in the package directory
- **Integration tests**: `test/*_test.go` subdirectory

### Naming Conventions
- Do NOT use "unit" or "integration" in file names
- Use descriptive names: `client_test.go`, `publisher_test.go`, `suite_test.go`

### Example Structure
```
adapter/adaptername/
├── client.go
├── client_test.go        # Unit tests
├── publisher.go
├── publisher_test.go     # Unit tests
└── test/
    ├── suite_test.go     # Test suite with container setup
    ├── client_test.go    # Integration tests for client
    └── publisher_test.go # Integration tests for publisher
```

### Mixed Test Files
If a file contains both unit and integration tests:
1. Move integration tests to `test/` subdirectory
2. Keep unit tests in the original file

## Test Suites
For complex setups with containers, use `testify/suite`:
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
    // Start container
}

func (s *MySuite) TearDownSuite() {
    // Stop container
}
```
