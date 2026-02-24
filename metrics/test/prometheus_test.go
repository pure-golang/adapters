package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"github.com/pure-golang/adapters/metrics"
)

func TestInitPrometheus(t *testing.T) {
	t.Run("success initializes prometheus", func(t *testing.T) {
		if testing.Short() {
			t.Skip("integration test")
		}

		err := metrics.InitPrometheus()
		require.NoError(t, err)
	})

	t.Run("sets meter provider", func(t *testing.T) {
		if testing.Short() {
			t.Skip("integration test")
		}

		err := metrics.InitPrometheus()
		require.NoError(t, err)

		// Verify meter provider is set by checking it's not nil
		provider := otel.GetMeterProvider()
		assert.NotNil(t, provider)
	})

	t.Run("starts runtime instrumentation", func(t *testing.T) {
		if testing.Short() {
			t.Skip("integration test")
		}

		// This test verifies that runtime instrumentation starts without error
		// The runtime.Start() is called inside InitPrometheus
		err := metrics.InitPrometheus()
		require.NoError(t, err)

		// If we got here without error, runtime instrumentation started successfully
		assert.NoError(t, err)
	})

	t.Run("can be called multiple times", func(t *testing.T) {
		if testing.Short() {
			t.Skip("integration test")
		}

		// First call
		err := metrics.InitPrometheus()
		require.NoError(t, err)

		// Second call should also succeed
		err = metrics.InitPrometheus()
		require.NoError(t, err)
	})

	t.Run("meter provider returns valid meter", func(t *testing.T) {
		if testing.Short() {
			t.Skip("integration test")
		}

		err := metrics.InitPrometheus()
		require.NoError(t, err)

		provider := otel.GetMeterProvider()
		assert.NotNil(t, provider)

		// Get a meter to verify the provider works
		meter := provider.Meter("test-meter")
		assert.NotNil(t, meter)
	})
}
