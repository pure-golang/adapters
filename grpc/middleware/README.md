# gRPC Middleware

Пакет `grpc/middleware` предоставляет набор интерцепторов для мониторинга, трассировки и логирования gRPC-запросов.

## Возможности

- Мониторинг: Сбор метрик о запросах
- Трассировка: Поддержка OpenTelemetry
- Логирование: Структурированные логи запросов
- Восстановление после паники

## Использование

### Простой пример

Самый простой способ подключить все компоненты - использовать функцию `SetupMonitoring`:

```go
import (
    "context"
    "log/slog"
    
    "google.golang.org/grpc"
    "github.com/pure-golang/adapters/grpc/middleware"
)

func SetupGRPCServer(logger *slog.Logger) *grpc.Server {
    // Настройка мониторинга с параметрами по умолчанию
    monitoringOptions := middleware.DefaultMonitoringOptions(logger)
    unaryInterceptors, streamInterceptors, serverOptions := middleware.SetupMonitoring(
        context.Background(), 
        monitoringOptions,
    )
    
    // Создание gRPC сервера со всеми интерцепторами
    serverOpts := append(serverOptions, 
        grpc.ChainUnaryInterceptor(unaryInterceptors...),
        grpc.ChainStreamInterceptor(streamInterceptors...),
    )
    
    return grpc.NewServer(serverOpts...)
}
```

### Расширенный пример

Если требуется тонкая настройка мониторинга:

```go
import (
    "context"
    "log/slog"
    
    "google.golang.org/grpc"
    "github.com/pure-golang/adapters/grpc/middleware"
)

func SetupCustomGRPCServer(logger *slog.Logger) *grpc.Server {
    // Кастомная конфигурация мониторинга
    monitoringOptions := &middleware.MonitoringOptions{
        Logger:             logger,
        EnableTracing:      true,     // Включить трассировку
        EnableMetrics:      true,     // Включить метрики Prometheus
        EnableLogging:      true,     // Включить логирование
        EnableStatsHandler: false,    // Отключить StatsHandler
    }
    
    unaryInterceptors, streamInterceptors, serverOptions := middleware.SetupMonitoring(
        context.Background(), 
        monitoringOptions,
    )
    
    // Добавление собственных интерцепторов
    customInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // Кастомная логика
        return handler(ctx, req)
    }
    
    unaryInterceptors = append(unaryInterceptors, customInterceptor)
    
    // Создание gRPC сервера
    serverOpts := append(serverOptions, 
        grpc.ChainUnaryInterceptor(unaryInterceptors...),
        grpc.ChainStreamInterceptor(streamInterceptors...),
    )
    
    return grpc.NewServer(serverOpts...)
}
```

## Компоненты

### Трассировка (Tracing)

Трассировка основана на OpenTelemetry и предоставляет:

- Автоматическое создание спанов для каждого запроса
- Пропагацию контекста трассировки между сервисами
- Обогащение спанов информацией о статусе, длительности и ошибках
- Поддержку потоковых (streaming) операций

### Метрики (Metrics)

Собираются следующие метрики:

- `grpc.server.requests_total` - счетчик запросов с метками метода и статуса
- `grpc.server.duration_ms` - гистограмма длительности запросов
- `grpc.server.request_size_bytes` - гистограмма размеров запросов
- `grpc.server.response_size_bytes` - гистограмма размеров ответов

### Логирование (Logging)

Логирование предоставляет:

- Структурированные логи для каждого запроса
- Информацию о длительности вызова
- Коды статусов ответов
- Детали ошибок при неудачных запросах
- Восстановление после паники с логированием

## Интеграция с адаптером gRPC

Весь мониторинг уже интегрирован с адаптером gRPC и включен по умолчанию:

```go
import (
    "github.com/pure-golang/adapters/grpc/std"
)

func main() {
    config := std.Config{
        Host: "",
        Port: 50051,
        EnableReflect: true,
    }
    
    server := std.NewDefault(config, func(grpcServer *grpc.Server) {
        // Регистрация сервисов
        pb.RegisterMyServiceServer(grpcServer, &myServiceImpl{})
    })
    
    server.Start()
}
```

## Обработка ошибок

Пакет `grpc/errors` предоставляет вспомогательные функции для преобразования ошибок Go в статусы gRPC:

```go
import (
    grpcerrors "github.com/pure-golang/adapters/grpc/errors"
    "google.golang.org/grpc/codes"
)

func (s *service) MyMethod(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    result, err := s.repository.Find(req.ID)
    if err != nil {
        return nil, grpcerrors.FromError(err) // Автоопределение типа ошибки
    }
    
    if result == nil {
        return nil, grpcerrors.NewError(codes.NotFound, "resource not found")
    }
    
    return &pb.Response{Data: result}, nil
}
```

## Полезные ссылки

- [OpenTelemetry для Go](https://opentelemetry.io/docs/instrumentation/go/)
- [gRPC Interceptors](https://github.com/grpc-ecosystem/go-grpc-middleware)
- [OpenTelemetry gRPC Integration](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc)
