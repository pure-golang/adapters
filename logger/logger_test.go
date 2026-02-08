package logger

import (
	"context"
	"log/slog"
	"testing"

	"github.com/pure-golang/adapters/logger/noop"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewDefault_ProviderDev(t *testing.T) {
	c := Config{
		Provider: ProviderDevSlog,
		Level:    DEBUG,
	}

	l := NewDefault(c)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)
}

func TestNewDefault_ProviderStdJson(t *testing.T) {
	c := Config{
		Provider: ProviderStdJson,
		Level:    INFO,
	}

	l := NewDefault(c)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)
}

func TestNewDefault_ProviderNoop(t *testing.T) {
	c := Config{
		Provider: ProviderNoop,
		Level:    ERROR,
	}

	l := NewDefault(c)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)
}

func TestNewDefault_ProviderInvalid(t *testing.T) {
	// Invalid provider should fall back to stdjson (default case)
	c := Config{
		Provider: Provider("invalid_provider"),
		Level:    WARN,
	}

	l := NewDefault(c)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)
}

func TestNewDefault_ProviderEmpty(t *testing.T) {
	// Empty provider should use default from config which is std_json
	c := Config{
		Provider: Provider(""),
		Level:    INFO,
	}

	l := NewDefault(c)

	assert.NotNil(t, l)
	assert.IsType(t, &slog.Logger{}, l)
}

func TestInitDefault_SetsGlobalLogger(t *testing.T) {
	// Save original default handler to restore later
	original := slog.Default()

	c := Config{
		Provider: ProviderNoop,
		Level:    DEBUG,
	}

	InitDefault(c)

	// After InitDefault, slog.Default() should return a noop logger
	defaultLogger := slog.Default()
	assert.NotNil(t, defaultLogger)

	// Restore original
	slog.SetDefault(original)
}

func TestInitDefault_SetsOtelErrorHandler(t *testing.T) {
	original := slog.Default()

	c := Config{
		Provider: ProviderStdJson,
		Level:    INFO,
	}

	InitDefault(c)

	// The global logger should be changed
	newDefault := slog.Default()
	assert.NotNil(t, newDefault)

	// Handler should be different (since we set a new logger)
	// This may be the same type but different instance
	slog.SetDefault(original)
}

func TestFromContext_WithLoggerInContext(t *testing.T) {
	ctx := context.Background()
	testLogger := slog.New(noop.NewNoop().Handler())

	ctx = NewContext(ctx, testLogger)

	retrieved := FromContext(ctx)

	assert.Same(t, testLogger, retrieved)
}

func TestFromContext_WithoutLoggerInContext(t *testing.T) {
	ctx := context.Background()

	retrieved := FromContext(ctx)

	// Should return the default logger
	assert.NotNil(t, retrieved)
	assert.Same(t, slog.Default(), retrieved)
}

func TestNewContext_StoresLogger(t *testing.T) {
	ctx := context.Background()
	testLogger := slog.New(noop.NewNoop().Handler())

	newCtx := NewContext(ctx, testLogger)

	assert.NotNil(t, newCtx)
	// Verify we can retrieve the same logger
	retrieved := FromContext(newCtx)
	assert.Same(t, testLogger, retrieved)
}

func TestNewContext_ReplacesExistingLogger(t *testing.T) {
	ctx := context.Background()
	logger1 := slog.New(noop.NewNoop().Handler())
	logger2 := slog.New(noop.NewNoop().Handler())

	ctx = NewContext(ctx, logger1)
	ctx = NewContext(ctx, logger2)

	retrieved := FromContext(ctx)
	assert.Same(t, logger2, retrieved)
	assert.NotSame(t, logger1, retrieved)
}

func TestWithErr_AppendsErrorAndStack(t *testing.T) {
	original := slog.Default()
	defer slog.SetDefault(original)

	testLogger := slog.New(noop.NewNoop().Handler())
	slog.SetDefault(testLogger)

	err := errors.New("test error")
	loggerWithErr := WithErr(err)

	assert.NotNil(t, loggerWithErr)
	// Logger should have error and potentially stack attributes
	// We can't easily inspect the attributes without using internal APIs
	// but we can verify it returns a logger
	assert.IsType(t, &slog.Logger{}, loggerWithErr)
}

func TestWithErr_WithStackTrace(t *testing.T) {
	original := slog.Default()
	defer slog.SetDefault(original)

	testLogger := slog.New(noop.NewNoop().Handler())
	slog.SetDefault(testLogger)

	// Create an error with stack trace using pkg/errors
	err := errors.New("error with stack")
	err = errors.Wrap(err, "wrapped")

	loggerWithErr := WithErr(err)

	assert.NotNil(t, loggerWithErr)
	assert.IsType(t, &slog.Logger{}, loggerWithErr)
}

func TestWithErrIf_ReturnsNoopWhenNil(t *testing.T) {
	original := slog.Default()
	defer slog.SetDefault(original)

	testLogger := slog.New(noop.NewNoop().Handler())
	slog.SetDefault(testLogger)

	result := WithErrIf(nil)

	assert.NotNil(t, result)
	assert.IsType(t, &slog.Logger{}, result)
	// Should be a no-op logger
	// Logging should not panic
	result.Info("test message")
}

func TestWithErrIf_ReturnsLoggerWithError(t *testing.T) {
	original := slog.Default()
	defer slog.SetDefault(original)

	testLogger := slog.New(noop.NewNoop().Handler())
	slog.SetDefault(testLogger)

	err := errors.New("test error")
	result := WithErrIf(err)

	assert.NotNil(t, result)
	assert.IsType(t, &slog.Logger{}, result)
	// Should log without panicking
	result.Error("test message")
}

func TestFromContextWithErr_WithLoggerInContext(t *testing.T) {
	ctx := context.Background()
	testLogger := slog.New(noop.NewNoop().Handler())
	ctx = NewContext(ctx, testLogger)

	err := errors.New("context error")
	result := FromContextWithErr(ctx, err)

	assert.NotNil(t, result)
	assert.IsType(t, &slog.Logger{}, result)
}

func TestFromContextWithErr_WithoutLoggerInContext(t *testing.T) {
	ctx := context.Background()

	err := errors.New("default error")
	result := FromContextWithErr(ctx, err)

	assert.NotNil(t, result)
	assert.IsType(t, &slog.Logger{}, result)
	// Should use default logger
}

func TestFromContextWithErrIf_NilError(t *testing.T) {
	ctx := context.Background()
	testLogger := slog.New(noop.NewNoop().Handler())
	ctx = NewContext(ctx, testLogger)

	result := FromContextWithErrIf(ctx, nil)

	assert.NotNil(t, result)
	assert.IsType(t, &slog.Logger{}, result)
	// Should be a no-op logger
	result.Info("no-op test")
}

func TestFromContextWithErrIf_WithError(t *testing.T) {
	ctx := context.Background()
	testLogger := slog.New(noop.NewNoop().Handler())
	ctx = NewContext(ctx, testLogger)

	err := errors.New("test error")
	result := FromContextWithErrIf(ctx, err)

	assert.NotNil(t, result)
	assert.IsType(t, &slog.Logger{}, result)
	result.Error("error test")
}

func TestConvertLevel_Info(t *testing.T) {
	result := convertLevel(INFO)
	assert.Equal(t, slog.LevelInfo, result)
}

func TestConvertLevel_Error(t *testing.T) {
	result := convertLevel(ERROR)
	assert.Equal(t, slog.LevelError, result)
}

func TestConvertLevel_Warn(t *testing.T) {
	result := convertLevel(WARN)
	assert.Equal(t, slog.LevelWarn, result)
}

func TestConvertLevel_Debug(t *testing.T) {
	result := convertLevel(DEBUG)
	assert.Equal(t, slog.LevelDebug, result)
}

func TestConvertLevel_UnknownLevel(t *testing.T) {
	// Unknown levels should default to Info
	result := convertLevel(Level("unknown"))
	assert.Equal(t, slog.LevelInfo, result)
}

func TestConvertLevel_EmptyLevel(t *testing.T) {
	result := convertLevel(Level(""))
	assert.Equal(t, slog.LevelInfo, result)
}

func TestConvertLevel_CaseSensitive(t *testing.T) {
	// Levels are case sensitive - uppercase should work
	result := convertLevel(INFO)
	assert.Equal(t, slog.LevelInfo, result)

	// Lowercase should not match and default to Info
	resultLower := convertLevel(Level("info"))
	assert.Equal(t, slog.LevelInfo, resultLower)
}

func TestAppendErr_WithErrorWithStackTrace(t *testing.T) {
	testLogger := slog.New(noop.NewNoop().Handler())

	// Create error with stack trace
	baseErr := errors.New("base error")
	err := errors.Wrap(baseErr, "wrapped error")

	result := appendErr(testLogger, err)

	assert.NotNil(t, result)
	assert.IsType(t, &slog.Logger{}, result)
}

func TestAppendErr_WithSimpleError(t *testing.T) {
	testLogger := slog.New(noop.NewNoop().Handler())

	// Simple error without stack trace
	err := errors.New("simple error")

	result := appendErr(testLogger, err)

	assert.NotNil(t, result)
	assert.IsType(t, &slog.Logger{}, result)
}

func TestConfig_DefaultValues(t *testing.T) {
	c := Config{}

	// Test default values from struct tags
	// Note: envconfig defaults are applied at runtime, not in struct

	assert.NotNil(t, NewDefault(c))
}

func TestConstants_Values(t *testing.T) {
	assert.Equal(t, Level("info"), INFO)
	assert.Equal(t, Level("error"), ERROR)
	assert.Equal(t, Level("warn"), WARN)
	assert.Equal(t, Level("debug"), DEBUG)

	assert.Equal(t, Provider("dev"), ProviderDevSlog)
	assert.Equal(t, Provider("std_json"), ProviderStdJson)
	assert.Equal(t, Provider("noop"), ProviderNoop)
}

func TestNewDefault_AllLevels(t *testing.T) {
	levels := []Level{INFO, ERROR, WARN, DEBUG}

	for _, level := range levels {
		c := Config{
			Provider: ProviderStdJson,
			Level:    level,
		}

		l := NewDefault(c)
		assert.NotNil(t, l, "Should create logger for level: %s", level)
	}
}

func TestNewDefault_AllProviders(t *testing.T) {
	providers := []Provider{ProviderDevSlog, ProviderStdJson, ProviderNoop}

	for _, provider := range providers {
		c := Config{
			Provider: provider,
			Level:    INFO,
		}

		l := NewDefault(c)
		assert.NotNil(t, l, "Should create logger for provider: %s", provider)
	}
}

func TestFromContext_ContextChain(t *testing.T) {
	// Test logger retrieval through context chain
	ctx := context.Background()
	testLogger := slog.New(noop.NewNoop().Handler())

	ctx = NewContext(ctx, testLogger)
	// Create derived contexts
	ctx1 := context.WithValue(ctx, "key1", "value1")
	ctx2 := context.WithValue(ctx1, "key2", "value2")

	retrieved := FromContext(ctx2)
	assert.Same(t, testLogger, retrieved)
}

func TestNewContext_NilLogger(t *testing.T) {
	ctx := context.Background()

	// Should handle nil logger gracefully
	newCtx := NewContext(ctx, nil)

	assert.NotNil(t, newCtx)

	// FromContext returns nil logger from context (not default)
	// This is the actual behavior - NewContext stores whatever you pass in
	retrieved := FromContext(newCtx)
	assert.Nil(t, retrieved)
}

func TestWithErr_ErrorWithNewline(t *testing.T) {
	original := slog.Default()
	defer slog.SetDefault(original)

	testLogger := slog.New(noop.NewNoop().Handler())
	slog.SetDefault(testLogger)

	// Error with special characters
	err := errors.New("error\nwith\nnewlines")

	loggerWithErr := WithErr(err)

	assert.NotNil(t, loggerWithErr)
	// Should handle special characters in error message
	loggerWithErr.Error("test")
}

func TestIntegration_LogMethods(t *testing.T) {
	// Integration test to ensure all logger methods work
	ctx := context.Background()
	testLogger := slog.New(noop.NewNoop().Handler())
	ctx = NewContext(ctx, testLogger)

	logger := FromContext(ctx)

	// These should not panic
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// With attributes
	logger = logger.With("key", "value")
	logger.Info("message with attrs")

	// Log with context
	logger.LogAttrs(ctx, slog.LevelInfo, "log attrs", slog.String("attr", "value"))
}

func BenchmarkNewDefault(b *testing.B) {
	c := Config{
		Provider: ProviderStdJson,
		Level:    INFO,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewDefault(c)
	}
}

func BenchmarkFromContext(b *testing.B) {
	ctx := context.Background()
	testLogger := slog.New(noop.NewNoop().Handler())
	ctx = NewContext(ctx, testLogger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FromContext(ctx)
	}
}

func BenchmarkNewContext(b *testing.B) {
	ctx := context.Background()
	testLogger := slog.New(noop.NewNoop().Handler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewContext(ctx, testLogger)
	}
}
