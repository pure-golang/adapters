# CLI Executor

Адаптер для выполнения внешних CLI утилит.

## Установка

Команда должна быть доступна в системе (в PATH).

## Использование

### Создание executor

```go
import (
    "github.com/pure-golang/adapters/executor/cli"
)

cfg := cli.Config{
    Command: "ffmpeg",
}

executor := cli.New(cfg)
defer executor.Close()
```

### Выполнение команды

```go
ctx := context.Background()

// Конвертация видео через FFmpeg
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

## Примеры использования

### FFmpeg - конвертация видео

```go
cfg := cli.Config{Command: "ffmpeg"}
executor := cli.New(cfg)
defer executor.Close()

ctx := context.Background()
_, err = executor.Execute(ctx,
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
executor := cli.New(cfg)
defer executor.Close()

ctx := context.Background()
_, err := executor.Execute(ctx, "cp", "local-file.txt", "gs://bucket/remote-file.txt")
```

### AWS CLI - управление S3

```go
cfg := cli.Config{Command: "aws"}
executor := cli.New(cfg)
defer executor.Close()

ctx := context.Background()
output, err := executor.Execute(ctx, "s3", "ls", "s3://my-bucket/")
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(output))
```

### ImageMagick - обработка изображений

```go
cfg := cli.Config{Command: "convert"}
executor := cli.New(cfg)
defer executor.Close()

ctx := context.Background()
_, err := executor.Execute(ctx,
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

func (m *mockExecutor) Execute(ctx context.Context, args ...string) ([]byte, error) {
    return []byte("mock output"), nil
}

func (m *mockExecutor) Start() error {
    return nil
}

func (m *mockExecutor) Close() error {
    return nil
}
```

## Обработка ошибок

```go
ctx := context.Background()
output, err := executor.Execute(ctx, args...)
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

output, err := executor.Execute(ctx, args...)
if errors.Is(err, context.DeadlineExceeded) {
    log.Println("Команда превысила таймаут")
}
```
