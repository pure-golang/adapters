package pgx

import (
	"testing"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/stretchr/testify/assert"
)

func TestParseTraceLogLevel_ValidLevels(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected tracelog.LogLevel
	}{
		{
			name:     "trace level",
			input:    "trace",
			expected: tracelog.LogLevelTrace,
		},
		{
			name:     "debug level",
			input:    "debug",
			expected: tracelog.LogLevelDebug,
		},
		{
			name:     "info level",
			input:    "info",
			expected: tracelog.LogLevelInfo,
		},
		{
			name:     "warn level",
			input:    "warn",
			expected: tracelog.LogLevelWarn,
		},
		{
			name:     "error level",
			input:    "error",
			expected: tracelog.LogLevelError,
		},
		{
			name:     "none level",
			input:    "none",
			expected: tracelog.LogLevelNone,
		},
		// Note: pgx LogLevelFromString is case-sensitive
		// Uppercase inputs will fail and default to LogLevelNone
		{
			name:     "TRACE uppercase (fails, defaults to None)",
			input:    "TRACE",
			expected: tracelog.LogLevelNone,
		},
		{
			name:     "Debug mixed case (fails, defaults to None)",
			input:    "Debug",
			expected: tracelog.LogLevelNone,
		},
		{
			name:     "INFO uppercase (fails, defaults to None)",
			input:    "INFO",
			expected: tracelog.LogLevelNone,
		},
		{
			name:     "WARN uppercase (fails, defaults to None)",
			input:    "WARN",
			expected: tracelog.LogLevelNone,
		},
		{
			name:     "ERROR uppercase (fails, defaults to None)",
			input:    "ERROR",
			expected: tracelog.LogLevelNone,
		},
		{
			name:     "NONE uppercase",
			input:    "NONE",
			expected: tracelog.LogLevelNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTraceLogLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseTraceLogLevel_InvalidLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid string",
			input: "invalid",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "number string",
			input: "123",
		},
		{
			name:  "special characters",
			input: "!@#$%",
		},
		{
			name:  "partially valid",
			input: "debugging",
		},
		{
			name:  "similar but invalid",
			input: "tracee",
		},
		{
			name:  "whitespace",
			input: " ",
		},
		{
			name:  "level with spaces",
			input: "debug ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTraceLogLevel(tt.input)
			// Invalid levels should default to LogLevelNone
			assert.Equal(t, tracelog.LogLevelNone, result)
		})
	}
}

func TestParseTraceLogLevel_ProductionDefaults(t *testing.T) {
	// Test default production values
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty for production",
			input: "",
		},
		{
			name:  "error for production",
			input: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTraceLogLevel(tt.input)
			if tt.input == "" {
				// Empty should default to None
				assert.Equal(t, tracelog.LogLevelNone, result)
			} else {
				assert.Equal(t, tracelog.LogLevelError, result)
			}
		})
	}
}

func TestParseTraceLogLevel_DevelopmentDefaults(t *testing.T) {
	// Test typical development values
	tests := []struct {
		name     string
		input    string
		expected tracelog.LogLevel
	}{
		{
			name:     "debug for dev",
			input:    "debug",
			expected: tracelog.LogLevelDebug,
		},
		{
			name:     "trace for dev",
			input:    "trace",
			expected: tracelog.LogLevelTrace,
		},
		{
			name:     "info for dev",
			input:    "info",
			expected: tracelog.LogLevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTraceLogLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseTraceLogLevel_LogLevelValues(t *testing.T) {
	// Verify the actual numeric values match pgx expectations
	// In pgx v5, the values are: Trace=6, Debug=5, Info=4, Warn=3, Error=2, None=1
	assert.Equal(t, tracelog.LogLevel(6), tracelog.LogLevelTrace)
	assert.Equal(t, tracelog.LogLevel(5), tracelog.LogLevelDebug)
	assert.Equal(t, tracelog.LogLevel(4), tracelog.LogLevelInfo)
	assert.Equal(t, tracelog.LogLevel(3), tracelog.LogLevelWarn)
	assert.Equal(t, tracelog.LogLevel(2), tracelog.LogLevelError)
	assert.Equal(t, tracelog.LogLevel(1), tracelog.LogLevelNone)

	// Verify our parse function returns correct values
	result := parseTraceLogLevel("trace")
	assert.Equal(t, tracelog.LogLevel(6), result)

	result = parseTraceLogLevel("debug")
	assert.Equal(t, tracelog.LogLevel(5), result)

	result = parseTraceLogLevel("info")
	assert.Equal(t, tracelog.LogLevel(4), result)

	result = parseTraceLogLevel("warn")
	assert.Equal(t, tracelog.LogLevel(3), result)

	result = parseTraceLogLevel("error")
	assert.Equal(t, tracelog.LogLevel(2), result)

	result = parseTraceLogLevel("none")
	assert.Equal(t, tracelog.LogLevel(1), result)
}
