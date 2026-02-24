---
name: "troubleshooting"
description: "Диагностика проблем: Docker, интеграционные тесты, context deadline, ошибки подключения к БД/RabbitMQ/S3"
modes: [Debug]
---
# Skill: Troubleshooting

## Tactical Instructions

### Integration Tests Fail
```bash
# Check if Docker is running
docker ps

# Check port availability
lsof -i :5432   # PostgreSQL
lsof -i :5672   # RabbitMQ
lsof -i :9000   # MinIO

# Run with verbose output
go test -v ./...
```

### Context Deadline Exceeded
- Query timeout may be too short → check `Config.QueryTimeout`
- Increase timeout in config or pass a longer-lived context
- In integration tests use `context.WithTimeout(context.Background(), 5*time.Second)`

### Database Connection Errors (PostgreSQL)
- Verify PostgreSQL is running and accessible
- Check DSN format: `postgres://user:pass@host:port/dbname?sslmode=disable`
- Verify credentials and database name
- Check firewall rules for port 5432

### RabbitMQ Connection Errors
- Verify RabbitMQ is running and accessible
- Check URL format: `amqp://user:pass@host:port/`
- Verify user permissions and queue/exchange existence
- Check firewall rules for port 5672

### S3 Storage Connection Errors
- Verify S3-compatible storage is running
- Check endpoint format:
  - MinIO: `localhost:9000`
  - Yandex Cloud: `storage.yandexcloud.net`
  - AWS S3: `s3.amazonaws.com`
- Verify access key and secret key
- Verify bucket exists and you have permissions
- Check firewall rules for the S3 port (MinIO: 9000, AWS/Yandex: 443)

### Quick Diagnostics Checklist
1. `docker ps` — is Docker running?
2. Port free? `lsof -i :{port}`
3. Credentials correct in `.env`?
4. Running `go test -short` by mistake? Remove `-short` for integration tests.
