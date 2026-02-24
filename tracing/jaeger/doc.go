// Package jaeger реализует [tracing.Provider] для Jaeger через OTLP.
//
// Использует OTLP HTTP exporter для отправки трейсов в Jaeger.
// Поддерживает Jaeger версии 1.35+ с OTLP collector.
//
// Использование:
//
//	import "github.com/pure-golang/adapters/tracing/jaeger"
//
//	cfg := jaeger.Config{
//	    EndPoint:    "http://localhost:4318/v1/traces",
//	    ServiceName: "my-service",
//	    AppVersion:  "1.0.0",
//	}
//
//	provider, err := jaeger.NewProviderBuilder(cfg)()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
// Конфигурация через переменные окружения:
//
//	TRACING_ENDPOINT — URL OTLP collector (required)
//	SERVICE_NAME     — имя сервиса для трейсов (required)
//	APP_VERSION      — версия приложения (required)
//
// Особенности:
//   - Использует OTLP HTTP протокол (порт 4318)
//   - Batch экспорт трейсов
//   - AlwaysSample сэмплер (все трейсы отправляются)
//   - Автоматическое добавление service.name и service.version
//   - Graceful shutdown через Close()
package jaeger
