package middleware

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
)

// MonitoringOptions содержит настройки мониторинга
type MonitoringOptions struct {
	Logger             *slog.Logger
	EnableTracing      bool
	EnableMetrics      bool
	EnableLogging      bool
	EnableStatsHandler bool
}

// DefaultMonitoringOptions возвращает настройки по умолчанию
func DefaultMonitoringOptions(logger *slog.Logger) *MonitoringOptions {
	return &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      true,
		EnableLogging:      true,
		EnableStatsHandler: true,
	}
}

// SetupMonitoring настраивает все компоненты мониторинга для gRPC сервера
// и возвращает необходимые интерцепторы и опции сервера
func SetupMonitoring(
	ctx context.Context,
	options *MonitoringOptions,
) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor, []grpc.ServerOption) {

	unaryInterceptors := []grpc.UnaryServerInterceptor{}
	streamInterceptors := []grpc.StreamServerInterceptor{}
	serverOptions := []grpc.ServerOption{}

	// Настраиваем трассировку OpenTelemetry
	if options.EnableTracing {
		// Устанавливаем пропагатор контекста для трассировки
		otel.SetTextMapPropagator(MetadataTextMapPropagator())

		// Добавляем интерцепторы трассировки
		unaryInterceptors = append(unaryInterceptors, TracingUnaryInterceptor())
		streamInterceptors = append(streamInterceptors, TracingStreamInterceptor())

		// Добавляем StatsHandler для дополнительных метрик трассировки
		if options.EnableStatsHandler {
			serverOptions = append(serverOptions, grpc.StatsHandler(otelgrpc.NewServerHandler()))
		}
	}

	// Добавляем метрики Prometheus
	if options.EnableMetrics {
		unaryInterceptors = append(unaryInterceptors, MetricsUnaryInterceptor())
		streamInterceptors = append(streamInterceptors, MetricsStreamInterceptor())
	}

	// Добавляем логирование и восстановление после паники
	if options.EnableLogging {
		unaryInterceptors = append(unaryInterceptors,
			RecoveryInterceptor(options.Logger),
			LoggingInterceptor(options.Logger),
		)
		streamInterceptors = append(streamInterceptors,
			RecoveryStreamInterceptor(options.Logger),
			LoggingStreamInterceptor(options.Logger),
		)
	}

	return unaryInterceptors, streamInterceptors, serverOptions
}
