package pgx

import (
	"context"
	"log/slog"
	"time"

	"github.com/pure-golang/adapters/logger"
	"github.com/jackc/pgx/v5/tracelog"
)

type Logger struct {
	minLevel tracelog.LogLevel
}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	if level < l.minLevel {
		return
	}

	// Добавляем duration в атрибуты если есть
	if duration, ok := data["time"]; ok {
		if d, ok := duration.(time.Duration); ok {
			data["duration_ms"] = d.Milliseconds()
		}
		delete(data, "time")
	}

	// Преобразуем в slog атрибуты
	attrs := make([]slog.Attr, 0, len(data))
	for k, v := range data {
		attrs = append(attrs, slog.Any(k, v))
	}

	logger.FromContext(ctx).WithGroup("postgres").LogAttrs(ctx, l.slogLevel(level), msg, attrs...)
}

func (l *Logger) slogLevel(level tracelog.LogLevel) slog.Level {
	switch level {
	case tracelog.LogLevelTrace:
		return slog.LevelDebug - 1
	case tracelog.LogLevelDebug:
		return slog.LevelDebug
	case tracelog.LogLevelInfo:
		return slog.LevelInfo
	case tracelog.LogLevelWarn:
		return slog.LevelWarn
	case tracelog.LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelError
	}
}
