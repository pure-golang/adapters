package middleware

import (
	"context"
	"path"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var tracer = otel.Tracer("github.com/pure-golang/adapters/grpc")

// metadataSupplier реализует propagation.TextMapCarrier для gRPC метаданных
type metadataSupplier struct {
	metadata *metadata.MD
}

func (s metadataSupplier) Get(key string) string {
	values := s.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (s metadataSupplier) Set(key string, value string) {
	s.metadata.Set(key, value)
}

func (s metadataSupplier) Keys() []string {
	keys := make([]string, 0, 10)
	for key := range *s.metadata {
		keys = append(keys, key)
	}
	return keys
}

// TracingUnaryInterceptor создает интерцептор для трассировки унарных RPC
func TracingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Извлекаем метаданные
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Получаем сервис и метод из полного пути
		service, method := splitMethodName(info.FullMethod)

		// Создаем контекст с пропагацией трассировки
		var span trace.Span
		ctx = otel.GetTextMapPropagator().Extract(ctx, metadataSupplier{metadata: &md})

		// Начинаем новый спан
		ctx, span = tracer.Start(
			ctx,
			path.Join(service, method),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", service),
				attribute.String("rpc.method", method),
			),
		)
		defer span.End()

		startTime := time.Now()

		// Записываем запрос в спан как событие
		span.AddEvent("request_received", trace.WithAttributes(
			attribute.String("request.type", "unary"),
		))

		// Вызываем обработчик с контекстом, содержащим спан
		resp, err := handler(ctx, req)

		// Записываем длительность
		duration := time.Since(startTime)
		span.SetAttributes(attribute.Int64("request.duration_ms", duration.Milliseconds()))

		// Обработка ошибок
		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.SetAttributes(
				attribute.String("rpc.status_code", s.Code().String()),
				attribute.String("error.message", err.Error()),
			)
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(attribute.String("rpc.status_code", "OK"))
		}

		// Записываем ответ как событие
		span.AddEvent("response_sent", trace.WithAttributes(
			attribute.Bool("response.error", err != nil),
		))

		return resp, err
	}
}

// TracingStreamInterceptor создает интерцептор для трассировки потоковых RPC
func TracingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Извлекаем метаданные
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Получаем сервис и метод из полного пути
		service, method := splitMethodName(info.FullMethod)

		// Создаем контекст с пропагацией трассировки
		var span trace.Span
		ctx = otel.GetTextMapPropagator().Extract(ctx, metadataSupplier{metadata: &md})

		// Начинаем новый спан
		streamType := "server_streaming"
		if info.IsClientStream {
			streamType = "client_streaming"
		}
		if info.IsClientStream && info.IsServerStream {
			streamType = "bidi_streaming"
		}

		ctx, span = tracer.Start(
			ctx,
			path.Join(service, method),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", service),
				attribute.String("rpc.method", method),
				attribute.String("stream.type", streamType),
			),
		)
		defer span.End()

		startTime := time.Now()

		// Оборачиваем ServerStream, чтобы использовать контекст с трассировкой
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Записываем начало обработки потока
		span.AddEvent("stream_started")

		// Обрабатываем поток
		err := handler(srv, wrappedStream)

		// Записываем длительность
		duration := time.Since(startTime)
		span.SetAttributes(attribute.Int64("stream.duration_ms", duration.Milliseconds()))

		// Обработка ошибок
		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.SetAttributes(
				attribute.String("rpc.status_code", s.Code().String()),
				attribute.String("error.message", err.Error()),
			)
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(attribute.String("rpc.status_code", "OK"))
		}

		// Записываем завершение потока
		span.AddEvent("stream_ended")

		return err
	}
}

// wrappedServerStream оборачивает grpc.ServerStream для использования контекста с трассировкой
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// splitMethodName разделяет полное имя метода на имя сервиса и метода
func splitMethodName(fullMethodName string) (string, string) {
	fullMethodName = path.Clean(fullMethodName)
	if !path.IsAbs(fullMethodName) {
		return "unknown", "unknown"
	}
	i := path.Dir(fullMethodName)
	j := path.Base(fullMethodName)
	if i == "." || i == "/" {
		return "unknown", j
	}
	return i[1:], j // удаляем начальный "/" из имени сервиса
}

// MetadataTextMapPropagator возвращает пропагатор контекста трассировки через метаданные gRPC
func MetadataTextMapPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
