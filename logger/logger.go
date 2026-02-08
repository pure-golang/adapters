package logger

import (
	"context"
	"log/slog"

	"github.com/pure-golang/adapters/logger/devslog"
	"github.com/pure-golang/adapters/logger/noop"
	"github.com/pure-golang/adapters/logger/stdjson"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
)

type Level string
type Provider string
type contextKeyT string

var contextKey = contextKeyT("github.com/pure-golang/adapters/logger")

const (
	INFO  Level = "info"
	ERROR Level = "error"
	WARN  Level = "warn"
	DEBUG Level = "debug"

	ProviderDevSlog Provider = "dev"      // for dev
	ProviderStdJson Provider = "std_json" // for production
	ProviderNoop    Provider = "noop"     // for unit tests
)

type Config struct {
	Provider Provider `envconfig:"LOG_PROVIDER" default:"std_json"`
	Level    Level    `envconfig:"LOG_LEVEL" default:"info"`
}

// NewDefault creates a new instance of slog.Logger by default using Config.
func NewDefault(c Config) *slog.Logger {
	level := convertLevel(c.Level)
	switch c.Provider {
	case ProviderDevSlog:
		return devslog.NewDefault(level)
	case ProviderNoop:
		return noop.NewNoop()
	case ProviderStdJson:
		fallthrough
	default:
		return stdjson.NewDefault(level)
	}
}

// InitDefault creates a new instance of slog.Logger and set it by default.
func InitDefault(c Config) {
	slog.SetDefault(NewDefault(c))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		slog.Default().Error(err.Error())
	}))
}

// FromContext pack logger into context.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey).(*slog.Logger); ok {
		return l
	}

	return slog.Default()
}

// NewContext extract logger from context if exists or return default.
func NewContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey, l)
}

// WithErr return default logger with error.
func WithErr(err error) *slog.Logger {
	return appendErr(slog.Default(), err)
}

// FromContextWithErr extract logger from context and attach error field.
func FromContextWithErr(ctx context.Context, err error) *slog.Logger {
	l := FromContext(ctx)
	return appendErr(l, err)
}

// WithErrIf return default logger with error if err != nil.
// Otherwise no-op.
func WithErrIf(err error) *slog.Logger {
	if err == nil {
		return noop.NewNoop()
	}

	return WithErr(err)
}

// FromContextWithErrIf extract logger from context (default if not exists).
// Append error and stack trace.
// Returns no-op if err == nil.
func FromContextWithErrIf(ctx context.Context, err error) *slog.Logger {
	if err == nil {
		return noop.NewNoop()
	}

	return FromContextWithErr(ctx, err)
}

func appendErr(l *slog.Logger, err error) *slog.Logger {
	var stackTracer interface {
		StackTrace() errors.StackTrace
	}

	if errors.As(err, &stackTracer) {
		l = l.With("stack", stackTracer.StackTrace())
	}

	return l.With("error", err.Error())
}

func convertLevel(level Level) slog.Level {
	switch level {
	case INFO:
		return slog.LevelInfo
	case ERROR:
		return slog.LevelError
	case WARN:
		return slog.LevelWarn
	case DEBUG:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
