package redis

import (
	"log/slog"
)

// newLogger создаёт логгер с группой "redis"
func newLogger(base *slog.Logger) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	return base.WithGroup("redis")
}
