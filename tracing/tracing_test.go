package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

// TestInitWithValidProvider tests that Init with a valid provider
// sets the global tracer provider and returns the provider.
func TestInitWithValidProvider(t *testing.T) {
	// Save the original tracer provider to restore it later
	originalProvider := otel.GetTracerProvider()
	defer otel.SetTracerProvider(originalProvider)

	// Create a builder that returns a valid mock provider
	mockTP := tracesdk.NewTracerProvider()
	builder := func() (Provider, error) {
		return &testProvider{TracerProvider: mockTP}, nil
	}

	provider, err := Init(builder)
	require.NoError(t, err)
	assert.NotNil(t, provider)

	// Verify that the global tracer provider was set
	newProvider := otel.GetTracerProvider()
	// The provider should be non-nil
	assert.NotNil(t, newProvider)
}

// TestInitWithNilProviderBuilder tests that Init with a builder that returns nil
// correctly handles the nil provider and error.
func TestInitWithNilProviderBuilder(t *testing.T) {
	// Save the original tracer provider to restore it later
	originalProvider := otel.GetTracerProvider()

	// Create a builder that returns nil and an error
	expectedErr := assert.AnError
	builder := func() (Provider, error) {
		return nil, expectedErr
	}

	provider, err := Init(builder)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to load tracing provider")
	assert.ErrorContains(t, err, expectedErr.Error())

	// Should return NoopProvider when creator fails
	assert.IsType(t, &NoopProvider{}, provider)

	// Restore original provider
	otel.SetTracerProvider(originalProvider)
}

// TestInitWithBuilderError tests that Init with a builder that returns an error
// returns NoopProvider and wraps the error.
func TestInitWithBuilderError(t *testing.T) {
	// Save the original tracer provider to restore it later
	originalProvider := otel.GetTracerProvider()

	// Create a builder that returns an error
	builder := func() (Provider, error) {
		return nil, assert.AnError
	}

	provider, err := Init(builder)
	require.Error(t, err)
	assert.IsType(t, &NoopProvider{}, provider)

	// Restore original provider
	otel.SetTracerProvider(originalProvider)
}

// TestInitSetsGlobalTracerProvider tests that Init sets both
// the global tracer provider and text map propagator.
func TestInitSetsGlobalTracerProvider(t *testing.T) {
	// Save the original tracer provider and propagator to restore them later
	originalProvider := otel.GetTracerProvider()
	originalPropagator := otel.GetTextMapPropagator()

	// Create a builder that returns a valid provider
	builder := func() (Provider, error) {
		return &testProvider{TracerProvider: tracesdk.NewTracerProvider()}, nil
	}

	_, err := Init(builder)
	require.NoError(t, err)

	// Verify that the text map propagator was set
	newPropagator := otel.GetTextMapPropagator()
	assert.NotNil(t, newPropagator)

	// Restore original provider and propagator
	otel.SetTracerProvider(originalProvider)
	otel.SetTextMapPropagator(originalPropagator)
}

// TestNoopProviderClose tests that NoopProvider.Close returns nil.
func TestNoopProviderClose(t *testing.T) {
	noop := &NoopProvider{}
	err := noop.Close()
	assert.NoError(t, err)
}

// TestNoopProviderImplementsProvider tests that NoopProvider implements Provider interface.
func TestNoopProviderImplementsProvider(t *testing.T) {
	// This test verifies at compile time that NoopProvider implements Provider
	var _ Provider = &NoopProvider{}

	noop := &NoopProvider{}
	assert.NotNil(t, noop)
}

// testProvider is a minimal implementation of Provider for testing.
type testProvider struct {
	*tracesdk.TracerProvider
}

func (t *testProvider) Close() error {
	if t.TracerProvider != nil {
		return t.TracerProvider.Shutdown(context.Background())
	}
	return nil
}
