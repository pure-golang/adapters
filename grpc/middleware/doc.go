// Package middleware предоставляет интерцепторы для gRPC серверов.
//
// Поддерживает:
//   - OpenTelemetry tracing (распределённая трассировка)
//   - Prometheus metrics (метрики запросов)
//   - Structured logging (логирование через slog)
//   - Recovery (восстановление после паники)
//
// Использование (SetupMonitoring):
//
//	import "github.com/pure-golang/adapters/grpc/middleware"
//
//	opts := middleware.DefaultMonitoringOptions(logger)
//	unary, stream, serverOpts := middleware.SetupMonitoring(ctx, opts)
//	server := grpc.NewServer(append(serverOpts,
//	    grpc.ChainUnaryInterceptor(unary...),
//	    grpc.ChainStreamInterceptor(stream...),
//	)...)
//
// Использование (отдельные интерцепторы):
//
//	// Tracing
//	unary := middleware.TracingUnaryInterceptor()
//	stream := middleware.TracingStreamInterceptor()
//
//	// Metrics
//	unary := middleware.MetricsUnaryInterceptor()
//	stream := middleware.MetricsStreamInterceptor()
//
//	// Logging
//	unary := middleware.LoggingInterceptor(logger)
//	stream := middleware.LoggingStreamInterceptor(logger)
//
//	// Recovery
//	unary := middleware.RecoveryInterceptor(logger)
//	stream := middleware.RecoveryStreamInterceptor(logger)
//
// Порядок интерцепторов (от внешнего к внутреннему):
//  1. Tracing — создание span'ов
//  2. Metrics — сбор метрик
//  3. Recovery — перехват паник, возврат ошибки
//  4. Logging — логирование запросов
//
// Tracing и Metrics ПЕРЕД Recovery: паника → Recovery возвращает ошибку → Tracing/Metrics фиксируют её.
package middleware
