package devslog

import (
	"log/slog"
	"os"

	"github.com/golang-cz/devslog"
)

func NewDefault(level slog.Level) *slog.Logger {
	opts := &devslog.Options{
		HandlerOptions: &slog.HandlerOptions{
			AddSource: true,
			Level:     level,
		},
		NewLineAfterLog:    true,
		MaxErrorStackTrace: 40,
		MaxSlicePrintSize:  40,
		SortKeys:           true,
		TimeFormat:         "[15:04:05]",
		DebugColor:         devslog.Magenta,
		StringerFormatter:  true,
	}

	return slog.New(devslog.NewHandler(os.Stdout, opts))
}
