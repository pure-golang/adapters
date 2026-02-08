package devslog

import (
	"context"
	"log/slog"
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
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")
}

func TestNewDefault_InfoLevel(t *testing.T) {
	l := NewDefault(slog.LevelInfo)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)

	// Verify logger is functional
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")
}

func TestNewDefault_WarnLevel(t *testing.T) {
	l := NewDefault(slog.LevelWarn)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)

	// Verify logger is functional
	l.Warn("warn message")
	l.Error("error message")
}

func TestNewDefault_ErrorLevel(t *testing.T) {
	l := NewDefault(slog.LevelError)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)

	// Verify logger is functional
	l.Error("error message")
}

func TestNewDefault_WithAttributes(t *testing.T) {
	l := NewDefault(slog.LevelInfo)

	require.NotNil(t, l)

	// Add attributes and verify logger works
	l = l.With("service", "test", "version", "1.0.0")
	l.Info("message with attributes", "key", "value")
}

func TestNewDefault_WithGroup(t *testing.T) {
	l := NewDefault(slog.LevelInfo)

	require.NotNil(t, l)

	// Add group and verify logger works
	l = l.WithGroup("request")
	l.Info("message with group")
}

func TestNewDefault_LogAttrs(t *testing.T) {
	l := NewDefault(slog.LevelInfo)

	require.NotNil(t, l)

	// Test LogAttrs method
	l.LogAttrs(nil, slog.LevelInfo, "test message", slog.String("key", "value"))
}

func TestNewDefault_HandlerConfiguration(t *testing.T) {
	l := NewDefault(slog.LevelDebug)

	require.NotNil(t, l)

	// Verify handler is set
	h := l.Handler()
	assert.NotNil(t, h)

	// Verify enabled levels
	assert.True(t, h.Enabled(nil, slog.LevelDebug))
	assert.True(t, h.Enabled(nil, slog.LevelInfo))
	assert.True(t, h.Enabled(nil, slog.LevelWarn))
	assert.True(t, h.Enabled(nil, slog.LevelError))
}

func TestNewDefault_HandlerLevelFiltering(t *testing.T) {
	l := NewDefault(slog.LevelWarn)

	require.NotNil(t, l)

	h := l.Handler()
	assert.NotNil(t, h)

	// Debug and Info should not be enabled at Warn level
	assert.False(t, h.Enabled(nil, slog.LevelDebug))
	assert.False(t, h.Enabled(nil, slog.LevelInfo))
	assert.True(t, h.Enabled(nil, slog.LevelWarn))
	assert.True(t, h.Enabled(nil, slog.LevelError))
}

func TestNewDefault_HandlerWithOptions(t *testing.T) {
	tests := []struct {
		name        string
		level       slog.Level
		checkLevel  slog.Level
		shouldMatch bool // true if checkLevel should be enabled at level
	}{
		{"LevelDebug-Debug", slog.LevelDebug, slog.LevelDebug, true},
		{"LevelDebug-Info", slog.LevelDebug, slog.LevelInfo, true},
		{"LevelDebug-Error", slog.LevelDebug, slog.LevelError, true},
		{"LevelInfo-Debug", slog.LevelInfo, slog.LevelDebug, false},
		{"LevelInfo-Info", slog.LevelInfo, slog.LevelInfo, true},
		{"LevelInfo-Error", slog.LevelInfo, slog.LevelError, true},
		{"LevelWarn-Debug", slog.LevelWarn, slog.LevelDebug, false},
		{"LevelWarn-Info", slog.LevelWarn, slog.LevelInfo, false},
		{"LevelWarn-Warn", slog.LevelWarn, slog.LevelWarn, true},
		{"LevelWarn-Error", slog.LevelWarn, slog.LevelError, true},
		{"LevelError-Debug", slog.LevelError, slog.LevelDebug, false},
		{"LevelError-Info", slog.LevelError, slog.LevelInfo, false},
		{"LevelError-Warn", slog.LevelError, slog.LevelWarn, false},
		{"LevelError-Error", slog.LevelError, slog.LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewDefault(tt.level)
			h := l.Handler()
			result := h.Enabled(context.Background(), tt.checkLevel)

			// For levels at or above threshold, should be enabled
			// For levels below threshold, should match expected
			if tt.shouldMatch {
				assert.True(t, result)
			} else {
				assert.False(t, result)
			}
		})
	}
}

func TestNewDefault_IntegrationScenarios(t *testing.T) {
	scenarios := []struct {
		name  string
		level slog.Level
	}{
		{"ProductionLogging", slog.LevelInfo},
		{"DevelopmentLogging", slog.LevelDebug},
		{"WarningOnlyLogging", slog.LevelWarn},
		{"ErrorOnlyLogging", slog.LevelError},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			l := NewDefault(sc.level)

			// Simulate typical usage patterns
			l.LogAttrs(nil, sc.level, "test log",
				slog.String("scenario", sc.name),
				slog.Int("value", 42),
			)

			// Add context
			l = l.With(
				slog.String("component", "test"),
				slog.String("function", "TestNewDefault_IntegrationScenarios"),
			)

			l.Info("structured logging test")
		})
	}
}

func TestNewDefault_LogLevels(t *testing.T) {
	levels := []slog.Level{
		slog.Level(-8), // LevelDebug - 4
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
		slog.Level(12), // Above error
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			l := NewDefault(level)
			assert.NotNil(t, l)

			// Verify logger doesn't panic at any level
			l.Log(nil, level, "test message")
		})
	}
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

func BenchmarkLogging_Debug(b *testing.B) {
	l := NewDefault(slog.LevelDebug)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Debug("benchmark debug message")
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
		"env", "test",
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("benchmark message with attrs", "iteration", i)
	}
}
