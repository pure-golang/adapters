---
name: "testing_conventions"
description: "Соглашения по тестам: маркеры unit/integration, AAA-паттерн, assertions, t.Parallel, t.Cleanup"
modes: [Code, Review]
---
# Skill: Testing Conventions

## Tactical Instructions

### Test Type Markers

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

### AAA Pattern (Arrange, Act, Assert)
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

### Assertions (testify)
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

### Resource Cleanup
Prefer `t.Cleanup()` over `defer` in tests:
```go
func TestDB(t *testing.T) {
    db, err := connect()
    require.NoError(t, err)
    t.Cleanup(func() { db.Close() })
    // ...
}
```

### Test Organization
- Unit tests: Standard `*_test.go` files in the same package
- Integration tests: Use `testify/suite` for complex setups with containers
- Suites: `SetupSuite()` / `TearDownSuite()` for container lifecycle
