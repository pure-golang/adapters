package metrics

import (
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// InitPrometheus implements opentelemetry interfaces  and set global state
func InitPrometheus() error {
	exporter, err := prometheus.New()
	if err != nil {
		return errors.Wrap(err, "failed to create prometheus instance")
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))

	otel.SetMeterProvider(provider)

	if err := runtime.Start(); err != nil {
		return errors.Wrap(err, "failed to start runtime")
	}

	return nil
}
