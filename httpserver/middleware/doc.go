// Package middleware предоставляет HTTP middleware для серверов.
//
// Поддерживает:
//   - OpenTelemetry tracing (распределённая трассировка)
//   - Prometheus metrics (метрики запросов)
//   - Structured logging (логирование через slog)
//   - Recovery (восстановление после паники)
//
// Использование:
//
//	import "github.com/pure-golang/adapters/httpserver/middleware"
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/api", handler)
//
//	// Обёртка с мониторингом
//	handler := middleware.Monitoring(mux)
//
//	// Recovery middleware отдельно
//	handler = middleware.Recovery(handler, logger)
//
// Метрики:
//   - http.request_count — счётчик запросов
//   - http.request_time — гистограмма времени выполнения (ms)
//   - http.request_body_len — гистограмма размера запроса (KB)
//   - http.response_body_len — гистограмма размера ответа (KB)
//
// Особенности:
//   - Monitoring автоматически добавляет trace_id в заголовок X-Trace-Id
//   - Логирует тело запроса и ответа (первые 2048 байт)
//   - Автоматически извлекает и сохраняет logger в context
//   - Recovery логирует панику и возвращает 500
package middleware
