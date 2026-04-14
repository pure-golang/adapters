package interceptor

import (
	"context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/pure-golang/adapters/logger"
)

// UserIDExtractor извлекает идентификатор пользователя из контекста.
// Возвращает (id, true) если пользователь аутентифицирован, (0, false) иначе.
// Если nil — userID не добавляется в логгер.
type UserIDExtractor func(ctx context.Context) (userID int, ok bool)

var (
	tracer = otel.Tracer("github.com/pure-golang/adapters/graphql/interceptor")
	meter  = otel.GetMeterProvider().Meter("github.com/pure-golang/adapters/graphql/interceptor")

	//nolint:errcheck // Синхронные инструменты OpenTelemetry никогда не возвращают ошибки.
	graphqlRequestsCount, _ = meter.Int64Counter("graphql.request_count")
	//nolint:errcheck // Синхронные инструменты OpenTelemetry никогда не возвращают ошибки.
	graphqlRequestTimeHist, _ = meter.Int64Histogram("graphql.request_time", metric.WithUnit("ms"))
)

// ObservabilityExtension добавляет трейсинг, метрики и логирование для GraphQL-операций.
type ObservabilityExtension struct {
	extractor UserIDExtractor
}

// NewObservabilityExtension создаёт новый ObservabilityExtension с заданным экстрактором пользователя.
func NewObservabilityExtension(extractor UserIDExtractor) *ObservabilityExtension {
	return &ObservabilityExtension{extractor: extractor}
}

// ExtensionName возвращает имя расширения.
func (e *ObservabilityExtension) ExtensionName() string { return "Observability" }

// Validate проверяет корректность конфигурации расширения.
func (e *ObservabilityExtension) Validate(_ graphql.ExecutableSchema) error { return nil }

// InterceptOperation добавляет трейсинг, логирование и метрики к GraphQL-операции.
func (e *ObservabilityExtension) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	opCtx := graphql.GetOperationContext(ctx)
	operationType := string(opCtx.Operation.Operation)
	operationName := opCtx.Operation.Name

	spanName := operationType
	if operationName != "" {
		spanName = operationType + " " + operationName
	}

	// Обогащаем HTTP span атрибутами операции — он уже лежит в контексте от Monitoring.
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.String("graphql.operation_type", operationType),
		attribute.String("graphql.operation_name", operationName),
	)

	// Обогащаем logger от Monitoring — все последующие логи автоматически получат эти поля.
	log := logger.FromContext(ctx).With("graphql.operation", spanName)
	if e.extractor != nil {
		if id, ok := e.extractor(ctx); ok {
			log = log.With("user_id", id)
		}
	}
	ctx = logger.NewContext(ctx, log)

	// Запускаем child span для GraphQL execution.
	reqTime := time.Now()
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))

	metricLabels := []attribute.KeyValue{
		attribute.String("graphql.operation_type", operationType),
		attribute.String("graphql.operation_name", operationName),
	}

	responseHandler := next(ctx)

	return func(ctx context.Context) *graphql.Response {
		resp := responseHandler(ctx)

		if resp == nil {
			span.SetStatus(codes.Ok, "")
			span.End()
			return nil
		}

		if len(resp.Errors) > 0 {
			span.SetStatus(codes.Error, resp.Errors[0].Message)
			span.SetAttributes(attribute.String("graphql.errors", fmt.Sprint(resp.Errors)))
			// Квалифицируем ошибку и на HTTP span.
			trace.SpanFromContext(ctx).SetAttributes(
				attribute.String("graphql.errors", resp.Errors[0].Message),
			)
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()

		graphqlRequestsCount.Add(ctx, 1, metric.WithAttributes(append(metricLabels,
			attribute.Bool("graphql.has_errors", len(resp.Errors) > 0))...))
		graphqlRequestTimeHist.Record(ctx, time.Since(reqTime).Milliseconds(),
			metric.WithAttributes(metricLabels...))

		return resp
	}
}
