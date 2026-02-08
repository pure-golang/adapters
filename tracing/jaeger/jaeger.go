package jaeger

import (
	"context"
	"fmt"

	"github.com/pure-golang/adapters/tracing"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

var _ tracing.Provider = (*Provider)(nil)

type Config struct {
	EndPoint    string `envconfig:"TRACING_ENDPOINT" required:"true"`
	ServiceName string `envconfig:"SERVICE_NAME" required:"true"`
	AppVersion  string `envconfig:"APP_VERSION" required:"true"`
}

// Provider extends tracesdk.TraceProvider based on jaeger.Exporter
type Provider struct {
	*tracesdk.TracerProvider
}

func (j *Provider) Close() error {
	ctx := context.Background()
	if err := j.ForceFlush(ctx); err != nil {
		// Ensure shutdown is called even if ForceFlush fails
		shutdownErr := j.TracerProvider.Shutdown(ctx)
		if shutdownErr != nil {
			return errors.Wrap(err, "jaeger force flush failed (also shutdown failed)")
		}
		return errors.Wrap(err, "jaeger force flush failed")
	}
	err := j.TracerProvider.Shutdown(ctx)

	return errors.Wrap(err, "shutdown jaeger")
}

func NewProviderBuilder(conf Config) func() (tracing.Provider, error) {
	return func() (tracing.Provider, error) {
		if conf.EndPoint == "" {
			return nil, errors.New("empty connection string")
		}
		if conf.ServiceName == "" {
			return nil, errors.New("service name is empty")
		}

		exp, err := otlptrace.New(
			context.Background(),
			otlptracehttp.NewClient(
				otlptracehttp.WithEndpointURL(conf.EndPoint),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create jaeger instance: %v", err)
		}
		tp := tracesdk.NewTracerProvider(
			tracesdk.WithBatcher(exp),
			tracesdk.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(conf.ServiceName),
				semconv.ServiceVersionKey.String(conf.AppVersion),
			)),
			tracesdk.WithSampler(tracesdk.AlwaysSample()),
		)

		return &Provider{TracerProvider: tp}, nil
	}
}
