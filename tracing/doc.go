// Package tracing определяет интерфейс для распределённой трассировки.
//
// Пакет предоставляет интерфейс [Provider] для OpenTelemetry tracer провайдеров.
// Реализации находятся в дочерних пакетах:
//   - [tracing/jaeger] — Jaeger/OTLP экспортер
//
// Интерфейсы:
//   - [Provider] — объединяет trace.TracerProvider и io.Closer
//   - [ProviderBuilder] — фабричная функция для создания провайдера
//
// Использование:
//
//	import "github.com/pure-golang/adapters/tracing"
//	import "github.com/pure-golang/adapters/tracing/jaeger"
//
//	provider, err := tracing.Init(jaeger.NewProviderBuilder(cfg))
//	if err != nil {
//	    log.Warn("tracing unavailable", "error", err) // только предупреждение
//	}
//	defer provider.Close()
//
// Важно: ошибка инициализации не должна останавливать приложение.
// Init всегда возвращает рабочий провайдер (NoopProvider при ошибке).
// Вызывающий код обязан обработать ошибку как warn, но не как fatal —
// мониторинг не должен влиять на доступность сервиса.
//
// Особенности:
//   - При ошибке инициализации возвращает NoopProvider
//   - Автоматически устанавливает глобальный TracerProvider
//   - Устанавливает TraceContext propagator
//   - Close() вызывает ForceFlush и Shutdown
package tracing
