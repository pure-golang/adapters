package tracing

import (
	"io"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Provider interface {
	trace.TracerProvider
	io.Closer
}

// ProviderBuilder wrap all realization details of constructor (ex. config struct)
type ProviderBuilder func() (Provider, error)

func Init(creator ProviderBuilder) (Provider, error) {
	var provider Provider
	provider, err := creator()
	if err != nil {
		provider = &NoopProvider{}
	} else {
		otel.SetTracerProvider(provider)
		otel.SetTextMapPropagator(propagation.TraceContext{})
	}

	return provider, errors.Wrapf(err, "failed to load tracing provider")
}

type NoopProvider struct{ *tracesdk.TracerProvider }

func (NoopProvider) Close() error { return nil }
