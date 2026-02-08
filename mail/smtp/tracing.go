package smtp

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("github.com/pure-golang/adapters/mail/smtp")
