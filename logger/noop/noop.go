package noop

import (
	"log/slog"
)

type writer struct {
	slog.Handler
}

func (n *writer) Write(_ []byte) (int, error) {
	return 0, nil
}

func NewNoop() *slog.Logger {
	return slog.New(slog.NewJSONHandler(new(writer), nil))
}
