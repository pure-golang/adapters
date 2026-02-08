package redis

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/pure-golang/adapters/kv/redis")

// startSpan создаёт новый span с атрибутами ключа и базы данных
func startSpan(ctx context.Context, operation string, key string, db int) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("db.system", "redis"),
	}
	if key != "" {
		attrs = append(attrs, attribute.String("redis.key", key))
	}
	if db > 0 {
		attrs = append(attrs, attribute.Int("redis.db", db))
	}
	return tracer.Start(ctx, "redis."+operation, trace.WithAttributes(attrs...))
}

// recordError записывает ошибку в спан
func recordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
}
