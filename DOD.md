# Definition of Done (DOD) для Адаптеров

Этот документ определяет критерии готовности при добавлении новых адаптеров в проект `adapters`.

## Критерии готовности

### 1. Требования к коду

- [ ] Код соответствует конвенциям проекта (см. `AGENTS.md`)
- [ ] Структура `Config` с тегами `envconfig` для конфигурации через переменные окружения
- [ ] Реализация интерфейса `Provider` (или `RunableProvider` для долгоиграющих сервисов):
  ```go
  type Provider interface {
      Start() error
      io.Closer
  }
  ```
- [ ] Обработка ошибок с заворачиванием в контекст через `errors.Wrap()`
- [ ] Использование `context.Context` первым аргументом во всех операциях, которые могут блокироваться
- [ ] Реализация `defer` для корректной очистки ресурсов

### 2. Требования к документации

- [ ] Наличие `README.md` в директории адаптера с примерами использования
- [ ] Все комментарии и документация на русском языке
- [ ] Описание всех параметров конфигурации в README
- [ ] Примеры базовых операций (подключение, выполнение запросов, обработка ошибок)

### 3. Требования к тестированию

#### Обязательные тесты

- [ ] **Unit-тесты** для основной бизнес-логики
- [ ] **Интеграционные тесты** с использованием `github.com/testcontainers/testcontainers-go`

#### Интеграционные тесты (обязательно)

Интеграционные тесты **должны** использовать библиотеку `github.com/testcontainers/testcontainers-go` вместо `github.com/ory/dockertest`.

Требования к интеграционным тестам:

- [ ] Поддержка флага `-short` для пропуска Docker-тестов:
  ```go
  func skipShort(t *testing.T) {
      if testing.Short() {
          t.Skip("skipping integration test in short mode")
      }
  }
  ```
- [ ] Автоматический запуск и остановка контейнеров через testcontainers
- [ ] Ожидание готовности сервиса перед выполнением тестов
- [ ] Корректная очистка ресурсов после завершения тестов

#### Покрытие тестами

Для адаптеров баз данных и очередей сообщений:

- [ ] Тест подключения и установления соединения
- [ ] Тест базовых операций (CRUD)
- [ ] Тест обработки ошибок (включая нарушение ограничений базы данных)
- [ ] Тест транзакций (commit/rollback) — если применимо
- [ ] Тест уровней изоляции транзакций — если применимо
- [ ] Тест обработки таймаутов и отмены контекста
- [ ] Тест конкурентных операций

#### Пример структуры интеграционного теста с testcontainers-go

```go
package sqlx_test

import (
    "context"
    "testing"
    "time"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgres(t *testing.T) (string, func()) {
    ctx := context.Background()

    req := testcontainers.ContainerRequest{
        Image:        "postgres:15-alpine",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_USER":     "postgres",
            "POSTGRES_PASSWORD": "secret",
            "POSTGRES_DB":       "testdb",
        },
        WaitingFor: wait.ForLog("database system is ready to accept connections").
            WithOccurrence(2).
            WithStartupTimeout(10 * time.Second),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatalf("failed to start container: %v", err)
    }

    host, err := container.Host(ctx)
    if err != nil {
        t.Fatalf("failed to get container host: %v", err)
    }

    port, err := container.MappedPort(ctx, "5432")
    if err != nil {
        t.Fatalf("failed to get container port: %v", err)
    }

    cleanup := func() {
        if err := container.Terminate(ctx); err != nil {
            t.Logf("failed to terminate container: %v", err)
        }
    }

    return fmt.Sprintf("postgres://postgres:secret@%s:%s/testdb?sslmode=disable",
        host, port.Port()), cleanup
}
```

### 4. Требования к наблюдаемости (Observability)

- [ ] OpenTelemetry tracing для всех операций:
  - Инициализация трейсера как пакетной переменной
  - Именование спанов: `packageName.operation`
  - Стандартные атрибуты для операции

- [ ] Интеграция со структурированным логированием (`log/slog`):
  - Логирование операций
  - Логирование ошибок с контекстом

- [ ] Метрики (если применимо):
  - Prometheus metrics для операций
  - Метрики ошибок и latency

### 5. Дополнительные требования

- [ ] `go build ./...` выполняется без ошибок
- [ ] `go test ./...` выполняется без ошибок
- [ ] `go vet ./...` не выдаёт предупреждений
- [ ] `go mod tidy` не изменяет `go.mod` и `go.sum`
- [ ] Отсутствие `TODO` в коде адаптера (или заменены наIssue)
- [ ] Добавление описания нового адаптера в `AGENTS.md` при введении новых паттернов

## Чеклист для Pull Request

Перед созданием PR для нового адаптера:

1. [ ] Все пункты DOD выполнены
2. [ ] Тесты проходят локально (включая интеграционные при запущенном Docker)
3. [ ] Тесты проходят в CI
4. [ ] README.md содержит полную документацию
5. [ ] AGENTS.md обновлён при необходимости
6. [ ] Код отформатирован (`go fmt ./...`)
7. [ ] Нет лишних зависимений в `go.mod`
