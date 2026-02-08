package noop

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoop_ReturnsLogger(t *testing.T) {
	l := NewNoop()

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)
}

func TestNewNoop_HandlerNotNil(t *testing.T) {
	l := NewNoop()

	require.NotNil(t, l)

	h := l.Handler()
	assert.NotNil(t, h)
}

func TestNewNoop_AllLogMethods(t *testing.T) {
	l := NewNoop()

	// All these should not panic
	l.Debug("debug message")
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")

	l.Log(nil, slog.LevelInfo, "custom level message")
}

func TestNewNoop_WithAttributes(t *testing.T) {
	l := NewNoop()

	l = l.With("key1", "value1", "key2", 42)
	l = l.WithGroup("group")

	// Should not panic
	l.Info("message with attributes")
}

func TestNewNoop_LogAttrs(t *testing.T) {
	l := NewNoop()

	// Should not panic with various attribute types
	l.LogAttrs(nil, slog.LevelInfo, "test",
		slog.String("string", "value"),
		slog.Int("int", 42),
		slog.Bool("bool", true),
		slog.Float64("float", 3.14),
		slog.Any("any", map[string]string{"key": "value"}),
	)
}

func TestNewNoop_HandlerEnabled(t *testing.T) {
	l := NewNoop()
	h := l.Handler()

	// No-op handler should not be enabled for any level
	// or it may be enabled but not output
	_ = h.Enabled(nil, slog.LevelInfo)
	// The result depends on the underlying JSON handler configuration
	// Just verify it doesn't panic
	assert.NotNil(t, h)
}

func TestNewNoop_HandlerHandle(t *testing.T) {
	l := NewNoop()
	h := l.Handler()

	// Should not panic when handling records
	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "test", 0)
	err := h.Handle(nil, r)

	assert.NoError(t, err)
}

func TestNewNoop_WithHandlerOptions(t *testing.T) {
	// The noop logger uses NewJSONHandler with nil options
	l := NewNoop()

	assert.NotNil(t, l)

	// Should work with all log levels
	l.Log(nil, slog.Level(-10), "very low level")
	l.Log(nil, slog.LevelDebug, "debug")
	l.Log(nil, slog.LevelInfo, "info")
	l.Log(nil, slog.LevelWarn, "warn")
	l.Log(nil, slog.LevelError, "error")
	l.Log(nil, slog.Level(100), "very high level")
}

func TestWriter_Write(t *testing.T) {
	w := &writer{}

	// Write should always succeed
	n, err := w.Write([]byte("test data"))

	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestWriter_WriteEmpty(t *testing.T) {
	w := &writer{}

	n, err := w.Write([]byte{})

	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestWriter_WriteNil(t *testing.T) {
	w := &writer{}

	n, err := w.Write(nil)

	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestWriter_WriteLargeData(t *testing.T) {
	w := &writer{}

	largeData := make([]byte, 1024*1024) // 1MB

	n, err := w.Write(largeData)

	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestWriter_WriteRepeated(t *testing.T) {
	w := &writer{}

	// Multiple writes should all succeed
	for i := 0; i < 100; i++ {
		n, err := w.Write([]byte("test"))
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	}
}

func TestNewNoop_Concurrent(t *testing.T) {
	l := NewNoop()

	// Concurrent logging should not panic
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				l.Info("concurrent test", "iteration", j)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestNewNoop_NilContext(t *testing.T) {
	l := NewNoop()

	// Should handle nil context
	l.LogAttrs(nil, slog.LevelInfo, "nil context test")

	// Should handle context with values
	ctx := context.WithValue(context.Background(), "key", "value")
	l.LogAttrs(ctx, slog.LevelInfo, "with context")
}

func TestNewNoop_WithNilAttributes(t *testing.T) {
	l := NewNoop()

	// Test with valid attributes only - slog.With panics with nil
	l.With("key", "value").Info("message")
}

func TestWriter_ImplementsHandler(t *testing.T) {
	w := &writer{}

	// writer should implement slog.Handler interface (via embedding)
	var _ slog.Handler = w
	_ = w // Use the variable

	// Writer also implements io.Writer via the Write method
	var _ interface{ Write([]byte) (int, error) } = w
	_ = w // Use the variable

	// Verify writer is not nil
	assert.NotNil(t, w)
}

func TestNewNoop_Reusable(t *testing.T) {
	l := NewNoop()

	// Create multiple loggers from the same base
	l1 := l.With("component", "component1")
	l2 := l.With("component", "component2")

	// Both should work
	l1.Info("message from component1")
	l2.Info("message from component2")

	// Original should still work
	l.Info("message from original")
}

func TestNewNoop_LogLevels(t *testing.T) {
	l := NewNoop()

	levels := []slog.Level{
		slog.Level(-100), // Very low
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
		slog.Level(100), // Very high
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			l.Log(nil, level, "test at level")
		})
	}
}

func TestNewNoop_Formatting(t *testing.T) {
	l := NewNoop()

	// Test various message formats
	l.Info("simple message")
	l.Info("message with numbers", "count", 42, "ratio", 3.14)
	l.Info("message with bool", "active", true, "valid", false)
}

func TestNewNoop_NestedGroups(t *testing.T) {
	l := NewNoop()

	// Create nested groups
	l = l.WithGroup("outer").WithGroup("inner").With("key", "value")
	l.Info("nested group message")
}

func TestNewNoop_EmptyMessage(t *testing.T) {
	l := NewNoop()

	l.Info("")
	l.Debug("")
	l.Warn("")
	l.Error("")
}

func BenchmarkNewNoop(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewNoop()
	}
}

func BenchmarkNoopLogging(b *testing.B) {
	l := NewNoop()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("benchmark message", "iteration", i)
	}
}

func BenchmarkWriter_Write(b *testing.B) {
	w := &writer{}
	data := []byte("test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Write(data)
	}
}

func BenchmarkNoop_WithAttrs(b *testing.B) {
	l := NewNoop().With("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("message")
	}
}
