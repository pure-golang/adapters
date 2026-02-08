package sqlx

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/pure-golang/adapters/db/pg/sqlx")

// TracingConfig содержит настройки для трейсинга
type TracingConfig struct {
	// CommentsAsAttributes добавляет SQL-комментарии как атрибуты спана
	CommentsAsAttributes bool
	// ExcludeErrors список ошибок, которые не нужно записывать в трейсинг
	ExcludeErrors []error
	// DisableErrSkip отключает пропуск ошибок из ExcludeErrors
	DisableErrSkip bool
}

// DefaultTracingConfig возвращает конфигурацию по умолчанию
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		CommentsAsAttributes: true,
		ExcludeErrors:        []error{pgx.ErrNoRows},
		DisableErrSkip:       false,
	}
}

// WithTracing добавляет трейсинг к конфигурации подключения
func WithTracing(cfg *Config, _ *TracingConfig) *Config {
	return cfg
}

// TracingQuerier определяет интерфейс для выполнения запросов с трейсингом
type TracingQuerier interface {
	Querier
	WithTracing(ctx context.Context, operation string, query string) (context.Context, trace.Span)
}

// WithTracing создает новый спан для операции с базой данных
func (c *Connection) WithTracing(ctx context.Context, operation string, query string) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("sqlx.%s", operation))
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", operation),
		attribute.String("db.statement", query),
	)
	return ctx, span
}

// WithTracing создает новый спан для операции в транзакции
func (tx *Tx) WithTracing(ctx context.Context, operation string, query string) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("sqlx.tx.%s", operation))
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", operation),
		attribute.String("db.statement", query),
		attribute.Bool("db.transaction", true),
	)
	return ctx, span
}
