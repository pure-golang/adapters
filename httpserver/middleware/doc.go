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
//	import amiddleware "github.com/pure-golang/adapters/httpserver/middleware"
//
//	handler := amiddleware.Chain(
//	    mux,
//	    amiddleware.Monitoring("/webhooks/livekit"),
//	    amiddleware.Recovery,
//	    authMiddleware,
//	)
//
// Порядок middleware (от внешнего к внутреннему):
//  1. Monitoring — трейсинг, метрики, логирование (видит все ответы, включая 500 от Recovery).
//  2. Recovery — перехват паник, возврат 500.
//  3. Auth / прикладные middleware.
//  4. Обработчик приложения.
//
// Monitoring ПЕРЕД Recovery: паника → Recovery отдаёт 500 → Monitoring фиксирует его в метриках и трейсах.
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
