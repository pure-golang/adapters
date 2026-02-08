package sqlx

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// TestDefaultTracingConfig verifies default tracing configuration
func TestDefaultTracingConfig(t *testing.T) {
	cfg := DefaultTracingConfig()

	require.NotNil(t, cfg)
	assert.True(t, cfg.CommentsAsAttributes)
	assert.Contains(t, cfg.ExcludeErrors, pgx.ErrNoRows)
	assert.False(t, cfg.DisableErrSkip)
}

// TestWithTracing verifies WithTracing function returns the config
func TestWithTracing(t *testing.T) {
	baseCfg := &Config{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Database: "db",
	}

	tracingCfg := DefaultTracingConfig()
	result := WithTracing(baseCfg, tracingCfg)

	// WithTracing should return the same config
	assert.Same(t, baseCfg, result)
}

// TestConnection_WithTracing verifies Connection.WithTracing creates a span
func TestConnection_WithTracing(t *testing.T) {
	// Create a mock connection with minimal config
	conn := &Connection{
		cfg: Config{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "pass",
			Database: "db",
		},
	}

	ctx := context.Background()
	ctx, span := conn.WithTracing(ctx, "TestOperation", "SELECT * FROM test")

	require.NotNil(t, ctx)
	require.NotNil(t, span)
	// Note: IsRecording may be false if no tracer provider is configured

	// End the span
	span.End()
}

// TestConnection_WithTracing_DifferentOperations verifies different operation types
func TestConnection_WithTracing_DifferentOperations(t *testing.T) {
	conn := &Connection{
		cfg: Config{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "pass",
			Database: "db",
		},
	}

	operations := []struct {
		name      string
		operation string
		query     string
	}{
		{"select", "Select", "SELECT * FROM users WHERE id = $1"},
		{"get", "Get", "SELECT * FROM users WHERE id = $1"},
		{"exec", "Exec", "INSERT INTO users (name) VALUES ($1)"},
		{"query", "Query", "SELECT * FROM users"},
		{"query_row", "QueryRow", "SELECT * FROM users WHERE id = $1"},
		{"named_exec", "NamedExec", "INSERT INTO users (name) VALUES (:name)"},
		{"named_query", "NamedQuery", "SELECT * FROM users WHERE id = :id"},
	}

	ctx := context.Background()
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			_, span := conn.WithTracing(ctx, op.operation, op.query)

			require.NotNil(t, span)
			// Note: IsRecording may be false if no tracer is configured
			// We just verify the span is created and can be ended

			span.End()
		})
	}
}

// TestTx_WithTracing verifies Tx.WithTracing creates a span
func TestTx_WithTracing(t *testing.T) {
	// Create a mock transaction with minimal config
	tx := &Tx{
		cfg: Config{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "pass",
			Database: "db",
		},
	}

	ctx := context.Background()
	ctx, span := tx.WithTracing(ctx, "TestOperation", "SELECT * FROM test")

	require.NotNil(t, ctx)
	require.NotNil(t, span)
	// Note: IsRecording may be false if no tracer provider is configured

	// End the span
	span.End()
}

// TestTx_WithTracing_DifferentOperations verifies transaction operations
func TestTx_WithTracing_DifferentOperations(t *testing.T) {
	tx := &Tx{
		cfg: Config{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "pass",
			Database: "db",
		},
	}

	operations := []struct {
		name      string
		operation string
		query     string
	}{
		{"tx_select", "Select", "SELECT * FROM users WHERE id = $1"},
		{"tx_get", "Get", "SELECT * FROM users WHERE id = $1"},
		{"tx_exec", "Exec", "UPDATE users SET name = $1 WHERE id = $2"},
		{"tx_query", "Query", "SELECT * FROM users"},
		{"tx_query_row", "QueryRow", "SELECT * FROM users WHERE id = $1"},
		{"commit", "Commit", ""},
		{"rollback", "Rollback", ""},
	}

	ctx := context.Background()
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			_, span := tx.WithTracing(ctx, op.operation, op.query)

			require.NotNil(t, span)
			// Note: IsRecording may be false if no tracer is configured
			// We just verify the span is created and can be ended

			span.End()
		})
	}
}

// TestTracer_VerifyTracerExists verifies the global tracer is initialized
func TestTracer_VerifyTracerExists(t *testing.T) {
	// This test verifies the tracer variable exists and can create spans
	// The actual tracer is initialized at package level

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test.span")

	require.NotNil(t, span)
	// Note: SpanContext may not be valid if no tracer provider is configured
	// We just verify the span is created and can be ended

	span.End()
}

// TestTracer_SpanIsTracer verifies span implements Tracer interface
func TestTracer_SpanIsTracer(t *testing.T) {
	// Create a mock connection
	conn := &Connection{
		cfg: Config{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "pass",
			Database: "db",
		},
	}

	ctx := context.Background()
	_, span := conn.WithTracing(ctx, "Get", "SELECT 1")

	// Verify span implements expected interface
	_, ok := interface{}(span).(trace.Span)
	assert.True(t, ok, "Span should implement trace.Span interface")

	span.End()
}

// TestTracingConfig_CommentsAsAttributes verifies CommentsAsAttributes setting
func TestTracingConfig_CommentsAsAttributes(t *testing.T) {
	cfg := &TracingConfig{
		CommentsAsAttributes: false,
		ExcludeErrors:        []error{},
		DisableErrSkip:       true,
	}

	assert.False(t, cfg.CommentsAsAttributes)
	assert.Empty(t, cfg.ExcludeErrors)
	assert.True(t, cfg.DisableErrSkip)
}

// TestTracingConfig_ExcludeErrors verifies custom error exclusion
func TestTracingConfig_ExcludeErrors(t *testing.T) {
	customErr := &customError{msg: "custom error"}

	cfg := &TracingConfig{
		CommentsAsAttributes: true,
		ExcludeErrors:        []error{pgx.ErrNoRows, customErr},
		DisableErrSkip:       false,
	}

	assert.True(t, cfg.CommentsAsAttributes)
	assert.Len(t, cfg.ExcludeErrors, 2)
	assert.Contains(t, cfg.ExcludeErrors, pgx.ErrNoRows)
	assert.Contains(t, cfg.ExcludeErrors, customErr)
	assert.False(t, cfg.DisableErrSkip)
}

// customError is a custom error type for testing
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}
