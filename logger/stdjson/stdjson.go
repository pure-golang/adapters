package stdjson

import (
	"log/slog"
	"os"
)

func NewDefault(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
