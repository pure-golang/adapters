---
name: "go_commands"
description: "Команды для запуска тестов, сборки и управления зависимостями Go-модуля"
modes: [Code, Debug, Orchestrator]
---
# Skill: Go Commands

## Tactical Instructions

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

# Run with verbose output
go test -v ./...

# Run a single test
go test -run TestFunctionName ./path/to/pkg
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
Integration tests use `github.com/testcontainers/testcontainers-go`.

**Docker must be running** for integration tests to pass:
```bash
docker ps  # verify Docker is running
```

Containers used:
- PostgreSQL: `postgres:15-alpine`
- RabbitMQ: `rabbitmq:management-alpine`

Use `-short` flag to skip integration tests in CI/CD when Docker is unavailable:
```bash
go test -short ./...
```
