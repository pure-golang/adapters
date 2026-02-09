package cli

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/pure-golang/adapters/executor/cli")

// recordError записывает ошибку в span
func recordError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
