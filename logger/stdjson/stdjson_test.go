package stdjson

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefault_DebugLevel(t *testing.T) {
	l := NewDefault(slog.LevelDebug)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)

	// Verify logger is functional
	l.Debug("debug message")
}

func TestNewDefault_InfoLevel(t *testing.T) {
	l := NewDefault(slog.LevelInfo)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)

	l.Info("info message")
}

func TestNewDefault_WarnLevel(t *testing.T) {
	l := NewDefault(slog.LevelWarn)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)

	l.Warn("warn message")
}

func TestNewDefault_ErrorLevel(t *testing.T) {
	l := NewDefault(slog.LevelError)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)

	l.Error("error message")
}

func TestNewDefault_WithCustomOutput(t *testing.T) {
	// Capture stdout to verify JSON output
	var buf bytes.Buffer

	// Create a custom logger with buffer output
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	require.NotNil(t, l)

	l.Info("test message", "key", "value")

	// Verify output is JSON
	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "test message", result["msg"])
	assert.Equal(t, "value", result["key"])
}

func TestNewDefault_HandlerConfiguration(t *testing.T) {
	l := NewDefault(slog.LevelDebug)

	require.NotNil(t, l)

	h := l.Handler()
	assert.NotNil(t, h)

	// Verify levels are enabled correctly
	assert.True(t, h.Enabled(nil, slog.LevelDebug))
	assert.True(t, h.Enabled(nil, slog.LevelInfo))
	assert.True(t, h.Enabled(nil, slog.LevelWarn))
	assert.True(t, h.Enabled(nil, slog.LevelError))
}

func TestNewDefault_LevelFiltering(t *testing.T) {
	l := NewDefault(slog.LevelWarn)
	h := l.Handler()

	assert.NotNil(t, h)

	// At Warn level, Debug and Info should not be enabled
	assert.False(t, h.Enabled(nil, slog.LevelDebug))
	assert.False(t, h.Enabled(nil, slog.LevelInfo))
	assert.True(t, h.Enabled(nil, slog.LevelWarn))
	assert.True(t, h.Enabled(nil, slog.LevelError))
}

func TestNewDefault_JSONOutput(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	l := NewDefault(slog.LevelInfo)

	l.Info("json test", "service", "test", "version", 1)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON
	var result map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	assert.NoError(t, err)
	assert.Contains(t, strings.ToLower(output), "json test")
}

func TestNewDefault_WithAttributes(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	l = l.With("component", "test", "env", "unit")
	l.Info("message with attrs")

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "test", result["component"])
	assert.Equal(t, "unit", result["env"])
	assert.Equal(t, "message with attrs", result["msg"])
}

func TestNewDefault_WithGroup(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	l = l.WithGroup("request")
	l.Info("message with group", "id", "123")

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Should have request group with id
	assert.Contains(t, result, "request")
}

func TestNewDefault_LogAttrs(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	l.LogAttrs(nil, slog.LevelInfo, "log attrs test",
		slog.String("string_key", "string_value"),
		slog.Int("int_key", 42),
		slog.Bool("bool_key", true),
	)

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "log attrs test", result["msg"])
	assert.Equal(t, "string_value", result["string_key"])
	assert.Equal(t, float64(42), result["int_key"])
	assert.Equal(t, true, result["bool_key"])
}

func TestNewDefault_AllLevels(t *testing.T) {
	levels := []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			l := NewDefault(level)
			assert.NotNil(t, l)
		})
	}
}

func TestNewDefault_ContextSupport(t *testing.T) {
	l := NewDefault(slog.LevelInfo)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", "abc123")

	l.LogAttrs(ctx, slog.LevelInfo, "context test", slog.String("key", "value"))
	// Should not panic
}

func TestNewDefault_NilContext(t *testing.T) {
	l := NewDefault(slog.LevelInfo)

	l.LogAttrs(nil, slog.LevelInfo, "nil context test")
	// Should not panic
}

func TestNewDefault_ErrorHandling(t *testing.T) {
	l := NewDefault(slog.LevelError)

	l.Error("error message", "error_code", 500, "details", "internal error")
	// Should not panic
}

func TestNewDefault_StructuredLogging(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	user := User{Name: "John Doe", Email: "john@example.com"}

	l.Info("user created", "user", user)

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// User object should be serialized
	assert.Contains(t, result, "user")
}

func TestNewDefault_EmptyMessage(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	l.Info("")
	l.Info("", "key", "value")

	// Should not panic, output may have empty or missing msg
}

func TestNewDefault_SpecialCharacters(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	l.Info("message with \"quotes\"", "key", "value with\nnewline")

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// JSON should properly escape special characters
	assert.Contains(t, result["msg"], "quotes")
}

func TestNewDefault_HigherLevel(t *testing.T) {
	l := NewDefault(slog.Level(100)) // Very high level

	// Should create logger without error
	assert.NotNil(t, l)

	// Lower level messages should still be possible to log
	// (they just won't be output by the handler)
	l.Info("info at high threshold")
}

func TestNewDefault_LowerLevel(t *testing.T) {
	l := NewDefault(slog.Level(-100)) // Very low level

	assert.NotNil(t, l)

	l.Log(nil, slog.Level(-50), "very verbose message")
	// Should not panic
}

func TestNewDefault_MultipleAttributes(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	l.Info("multi attr message",
		"key1", "value1",
		"key2", 42,
		"key3", true,
		"key4", 3.14,
		"key5", []string{"a", "b", "c"},
	)

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, float64(42), result["key2"])
	assert.Equal(t, true, result["key3"])
	assert.Equal(t, 3.14, result["key4"])
}

func TestNewDefault_ConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	done := make(chan bool)

	// Concurrent logging
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 10; j++ {
				l.Info("concurrent", "worker", n, "iteration", j)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Buffer should have content
	assert.Greater(t, buf.Len(), 0)
}

func TestNewDefault_NestedGroups(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	l = l.WithGroup("outer").WithGroup("inner")
	l.Info("nested message", "key", "value")

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Should have nested groups
	assert.Contains(t, result, "outer")
}

func TestNewDefault_TimeField(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	l.Info("time test")

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// JSON handler should include time field
	assert.Contains(t, result, "time")
}

func TestNewDefault_LevelField(t *testing.T) {
	var buf bytes.Buffer

	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	l.Info("level test")

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// JSON handler should include level field
	assert.Contains(t, result, "level")
}

func TestNewDefault_CustomLevels(t *testing.T) {
	// Slog levels: Debug=-4, Info=0, Warn=4, Error=8
	customLevel := slog.Level(2) // Between Info (0) and Warn (4)

	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: customLevel}))

	assert.NotNil(t, l)

	// At level 2, Info (0) should not be enabled (0 < 2)
	// Warn (4) should be enabled (4 >= 2)
	// Error (8) should be enabled (8 >= 2)
	h := l.Handler()
	assert.False(t, h.Enabled(nil, slog.LevelInfo))
	assert.True(t, h.Enabled(nil, slog.LevelWarn))
	assert.True(t, h.Enabled(nil, slog.LevelError))
}

func TestNewDefault_Reusable(t *testing.T) {
	l := NewDefault(slog.LevelInfo)

	// Create multiple loggers
	l1 := l.With("component", "comp1")
	l2 := l.With("component", "comp2")

	assert.NotNil(t, l1)
	assert.NotNil(t, l2)

	l1.Info("from component 1")
	l2.Info("from component 2")
	l.Info("from base")
}

func BenchmarkNewDefault_Debug(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewDefault(slog.LevelDebug)
	}
}

func BenchmarkNewDefault_Info(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewDefault(slog.LevelInfo)
	}
}

func BenchmarkLogging_Info(b *testing.B) {
	l := NewDefault(slog.LevelInfo)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("benchmark info message")
	}
}

func BenchmarkLogging_WithAttrs(b *testing.B) {
	l := NewDefault(slog.LevelInfo).With(
		"service", "benchmark",
		"version", "1.0",
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("benchmark message", "iteration", i)
	}
}

func BenchmarkLogging_LogAttrs(b *testing.B) {
	l := NewDefault(slog.LevelInfo)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.LogAttrs(nil, slog.LevelInfo, "benchmark",
			slog.String("service", "test"),
			slog.Int("iteration", i),
		)
	}
}
