# CLI Executor

Адаптер для выполнения внешних CLI утилит.

## Установка

Команда должна быть доступна в системе (в PATH).

## Конфигурация через переменные окружения

Адаптер поддерживает загрузку конфигурации из переменных окружения с помощью `envconfig`:

### Параметры конфигурации

| Переменная окружения | Описание | Обязательный | По умолчанию |
|---------------------|----------|--------------|--------------|
| `CLI_COMMAND` | Имя исполняемой команды (например, "ffmpeg", "gsutil", "aws") | Да | - |

### Пример использования с переменными окружения

```go
import (
    "github.com/pure-golang/adapters/executor/cli"
)

// Загрузка конфигурации из переменных окружения
var cfg cli.Config
if err := cli.InitConfig(&cfg); err != nil {
    log.Fatalf("ошибка загрузки конфигурации: %v", err)
}

executor := cli.New(cfg, nil, nil)
defer executor.Close()
```

### Пример файла .env

```bash
# CLI Executor
CLI_COMMAND=ffmpeg
```

### Прямое создание конфигурации

```go
import (
    "github.com/pure-golang/adapters/executor/cli"
)

cfg := cli.Config{
    Command: "ffmpeg",
}

executor := cli.New(cfg, nil, nil)
defer executor.Close()
```

## Использование

### Создание executor

### Проверка наличия команды

Метод `Start()` проверяет, что команда доступна в системе (в PATH):

```go
// Проверка наличия команды перед использованием
err := executor.Start()
if err != nil {
    log.Fatalf("Команда не найдена: %v", err)
}
```

### Выполнение команды

```go
ctx := context.Background()

// Конвертация видео через FFmpeg
err := executor.Execute(ctx,
    "-i", "input.mp4",
    "-c:v", "libx264",
    "-c:a", "aac",
    "-y", "output.mp4",
)
if err != nil {
    log.Fatal(err)
}
```

## Примеры использования

### FFmpeg - конвертация видео

```go
cfg := cli.Config{Command: "ffmpeg"}
executor := cli.New(cfg, nil, nil)
defer executor.Close()

// Проверка наличия FFmpeg
if err := executor.Start(); err != nil {
    log.Fatalf("FFmpeg не установлен: %v", err)
}

ctx := context.Background()
err = executor.Execute(ctx,
    "-i", "input.avi",
    "-c:v", "libx264",
    "-preset", "medium",
    "-crf", "23",
    "-c:a", "aac",
    "-b:a", "192k",
    "-movflags", "+faststart",
    "-y", "output.mp4",
)
```

### GSutil - загрузка файлов в Google Cloud Storage

```go
cfg := cli.Config{Command: "gsutil"}
executor := cli.New(cfg, nil, nil)
defer executor.Close()

// Проверка наличия gsutil
if err := executor.Start(); err != nil {
    log.Fatalf("gsutil не установлен: %v", err)
}

ctx := context.Background()
err := executor.Execute(ctx, "cp", "local-file.txt", "gs://bucket/remote-file.txt")
```

### AWS CLI - управление S3

```go
cfg := cli.Config{Command: "aws"}
executor := cli.New(cfg, nil, nil)
defer executor.Close()

// Проверка наличия AWS CLI
if err := executor.Start(); err != nil {
    log.Fatalf("AWS CLI не установлен: %v", err)
}

ctx := context.Background()
err := executor.Execute(ctx, "s3", "ls", "s3://my-bucket/")
if err != nil {
    log.Fatal(err)
}
```

### ImageMagick - обработка изображений

```go
cfg := cli.Config{Command: "convert"}
executor := cli.New(cfg, nil, nil)
defer executor.Close()

// Проверка наличия ImageMagick
if err := executor.Start(); err != nil {
    log.Fatalf("ImageMagick не установлен: %v", err)
}

ctx := context.Background()
err := executor.Execute(ctx,
    "input.jpg",
    "-resize", "800x600",
    "-quality", "85",
    "output.jpg",
)
```

## Мокирование для тестов

```go
// Создайте mock executor для тестов
type mockExecutor struct{}

func (m *mockExecutor) Start() error {
    return nil
}

func (m *mockExecutor) Execute(ctx context.Context, args ...string) error {
    return nil
}

func (m *mockExecutor) Close() error {
    return nil
}
```

## Обработка ошибок

```go
// Проверка наличия команды
err := executor.Start()
if err != nil {
    log.Printf("Команда не найдена: %v\n", err)
    return
}

ctx := context.Background()
err = executor.Execute(ctx, args...)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Таймаут выполнения команды")
    } else if strings.Contains(err.Error(), "command not found") {
        log.Println("Команда не найдена")
    } else {
        log.Printf("Ошибка выполнения: %v\n", err)
    }
    return
}
```

## Ограничения контекста

Вы можете использовать контекст для ограничения времени выполнения:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

err := executor.Execute(ctx)
if errors.Is(err, context.DeadlineExceeded) {
    log.Println("Команда превысила таймаут")
}
```

## Потоковая обработка вывода

Использование `io.Writer` в конструкторе позволяет обрабатывать вывод в реальном времени, что полезно для команд с большим объёмом вывода или для мониторинга логов:

**Правила:**
- Если `stdout` задан в конструкторе — используется потоковая запись
- Если `stderr` равен `nil` — используется `os.Stderr` по умолчанию

```go
import (
    "bufio"
    "io"
)

// Потоковая обработка строк в реальном времени
r, w := io.Pipe()
go func() {
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        line := scanner.Text()
        // Обработка каждой строки в реальном времени
        fmt.Printf("LOG: %s\n", line)
    }
}()

cfg := cli.Config{
    Command: "tail",
}
executor := cli.New(cfg, w, nil) // stdout = w, stderr = nil (используется os.Stderr)
defer executor.Close()

ctx := context.Background()
err := executor.Execute(ctx, "-f", "/var/log/app.log")
if err != nil {
    log.Fatal(err)
}
```

### Запись вывода в файл

```go
f, err := os.Create("output.log")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

cfg := cli.Config{
    Command: "my-command",
}
executor := cli.New(cfg, f, nil) // stdout = f, stderr = nil (используется os.Stderr)
defer executor.Close()

ctx := context.Background()
err = executor.Execute(ctx, "arg1", "arg2")
```

### Комбинирование нескольких writers

```go
import "io"

// Запись и в файл, и в stdout
f, _ := os.Create("output.log")
defer f.Close()

multiWriter := io.MultiWriter(f, os.Stdout)

cfg := cli.Config{
    Command: "my-command",
}
executor := cli.New(cfg, multiWriter, nil) // stdout = multiWriter, stderr = nil (используется os.Stderr)
defer executor.Close()

ctx := context.Background()
err = executor.Execute(ctx, "arg1", "arg2")
```

## Метрики

Адаптер автоматически собирает Prometheus метрики для всех выполняемых команд:

### Доступные метрики

| Метрика | Тип | Описание | Лейблы |
|---------|-----|----------|--------|
| `executor_cli_duration_seconds` | Histogram | Длительность выполнения CLI команд | `command`, `status` |
| `executor_cli_executions_total` | Counter | Общее количество выполнений CLI команд | `command`, `status` |

### Лейблы

- `command`: имя выполняемой команды (например, "ffmpeg", "gsutil")
- `status`: статус выполнения ("success" или "error")

### Пример использования метрик

```go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

// Запуск HTTP сервера для экспорта метрик
go func() {
    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":8080", nil))
}()

// После выполнения команд метрики будут доступны по адресу http://localhost:8080/metrics
```

### Пример вывода метрик

```
# HELP executor_cli_duration_seconds Длительность выполнения CLI команд
# TYPE executor_cli_duration_seconds histogram
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="0.005"} 0
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="0.01"} 0
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="0.025"} 0
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="0.05"} 0
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="0.1"} 0
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="0.25"} 0
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="0.5"} 0
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="1"} 1
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="2.5"} 1
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="5"} 1
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="10"} 1
executor_cli_duration_seconds_bucket{command="ffmpeg",status="success",le="+Inf"} 1
executor_cli_duration_seconds_sum{command="ffmpeg",status="success"} 0.876
executor_cli_duration_seconds_count{command="ffmpeg",status="success"} 1

# HELP executor_cli_executions_total Общее количество выполнений CLI команд
# TYPE executor_cli_executions_total counter
executor_cli_executions_total{command="ffmpeg",status="success"} 1
executor_cli_executions_total{command="ffmpeg",status="error"} 0
```

## Логирование

Адаптер использует структурированное логирование через `log/slog` для записи всех операций:

### Уровни логирования

- `INFO`: успешные операции (проверка команды, выполнение команды)
- `ERROR`: ошибки (команда не найдена, ошибка выполнения)

### Примеры логов

```json
{"time":"2026-02-11T16:00:00.000Z","level":"INFO","msg":"проверка наличия команды","command":"ffmpeg"}
{"time":"2026-02-11T16:00:00.001Z","level":"INFO","msg":"команда найдена","command":"ffmpeg"}
{"time":"2026-02-11T16:00:01.000Z","level":"INFO","msg":"выполнение команды","command":"ffmpeg","args":["-i","input.mp4","output.mp4"]}
{"time":"2026-02-11T16:00:01.876Z","level":"INFO","msg":"команда выполнена успешно","command":"ffmpeg","args":["-i","input.mp4","output.mp4"],"duration_seconds":0.876}
```

### Настройка логгера

```go
import (
    "log/slog"
    "os"
)

// Настройка JSON логгера для production
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
slog.SetDefault(logger)

// Настройка текстового логгера для development
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
slog.SetDefault(logger)
```

### Пример логов при ошибках

```json
{"time":"2026-02-11T16:00:00.000Z","level":"ERROR","msg":"команда не найдена","command":"nonexistent","error":"executable file not found in $PATH"}
{"time":"2026-02-11T16:00:01.000Z","level":"ERROR","msg":"ошибка выполнения команды","command":"ffmpeg","args":["-i","input.mp4","output.mp4"],"error":"exit status 1","duration_seconds":0.123}
```

## Наблюдаемость (Observability)

Адаптер полностью поддерживает наблюдаемость через:

1. **OpenTelemetry Tracing**: все операции выполняются с трассировкой
2. **Prometheus Metrics**: автоматический сбор метрик выполнения команд
3. **Structured Logging**: структурированные логи через `log/slog`

### Пример интеграции с OpenTelemetry

```go
import (
    "go.opentelemetry.io/otel"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Инициализация трейсера
tp := sdktrace.NewTracerProvider()
otel.SetTracerProvider(tp)
defer tp.Shutdown(context.Background())

// После инициализации все операции executor будут автоматически трассироваться
```

## Тестирование

### Запуск тестов

```bash
# Запуск всех тестов
go test ./executor/cli/...

# Запуск только unit-тестов (без интеграционных)
go test -short ./executor/cli/...

# Запуск с детальным выводом
go test -v ./executor/cli/...
```

### Интеграционные тесты

Интеграционные тесты требуют наличия установленных в системе команд (echo, sh, sleep, printf). Для пропуска интеграционных тестов используйте флаг `-short`:

```bash
# Пропуск интеграционных тестов
go test -short ./executor/cli/...
```

### Покрытие тестов

```bash
# Запуск тестов с покрытием
go test -cover ./executor/cli/...

# Запуск тестов с детальным покрытием
go test -coverprofile=coverage.out ./executor/cli/...
go tool cover -html=coverage.out
```

