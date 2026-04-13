package interceptor

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/pure-golang/adapters/logger"
)

// TestMain устанавливает единый записывающий TracerProvider для всего пакета.
// Package-level var tracer делегирует в него после первого SetTracerProvider.
// Повторные вызовы SetTracerProvider не обновляют уже делегированный tracer,
// поэтому тесты со span-ассертами не помечаются t.Parallel() и используют
// sharedExp.reset() для изоляции.
func TestMain(m *testing.M) {
	sharedExp = &syncSpanExporter{}
	sharedTP = sdktrace.NewTracerProvider(sdktrace.WithSyncer(sharedExp))
	otel.SetTracerProvider(sharedTP)

	code := m.Run()

	if err := sharedTP.Shutdown(context.Background()); err != nil {
		log.Printf("tracer provider shutdown: %v", err)
	}
	os.Exit(code)
}

// syncSpanExporter — потокобезопасный экспортер spans для тестов.
type syncSpanExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func (e *syncSpanExporter) reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = nil
}

func (e *syncSpanExporter) getSpans() []sdktrace.ReadOnlySpan {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]sdktrace.ReadOnlySpan, len(e.spans))
	copy(result, e.spans)
	return result
}

func (e *syncSpanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *syncSpanExporter) Shutdown(_ context.Context) error { return nil }

var (
	sharedExp *syncSpanExporter
	sharedTP  *sdktrace.TracerProvider
)

// attrValue ищет строковый атрибут span по ключу.
func attrValue(span sdktrace.ReadOnlySpan, key string) (string, bool) {
	for _, kv := range span.Attributes() {
		if string(kv.Key) == key {
			return kv.Value.AsString(), true
		}
	}
	return "", false
}

// makeCtx формирует контекст с заданным типом и именем GraphQL-операции.
func makeCtx(operation ast.Operation, name string) context.Context {
	opCtx := &graphql.OperationContext{
		Operation: &ast.OperationDefinition{
			Operation: operation,
			Name:      name,
		},
	}
	return graphql.WithOperationContext(context.Background(), opCtx)
}

// okNext возвращает OperationHandler, отдающий пустой Response без ошибок.
func okNext() graphql.OperationHandler {
	return func(ctx context.Context) graphql.ResponseHandler {
		return func(ctx context.Context) *graphql.Response {
			return &graphql.Response{}
		}
	}
}

// errNext возвращает OperationHandler, отдающий Response с заданным сообщением ошибки.
func errNext(msg string) graphql.OperationHandler {
	return func(ctx context.Context) graphql.ResponseHandler {
		return func(ctx context.Context) *graphql.Response {
			return &graphql.Response{
				Errors: gqlerror.List{{Message: msg}},
			}
		}
	}
}

// captureNext возвращает OperationHandler, записывающий ctx в переданный указатель.
func captureNext(out *context.Context) graphql.OperationHandler {
	return func(ctx context.Context) graphql.ResponseHandler {
		*out = ctx
		return func(ctx context.Context) *graphql.Response { return &graphql.Response{} }
	}
}

// TestInterceptOperation_query проверяет span name для операции query с именем.
func TestInterceptOperation_query(t *testing.T) {
	// Arrange.
	sharedExp.reset()
	ctx := makeCtx(ast.Query, "GetUser")
	ext := NewObservabilityExtension(nil)

	// Act.
	resp := ext.InterceptOperation(ctx, okNext())(ctx)

	// Assert.
	require.NotNil(t, resp)
	spans := sharedExp.getSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "query GetUser", spans[0].Name())
}

// TestInterceptOperation_mutation проверяет span name для операции mutation.
func TestInterceptOperation_mutation(t *testing.T) {
	// Arrange.
	sharedExp.reset()
	ctx := makeCtx(ast.Mutation, "CreateRoom")
	ext := NewObservabilityExtension(nil)

	// Act.
	resp := ext.InterceptOperation(ctx, okNext())(ctx)

	// Assert.
	require.NotNil(t, resp)
	spans := sharedExp.getSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "mutation CreateRoom", spans[0].Name())
}

// TestInterceptOperation_anonymous_operation проверяет span name когда имя операции пустое.
func TestInterceptOperation_anonymous_operation(t *testing.T) {
	// Arrange.
	sharedExp.reset()
	ctx := makeCtx(ast.Query, "")
	ext := NewObservabilityExtension(nil)

	// Act.
	resp := ext.InterceptOperation(ctx, okNext())(ctx)

	// Assert.
	require.NotNil(t, resp)
	spans := sharedExp.getSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "query", spans[0].Name())
}

// TestInterceptOperation_success проверяет, что span получает статус Ok при успешном ответе.
func TestInterceptOperation_success(t *testing.T) {
	// Arrange.
	sharedExp.reset()
	ctx := makeCtx(ast.Query, "GetUser")
	ext := NewObservabilityExtension(nil)

	// Act.
	resp := ext.InterceptOperation(ctx, okNext())(ctx)

	// Assert.
	require.NotNil(t, resp)
	spans := sharedExp.getSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Ok, spans[0].Status().Code)
}

// TestInterceptOperation_graphql_error проверяет, что span получает статус Error при наличии ошибок.
func TestInterceptOperation_graphql_error(t *testing.T) {
	// Arrange.
	sharedExp.reset()
	ctx := makeCtx(ast.Query, "GetUser")
	ext := NewObservabilityExtension(nil)

	// Act.
	resp := ext.InterceptOperation(ctx, errNext("not found"))(ctx)

	// Assert.
	require.NotNil(t, resp)
	spans := sharedExp.getSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status().Code)
	assert.Equal(t, "not found", spans[0].Status().Description)

	val, ok := attrValue(spans[0], "graphql.errors")
	assert.True(t, ok)
	assert.Contains(t, val, "not found")
}

// TestInterceptOperation_logger_enriched_with_operation проверяет, что ctx содержит logger с graphql.operation.
func TestInterceptOperation_logger_enriched_with_operation(t *testing.T) {
	t.Parallel()

	// Arrange.
	var buf bytes.Buffer
	baseLog := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := makeCtx(ast.Query, "GetUser")
	ctx = logger.NewContext(ctx, baseLog)

	var capturedCtx context.Context
	ext := NewObservabilityExtension(nil)

	// Act.
	resp := ext.InterceptOperation(ctx, captureNext(&capturedCtx))(ctx)
	require.NotNil(t, resp)

	// Assert.
	logger.FromContext(capturedCtx).Info("test")
	assert.Contains(t, buf.String(), "graphql.operation")
	assert.Contains(t, buf.String(), "query GetUser")
}

// TestInterceptOperation_logger_enriched_with_user_id проверяет добавление user_id при успешной аутентификации.
func TestInterceptOperation_logger_enriched_with_user_id(t *testing.T) {
	t.Parallel()

	// Arrange.
	var buf bytes.Buffer
	baseLog := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := makeCtx(ast.Mutation, "CreateRoom")
	ctx = logger.NewContext(ctx, baseLog)

	extractor := UserIDExtractor(func(_ context.Context) (int, bool) { return 42, true })
	ext := NewObservabilityExtension(extractor)
	var capturedCtx context.Context

	// Act.
	resp := ext.InterceptOperation(ctx, captureNext(&capturedCtx))(ctx)
	require.NotNil(t, resp)

	// Assert.
	logger.FromContext(capturedCtx).Info("test")
	assert.Contains(t, buf.String(), "user_id")
	assert.Contains(t, buf.String(), fmt.Sprint(42))
}

// TestInterceptOperation_nil_extractor проверяет, что nil extractor не паникует и не добавляет user_id.
func TestInterceptOperation_nil_extractor(t *testing.T) {
	t.Parallel()

	// Arrange.
	var buf bytes.Buffer
	baseLog := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := makeCtx(ast.Query, "GetUser")
	ctx = logger.NewContext(ctx, baseLog)

	ext := NewObservabilityExtension(nil)
	var capturedCtx context.Context

	// Act + Assert.
	assert.NotPanics(t, func() {
		resp := ext.InterceptOperation(ctx, captureNext(&capturedCtx))(ctx)
		require.NotNil(t, resp)
	})

	logger.FromContext(capturedCtx).Info("test")
	assert.NotContains(t, buf.String(), "user_id")
}

// TestInterceptOperation_extractor_false проверяет, что user_id не добавляется когда extractor возвращает false.
func TestInterceptOperation_extractor_false(t *testing.T) {
	t.Parallel()

	// Arrange.
	var buf bytes.Buffer
	baseLog := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := makeCtx(ast.Query, "GetUser")
	ctx = logger.NewContext(ctx, baseLog)

	extractor := UserIDExtractor(func(_ context.Context) (int, bool) { return 0, false })
	ext := NewObservabilityExtension(extractor)
	var capturedCtx context.Context

	// Act.
	resp := ext.InterceptOperation(ctx, captureNext(&capturedCtx))(ctx)
	require.NotNil(t, resp)

	// Assert.
	logger.FromContext(capturedCtx).Info("test")
	assert.NotContains(t, buf.String(), "user_id")
}

// TestInterceptOperation_metrics_no_panic проверяет, что метрики записываются без паники.
func TestInterceptOperation_metrics_no_panic(t *testing.T) {
	t.Parallel()

	// Arrange.
	ctx := makeCtx(ast.Query, "GetUser")
	ext := NewObservabilityExtension(nil)

	// Act + Assert.
	assert.NotPanics(t, func() {
		resp := ext.InterceptOperation(ctx, okNext())(ctx)
		require.NotNil(t, resp)
	})
}

// TestInterceptOperation_http_span_enriched проверяет обогащение HTTP span атрибутами graphql.operation_type и graphql.operation_name.
func TestInterceptOperation_http_span_enriched(t *testing.T) {
	// Arrange.
	sharedExp.reset()

	// Имитируем HTTP span, который Monitoring кладёт в контекст.
	httpCtx, httpSpan := sharedTP.Tracer("http").Start(context.Background(), "/graphql",
		trace.WithSpanKind(trace.SpanKindServer))

	ctx := graphql.WithOperationContext(httpCtx, &graphql.OperationContext{
		Operation: &ast.OperationDefinition{
			Operation: ast.Mutation,
			Name:      "CreateRoom",
		},
	})
	ext := NewObservabilityExtension(nil)

	// Act.
	resp := ext.InterceptOperation(ctx, okNext())(ctx)
	require.NotNil(t, resp)
	httpSpan.End()

	// Assert.
	spans := sharedExp.getSpans()
	require.Len(t, spans, 2) // child (graphql) + parent (http).

	var httpRecorded sdktrace.ReadOnlySpan
	for _, s := range spans {
		if s.Name() == "/graphql" {
			httpRecorded = s
		}
	}
	require.NotNil(t, httpRecorded, "HTTP span not found")

	opType, ok := attrValue(httpRecorded, "graphql.operation_type")
	assert.True(t, ok)
	assert.Equal(t, "mutation", opType)

	opName, ok := attrValue(httpRecorded, "graphql.operation_name")
	assert.True(t, ok)
	assert.Equal(t, "CreateRoom", opName)
}
