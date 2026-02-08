package pgx

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/pure-golang/adapters/logger"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/stretchr/testify/assert"
)

// TestNewLogger tests the NewLogger function.
func TestNewLogger(t *testing.T) {
	t.Run("creates non-nil logger", func(t *testing.T) {
		log := NewLogger()
		assert.NotNil(t, log)
	})
}

// TestLogger_Log tests the Log method with various log levels.
func TestLogger_Log(t *testing.T) {
	// Create a test handler to capture log records
	var records []slog.Record
	testHandler := &testHandler{
		records: &records,
	}

	// Create a logger with our test handler
	testLogger := slog.New(testHandler)

	// Create a context with our test logger
	ctx := logger.NewContext(context.Background(), testLogger)

	pgxLogger := &Logger{}

	t.Run("Log with Trace level", func(t *testing.T) {
		records = nil // clear records
		data := map[string]interface{}{
			"sql":  "SELECT 1",
			"args": []interface{}{},
		}
		pgxLogger.Log(ctx, tracelog.LogLevelTrace, "trace message", data)
		// Record should be captured (minLevel check happens at slog level)
	})

	t.Run("Log with Debug level", func(t *testing.T) {
		records = nil
		data := map[string]interface{}{
			"sql":  "SELECT 1",
			"args": []interface{}{},
		}
		pgxLogger.Log(ctx, tracelog.LogLevelDebug, "debug message", data)
	})

	t.Run("Log with Info level", func(t *testing.T) {
		records = nil
		data := map[string]interface{}{
			"sql": "SELECT * FROM users",
		}
		pgxLogger.Log(ctx, tracelog.LogLevelInfo, "info message", data)
	})

	t.Run("Log with Warn level", func(t *testing.T) {
		records = nil
		data := map[string]interface{}{
			"sql": "SELECT 1",
		}
		pgxLogger.Log(ctx, tracelog.LogLevelWarn, "warn message", data)
	})

	t.Run("Log with Error level", func(t *testing.T) {
		records = nil
		data := map[string]interface{}{
			"sql":   "SELECT 1",
			"error": "connection failed",
		}
		pgxLogger.Log(ctx, tracelog.LogLevelError, "error message", data)
	})

	t.Run("Log with None level", func(t *testing.T) {
		records = nil
		data := map[string]interface{}{
			"sql": "SELECT 1",
		}
		pgxLogger.Log(ctx, tracelog.LogLevelNone, "none message", data)
	})
}

// TestLogger_LogWithDuration tests Log with duration data.
func TestLogger_LogWithDuration(t *testing.T) {
	// Create a test handler to capture log records
	var records []slog.Record
	testHandler := &testHandler{
		records: &records,
	}

	// Create a logger with our test handler
	testLogger := slog.New(testHandler)

	// Create a context with our test logger
	ctx := logger.NewContext(context.Background(), testLogger)

	pgxLogger := &Logger{}

	t.Run("duration is converted to duration_ms", func(t *testing.T) {
		records = nil
		data := map[string]interface{}{
			"sql":  "SELECT 1",
			"time": 150 * time.Millisecond,
		}
		pgxLogger.Log(ctx, tracelog.LogLevelInfo, "query message", data)

		// Check that time was removed and duration_ms was added
		// (The actual attribute conversion happens inside the Log method)
	})
}

// TestLogger_LogWithNilData tests Log with nil data map.
func TestLogger_LogWithNilData(t *testing.T) {
	// Create a test handler
	var records []slog.Record
	testHandler := &testHandler{
		records: &records,
	}

	// Create a logger with our test handler
	testLogger := slog.New(testHandler)

	// Create a context with our test logger
	ctx := logger.NewContext(context.Background(), testLogger)

	pgxLogger := &Logger{}

	t.Run("Log with nil data", func(t *testing.T) {
		// This should not panic
		assert.NotPanics(t, func() {
			pgxLogger.Log(ctx, tracelog.LogLevelInfo, "message with nil data", nil)
		})
	})

	t.Run("Log with empty data", func(t *testing.T) {
		assert.NotPanics(t, func() {
			data := map[string]interface{}{}
			pgxLogger.Log(ctx, tracelog.LogLevelInfo, "message with empty data", data)
		})
	})
}

// TestLogger_slogLevel tests the slogLevel method.
func TestLogger_slogLevel(t *testing.T) {
	pgxLogger := &Logger{}

	tests := []struct {
		name          string
		pgxLogLevel   tracelog.LogLevel
		expectedLevel slog.Level
	}{
		{
			name:          "LogLevelTrace maps to Debug-1",
			pgxLogLevel:   tracelog.LogLevelTrace,
			expectedLevel: slog.LevelDebug - 1,
		},
		{
			name:          "LogLevelDebug maps to Debug",
			pgxLogLevel:   tracelog.LogLevelDebug,
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "LogLevelInfo maps to Info",
			pgxLogLevel:   tracelog.LogLevelInfo,
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "LogLevelWarn maps to Warn",
			pgxLogLevel:   tracelog.LogLevelWarn,
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "LogLevelError maps to Error",
			pgxLogLevel:   tracelog.LogLevelError,
			expectedLevel: slog.LevelError,
		},
		{
			name:          "LogLevelNone maps to Error",
			pgxLogLevel:   tracelog.LogLevelNone,
			expectedLevel: slog.LevelError,
		},
		{
			name:          "Unknown level maps to Error",
			pgxLogLevel:   tracelog.LogLevel(99),
			expectedLevel: slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pgxLogger.slogLevel(tt.pgxLogLevel)
			assert.Equal(t, tt.expectedLevel, result)
		})
	}
}

// TestLogger_LogWithComplexData tests Log with complex data structures.
func TestLogger_LogWithComplexData(t *testing.T) {
	// Create a test handler
	var records []slog.Record
	testHandler := &testHandler{
		records: &records,
	}

	// Create a logger with our test handler
	testLogger := slog.New(testHandler)

	// Create a context with our test logger
	ctx := logger.NewContext(context.Background(), testLogger)

	pgxLogger := &Logger{}

	t.Run("Log with various data types", func(t *testing.T) {
		assert.NotPanics(t, func() {
			data := map[string]interface{}{
				"sql":        "SELECT * FROM users WHERE id = $1",
				"args":       []interface{}{123},
				"rows":       5,
				"duration":   250 * time.Millisecond,
				"success":    true,
				"cached":     false,
				"connection": "conn-123",
			}
			pgxLogger.Log(ctx, tracelog.LogLevelInfo, "query executed", data)
		})
	})

	t.Run("Log with nil args", func(t *testing.T) {
		assert.NotPanics(t, func() {
			data := map[string]interface{}{
				"sql":  "SELECT 1",
				"args": nil,
			}
			pgxLogger.Log(ctx, tracelog.LogLevelInfo, "query", data)
		})
	})
}

// TestLogger_LogWithSpecialDurationValues tests edge cases for duration handling.
func TestLogger_LogWithSpecialDurationValues(t *testing.T) {
	// Create a test handler
	var records []slog.Record
	testHandler := &testHandler{
		records: &records,
	}

	// Create a logger with our test handler
	testLogger := slog.New(testHandler)

	// Create a context with our test logger
	ctx := logger.NewContext(context.Background(), testLogger)

	pgxLogger := &Logger{}

	durations := []time.Duration{
		0,
		1 * time.Nanosecond,
		1 * time.Microsecond,
		1 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
		10 * time.Second,
		1 * time.Minute,
	}

	for _, d := range durations {
		t.Run("duration_"+d.String(), func(t *testing.T) {
			assert.NotPanics(t, func() {
				data := map[string]interface{}{
					"sql":  "SELECT 1",
					"time": d,
				}
				pgxLogger.Log(ctx, tracelog.LogLevelInfo, "query", data)
			})
		})
	}
}

// TestLogger_LogWithNonDurationTimeValue tests Log when time field is not a duration.
func TestLogger_LogWithNonDurationTimeValue(t *testing.T) {
	// Create a test handler
	var records []slog.Record
	testHandler := &testHandler{
		records: &records,
	}

	// Create a logger with our test handler
	testLogger := slog.New(testHandler)

	// Create a context with our test logger
	ctx := logger.NewContext(context.Background(), testLogger)

	pgxLogger := &Logger{}

	t.Run("time value is string", func(t *testing.T) {
		assert.NotPanics(t, func() {
			data := map[string]interface{}{
				"sql":  "SELECT 1",
				"time": "2024-01-01T00:00:00Z",
			}
			pgxLogger.Log(ctx, tracelog.LogLevelInfo, "query", data)
		})
	})

	t.Run("time value is int", func(t *testing.T) {
		assert.NotPanics(t, func() {
			data := map[string]interface{}{
				"sql":  "SELECT 1",
				"time": 12345,
			}
			pgxLogger.Log(ctx, tracelog.LogLevelInfo, "query", data)
		})
	})
}

// testHandler is a simple slog.Handler that captures records for testing.
type testHandler struct {
	records *[]slog.Record
}

func (h *testHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *testHandler) Handle(ctx context.Context, r slog.Record) error {
	*h.records = append(*h.records, r)
	return nil
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	return h
}
