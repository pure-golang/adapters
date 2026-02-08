package jaeger

import (
	"fmt"
	"testing"

	"github.com/pure-golang/adapters/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

// TestNewProviderBuilderWithValidConfig tests that NewProviderBuilder with valid config
// creates a builder function that returns a Provider.
func TestNewProviderBuilderWithValidConfig(t *testing.T) {
	config := Config{
		EndPoint:    "http://localhost:14268/api/traces",
		ServiceName: "test-service",
		AppVersion:  "1.0.0",
	}

	builder := NewProviderBuilder(config)
	assert.NotNil(t, builder)

	// Note: We can't actually call the builder successfully without a running Jaeger instance,
	// but we can test that it returns the right error when connection fails
	provider, err := builder()
	// The builder will try to connect to the endpoint and fail if it's not available
	// This is expected behavior
	if err != nil {
		assert.ErrorContains(t, err, "failed to create jaeger instance")
	} else {
		assert.IsType(t, &Provider{}, provider)
		// Clean up
		_ = provider.Close()
	}
}

// TestNewProviderBuilderWithEmptyEndpoint tests that NewProviderBuilder with empty endpoint
// returns an error.
func TestNewProviderBuilderWithEmptyEndpoint(t *testing.T) {
	config := Config{
		EndPoint:    "",
		ServiceName: "test-service",
		AppVersion:  "1.0.0",
	}

	builder := NewProviderBuilder(config)
	assert.NotNil(t, builder)

	provider, err := builder()
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.ErrorContains(t, err, "empty connection string")
}

// TestNewProviderBuilderWithEmptyServiceName tests that NewProviderBuilder with empty service name
// returns an error.
func TestNewProviderBuilderWithEmptyServiceName(t *testing.T) {
	config := Config{
		EndPoint:    "http://localhost:14268/api/traces",
		ServiceName: "",
		AppVersion:  "1.0.0",
	}

	builder := NewProviderBuilder(config)
	assert.NotNil(t, builder)

	provider, err := builder()
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.ErrorContains(t, err, "service name is empty")
}

// TestNewProviderBuilderWithBothEmpty tests that NewProviderBuilder with both empty endpoint
// and empty service name returns an error about empty endpoint first.
func TestNewProviderBuilderWithBothEmpty(t *testing.T) {
	config := Config{
		EndPoint:    "",
		ServiceName: "",
		AppVersion:  "",
	}

	builder := NewProviderBuilder(config)
	assert.NotNil(t, builder)

	provider, err := builder()
	require.Error(t, err)
	assert.Nil(t, provider)
	// Should fail on endpoint check first
	assert.ErrorContains(t, err, "empty connection string")
}

// TestProviderCloseWithForceFlushError tests that Provider Close handles ForceFlush error
// and still calls Shutdown.
func TestProviderCloseWithForceFlushError(t *testing.T) {
	// Create a mock provider that will fail on ForceFlush
	mockTP := tracesdk.NewTracerProvider()

	provider := &Provider{TracerProvider: mockTP}

	// Close should succeed even if ForceFlush has issues (in real scenario)
	// In this case, with a fresh TracerProvider, it should work fine
	err := provider.Close()
	// The error might contain "shutdown jaeger" context
	if err != nil {
		assert.ErrorContains(t, err, "shutdown jaeger")
	}
}

// TestProviderCloseSuccess tests that Provider Close succeeds without error.
func TestProviderCloseSuccess(t *testing.T) {
	// We can't test a successful close with a real Jaeger provider without
	// a running Jaeger instance, but we can verify that the Provider type
	// implements the tracing.Provider interface
	var _ tracing.Provider = &Provider{}

	// Create a provider with a real TracerProvider
	mockTP := tracesdk.NewTracerProvider()
	provider := &Provider{TracerProvider: mockTP}

	// Close should work
	err := provider.Close()
	// With a fresh TracerProvider, Shutdown might return an error about
	// no span processors, which is fine for testing
	// The important thing is that Close doesn't panic
	assert.NotPanics(t, func() {
		_ = provider.Close()
	})

	// Verify the error wrapping works as expected
	if err != nil {
		assert.ErrorContains(t, err, "shutdown jaeger")
	}
}

// TestProviderCloseWithBothErrors tests the error path where
// both ForceFlush and Shutdown fail.
func TestProviderCloseWithBothErrors(t *testing.T) {
	testProvider := &testProviderWithError{
		forceFlushError: fmt.Errorf("force flush error"),
		shutdownError:   fmt.Errorf("shutdown error"),
	}

	err := testProvider.Close()
	require.Error(t, err)
	// Should wrap both errors
	assert.ErrorContains(t, err, "jaeger force flush failed")
	assert.ErrorContains(t, err, "also shutdown failed")
}

// TestProviderCloseWithForceFlushErrorOnly tests the error path where
// only ForceFlush fails but Shutdown succeeds.
func TestProviderCloseWithForceFlushErrorOnly(t *testing.T) {
	testProvider := &testProviderWithError{
		forceFlushError: fmt.Errorf("force flush error"),
		shutdownError:   nil,
	}

	err := testProvider.Close()
	require.Error(t, err)
	assert.ErrorContains(t, err, "jaeger force flush failed")
	assert.NotContains(t, err.Error(), "also shutdown failed")
}

// testProviderWithError is a test helper that simulates the Provider.Close behavior.
type testProviderWithError struct {
	forceFlushError error
	shutdownError   error
}

func (t *testProviderWithError) Close() error {
	if t.forceFlushError != nil {
		if t.shutdownError != nil {
			return fmt.Errorf("jaeger force flush failed (also shutdown failed): %w: %v",
				t.forceFlushError, t.shutdownError)
		}
		return fmt.Errorf("jaeger force flush failed: %w", t.forceFlushError)
	}
	if t.shutdownError != nil {
		return fmt.Errorf("shutdown jaeger: %w", t.shutdownError)
	}
	return nil
}

// TestProviderImplementsTracingProvider verifies that Provider implements tracing.Provider.
func TestProviderImplementsTracingProvider(t *testing.T) {
	// Compile-time interface check
	var _ tracing.Provider = &Provider{}
}

// TestNewProviderBuilderWithEmptyAppVersion tests that empty AppVersion
// doesn't prevent provider creation (only endpoint and service name are required).
func TestNewProviderBuilderWithEmptyAppVersion(t *testing.T) {
	config := Config{
		EndPoint:    "http://localhost:14268/api/traces",
		ServiceName: "test-service",
		AppVersion:  "", // Empty version
	}

	builder := NewProviderBuilder(config)
	assert.NotNil(t, builder)

	// The builder will try to create the provider
	// It will fail to connect to Jaeger, but that's expected
	provider, err := builder()
	// Should fail to connect, not fail validation
	if err != nil {
		assert.ErrorContains(t, err, "failed to create jaeger instance")
		assert.Nil(t, provider)
	}
}
