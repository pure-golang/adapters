package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor создает интерцептор для логирования gRPC запросов
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		logAttrs := []any{
			slog.String("method", info.FullMethod),
			slog.Duration("duration", duration),
		}

		// Добавляем информацию о статусе
		if err != nil {
			s := status.Convert(err)
			logAttrs = append(logAttrs,
				slog.String("status_code", s.Code().String()),
				slog.Any("error", err),
			)
			logger.ErrorContext(ctx, "gRPC request failed", logAttrs...)
		} else {
			logAttrs = append(logAttrs, slog.String("status_code", "OK"))
			logger.InfoContext(ctx, "gRPC request processed", logAttrs...)
		}

		return resp, err
	}
}

// RecoveryInterceptor создает интерцептор для восстановления после паники
func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(ctx, "Recovered from panic in gRPC handler",
					slog.Any("panic", r),
					slog.String("method", info.FullMethod),
				)
				err = status.Error(14, "internal server error") // UNAVAILABLE
			}
		}()
		return handler(ctx, req)
	}
}

// LoggingStreamInterceptor создает интерцептор для логирования потоковых gRPC запросов
func LoggingStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		duration := time.Since(start)

		logAttrs := []any{
			slog.String("method", info.FullMethod),
			slog.Duration("duration", duration),
			slog.Bool("client_stream", info.IsClientStream),
			slog.Bool("server_stream", info.IsServerStream),
		}

		if err != nil {
			s := status.Convert(err)
			logAttrs = append(logAttrs,
				slog.String("status_code", s.Code().String()),
				slog.Any("error", err),
			)
			logger.ErrorContext(ss.Context(), "gRPC stream failed", logAttrs...)
		} else {
			logAttrs = append(logAttrs, slog.String("status_code", "OK"))
			logger.InfoContext(ss.Context(), "gRPC stream processed", logAttrs...)
		}

		return err
	}
}

// RecoveryStreamInterceptor создает интерцептор для восстановления в потоковых запросах
func RecoveryStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(ss.Context(), "Recovered from panic in gRPC stream handler",
					slog.Any("panic", r),
					slog.String("method", info.FullMethod),
				)
				err = status.Error(14, "internal server error") // UNAVAILABLE
			}
		}()
		return handler(srv, ss)
	}
}
