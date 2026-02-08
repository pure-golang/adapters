package middleware

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	meter = otel.Meter("github.com/pure-golang/adapters/grpc")

	requestsCount       metric.Int64Counter
	requestDuration     metric.Int64Histogram
	requestPayloadSize  metric.Int64Histogram
	responsePayloadSize metric.Int64Histogram
)

func init() {
	var err error

	requestsCount, err = meter.Int64Counter(
		"grpc.server.requests_total",
		metric.WithDescription("Total number of gRPC requests"),
	)
	if err != nil {
		panic(errors.Wrap(err, "failed to create requests counter"))
	}

	requestDuration, err = meter.Int64Histogram(
		"grpc.server.duration_ms",
		metric.WithDescription("gRPC request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(errors.Wrap(err, "failed to create request duration histogram"))
	}

	requestPayloadSize, err = meter.Int64Histogram(
		"grpc.server.request_size_bytes",
		metric.WithDescription("gRPC request size in bytes"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		panic(errors.Wrap(err, "failed to create request size histogram"))
	}

	responsePayloadSize, err = meter.Int64Histogram(
		"grpc.server.response_size_bytes",
		metric.WithDescription("gRPC response size in bytes"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		panic(errors.Wrap(err, "failed to create response size histogram"))
	}
}

// getMessageSize возвращает размер protobuf сообщения в байтах
func getMessageSize(msg interface{}) int64 {
	if pm, ok := msg.(proto.Message); ok {
		return int64(proto.Size(pm))
	}
	return 0
}

// MetricsUnaryInterceptor создает интерцептор для метрик gRPC запросов
func MetricsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// Измеряем размер запроса
		requestSize := getMessageSize(req)
		requestPayloadSize.Record(ctx, requestSize, metric.WithAttributes(
			attribute.String("grpc.method", info.FullMethod),
		))

		// Атрибуты для метрик
		metricAttrs := []attribute.KeyValue{
			attribute.String("grpc.method", info.FullMethod),
		}

		// Обрабатываем запрос
		resp, err := handler(ctx, req)

		// Измеряем размер ответа
		responseSize := getMessageSize(resp)
		responsePayloadSize.Record(ctx, responseSize, metric.WithAttributes(
			attribute.String("grpc.method", info.FullMethod),
		))

		// Записываем метрики
		duration := time.Since(startTime)
		requestDuration.Record(ctx, duration.Milliseconds(), metric.WithAttributes(metricAttrs...))

		// Добавляем код статуса
		statusCode := status.Code(err)
		statusAttrs := append(metricAttrs, attribute.String("grpc.status", statusCode.String()))
		requestsCount.Add(ctx, 1, metric.WithAttributes(statusAttrs...))

		return resp, err
	}
}

// MetricsStreamInterceptor создает интерцептор для метрик потоковых gRPC запросов
func MetricsStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		// Определяем тип потока
		streamType := "server_streaming"
		if info.IsClientStream {
			streamType = "client_streaming"
		}
		if info.IsClientStream && info.IsServerStream {
			streamType = "bidi_streaming"
		}

		metricAttrs := []attribute.KeyValue{
			attribute.String("grpc.method", info.FullMethod),
			attribute.String("stream.type", streamType),
		}

		// Обрабатываем поток
		err := handler(srv, ss)

		// Записываем метрики
		duration := time.Since(startTime)
		requestDuration.Record(ss.Context(), duration.Milliseconds(), metric.WithAttributes(metricAttrs...))

		// Добавляем код статуса
		statusCode := status.Code(err)
		statusAttrs := append(metricAttrs, attribute.String("grpc.status", statusCode.String()))
		requestsCount.Add(ss.Context(), 1, metric.WithAttributes(statusAttrs...))

		return err
	}
}
