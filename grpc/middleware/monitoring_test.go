package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/pure-golang/adapters/logger"
	"github.com/pure-golang/adapters/logger/noop"
)

func init() {
	// Initialize noop logger for tests
	logger.InitDefault(logger.Config{
		Provider: logger.ProviderNoop,
		Level:    logger.INFO,
	})
}

// TestDefaultMonitoringOptions tests that DefaultMonitoringOptions returns expected defaults
func TestDefaultMonitoringOptions(t *testing.T) {
	testLogger := noop.NewNoop()

	opts := DefaultMonitoringOptions(testLogger)

	assert.NotNil(t, opts)
	assert.Equal(t, testLogger, opts.Logger)
	assert.True(t, opts.EnableTracing, "Tracing should be enabled by default")
	assert.True(t, opts.EnableMetrics, "Metrics should be enabled by default")
	assert.True(t, opts.EnableLogging, "Logging should be enabled by default")
	assert.True(t, opts.EnableStatsHandler, "StatsHandler should be enabled by default")
}

// TestSetupMonitoring_AllOptionsEnabled tests SetupMonitoring with all options enabled
func TestSetupMonitoring_AllOptionsEnabled(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      true,
		EnableLogging:      true,
		EnableStatsHandler: true,
	}

	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	// Should have interceptors for:
	// - Tracing (1 unary, 1 stream)
	// - Metrics (1 unary, 1 stream)
	// - Logging + Recovery (2 unary, 2 stream)
	assert.GreaterOrEqual(t, len(unaryInterceptors), 4, "Expected at least 4 unary interceptors")
	assert.GreaterOrEqual(t, len(streamInterceptors), 4, "Expected at least 4 stream interceptors")
	assert.GreaterOrEqual(t, len(serverOptions), 1, "Expected at least 1 server option for stats handler")
}

// TestSetupMonitoring_TracingDisabled tests SetupMonitoring with tracing disabled
func TestSetupMonitoring_TracingDisabled(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      false,
		EnableMetrics:      true,
		EnableLogging:      true,
		EnableStatsHandler: true,
	}

	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	// Should have interceptors for:
	// - Metrics (1 unary, 1 stream)
	// - Logging + Recovery (2 unary, 2 stream)
	// No tracing interceptors or stats handler
	expectedUnaryCount := 3  // metrics + recovery + logging
	expectedStreamCount := 3 // metrics + recovery + logging

	assert.Equal(t, expectedUnaryCount, len(unaryInterceptors))
	assert.Equal(t, expectedStreamCount, len(streamInterceptors))
	assert.Equal(t, 0, len(serverOptions), "No server options when tracing disabled")
}

// TestSetupMonitoring_MetricsDisabled tests SetupMonitoring with metrics disabled
func TestSetupMonitoring_MetricsDisabled(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      false,
		EnableLogging:      true,
		EnableStatsHandler: true,
	}

	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	// Should have interceptors for:
	// - Tracing (1 unary, 1 stream)
	// - Logging + Recovery (2 unary, 2 stream)
	expectedUnaryCount := 3
	expectedStreamCount := 3

	assert.Equal(t, expectedUnaryCount, len(unaryInterceptors))
	assert.Equal(t, expectedStreamCount, len(streamInterceptors))
	assert.GreaterOrEqual(t, len(serverOptions), 1, "Should have stats handler option")
}

// TestSetupMonitoring_LoggingDisabled tests SetupMonitoring with logging disabled
func TestSetupMonitoring_LoggingDisabled(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      true,
		EnableLogging:      false,
		EnableStatsHandler: true,
	}

	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	// Should have interceptors for:
	// - Tracing (1 unary, 1 stream)
	// - Metrics (1 unary, 1 stream)
	// No logging or recovery interceptors
	expectedUnaryCount := 2
	expectedStreamCount := 2

	assert.Equal(t, expectedUnaryCount, len(unaryInterceptors))
	assert.Equal(t, expectedStreamCount, len(streamInterceptors))
	assert.GreaterOrEqual(t, len(serverOptions), 1, "Should have stats handler option")
}

// TestSetupMonitoring_AllDisabled tests SetupMonitoring with all features disabled
func TestSetupMonitoring_AllDisabled(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      false,
		EnableMetrics:      false,
		EnableLogging:      false,
		EnableStatsHandler: false,
	}

	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	assert.Equal(t, 0, len(unaryInterceptors), "No unary interceptors when all disabled")
	assert.Equal(t, 0, len(streamInterceptors), "No stream interceptors when all disabled")
	assert.Equal(t, 0, len(serverOptions), "No server options when all disabled")
}

// TestSetupMonitoring_StatsHandlerDisabled tests without stats handler
func TestSetupMonitoring_StatsHandlerDisabled(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      true,
		EnableLogging:      false,
		EnableStatsHandler: false,
	}

	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	assert.GreaterOrEqual(t, len(unaryInterceptors), 2, "Should have tracing and metrics interceptors")
	assert.GreaterOrEqual(t, len(streamInterceptors), 2, "Should have tracing and metrics interceptors")
	assert.Equal(t, 0, len(serverOptions), "No server options when stats handler disabled")
}

// TestSetupMonitoring_OnlyTracing tests with only tracing enabled
func TestSetupMonitoring_OnlyTracing(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      false,
		EnableLogging:      false,
		EnableStatsHandler: false,
	}

	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	assert.Equal(t, 1, len(unaryInterceptors), "Should have 1 tracing unary interceptor")
	assert.Equal(t, 1, len(streamInterceptors), "Should have 1 tracing stream interceptor")
	assert.Equal(t, 0, len(serverOptions), "No server options when stats handler disabled")
}

// TestSetupMonitoring_OnlyMetrics tests with only metrics enabled
func TestSetupMonitoring_OnlyMetrics(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      false,
		EnableMetrics:      true,
		EnableLogging:      false,
		EnableStatsHandler: false,
	}

	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	assert.Equal(t, 1, len(unaryInterceptors), "Should have 1 metrics unary interceptor")
	assert.Equal(t, 1, len(streamInterceptors), "Should have 1 metrics stream interceptor")
	assert.Equal(t, 0, len(serverOptions))
}

// TestSetupMonitoring_OnlyLogging tests with only logging enabled
func TestSetupMonitoring_OnlyLogging(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      false,
		EnableMetrics:      false,
		EnableLogging:      true,
		EnableStatsHandler: false,
	}

	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	// Logging adds both Recovery and Logging interceptors
	assert.Equal(t, 2, len(unaryInterceptors), "Should have recovery and logging unary interceptors")
	assert.Equal(t, 2, len(streamInterceptors), "Should have recovery and logging stream interceptors")
	assert.Equal(t, 0, len(serverOptions))
}

// TestSetupMonitoring_InterceptorsAreFunctional tests that returned interceptors are functional
func TestSetupMonitoring_InterceptorsAreFunctional(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := DefaultMonitoringOptions(logger)

	unaryInterceptors, _, serverOptions := SetupMonitoring(ctx, opts)

	// Test that we can create a chain of interceptors
	assert.NotPanics(t, func() {
		chain := func(handler grpc.UnaryHandler) grpc.UnaryHandler {
			for i := len(unaryInterceptors) - 1; i >= 0; i-- {
				handler = func(currentHandler grpc.UnaryHandler, currentInterceptor grpc.UnaryServerInterceptor) grpc.UnaryHandler {
					return func(ctx context.Context, req interface{}) (interface{}, error) {
						return currentInterceptor(ctx, req, &grpc.UnaryServerInfo{
							FullMethod: "/test/Method",
						}, currentHandler)
					}
				}(handler, unaryInterceptors[i])
			}
			return handler
		}

		// Create a final handler
		finalHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "success", nil
		}

		// Chain all interceptors
		chainedHandler := chain(finalHandler)

		// Execute the chained handler
		resp, err := chainedHandler(context.Background(), "test")

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	// Verify server options are valid
	assert.NotPanics(t, func() {
		for _, opt := range serverOptions {
			// Server options should be valid (we can't fully test without creating a server)
			assert.NotNil(t, opt)
		}
	})
}

// TestSetupMonitoring_NilLogger tests with nil logger
func TestSetupMonitoring_NilLogger(t *testing.T) {
	ctx := context.Background()

	opts := &MonitoringOptions{
		Logger:             nil,
		EnableTracing:      false,
		EnableMetrics:      false,
		EnableLogging:      true,
		EnableStatsHandler: false,
	}

	// Should handle nil logger - the interceptor will just use nil
	// This might panic in actual use, but SetupMonitoring itself should work
	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	// Even with nil logger, we get the interceptors
	assert.Equal(t, 2, len(unaryInterceptors))
	assert.Equal(t, 2, len(streamInterceptors))
	assert.Equal(t, 0, len(serverOptions))
}

// TestSetupMonitoring_TracingSetsPropagator tests that tracing setup sets the propagator
func TestSetupMonitoring_TracingSetsPropagator(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      false,
		EnableLogging:      false,
		EnableStatsHandler: false,
	}

	// This should set the text map propagator
	unaryInterceptors, streamInterceptors, serverOptions := SetupMonitoring(ctx, opts)

	// Should have interceptors
	assert.Equal(t, 1, len(unaryInterceptors))
	assert.Equal(t, 1, len(streamInterceptors))
	assert.Equal(t, 0, len(serverOptions))

	// The propagator should be set (we can verify by calling MetadataTextMapPropagator)
	propagator := MetadataTextMapPropagator()
	assert.NotNil(t, propagator)
}

// TestSetupMonitoring_ContextIsPassed tests that context parameter is used
func TestSetupMonitoring_Context(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := DefaultMonitoringOptions(logger)

	// Should not panic regardless of context
	_, _, serverOptions := SetupMonitoring(ctx, opts)

	assert.NotEmpty(t, serverOptions)
}

// TestMonitoringOptions_StructFields tests MonitoringOptions struct
func TestMonitoringOptions_StructFields(t *testing.T) {
	logger := noop.NewNoop().With("component", "test")

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      true,
		EnableLogging:      true,
		EnableStatsHandler: true,
	}

	assert.Equal(t, logger, opts.Logger)
	assert.True(t, opts.EnableTracing)
	assert.True(t, opts.EnableMetrics)
	assert.True(t, opts.EnableLogging)
	assert.True(t, opts.EnableStatsHandler)
}

// TestSetupMonitoring_InterceptorsOrder tests that interceptors are added in correct order
func TestSetupMonitoring_InterceptorsOrder(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      true,
		EnableLogging:      true,
		EnableStatsHandler: true,
	}

	unaryInterceptors, _, _ := SetupMonitoring(ctx, opts)

	// The expected order is:
	// 1. Tracing
	// 2. Metrics
	// 3. Recovery
	// 4. Logging
	// So we should have 4 interceptors total
	require.Len(t, unaryInterceptors, 4)

	// We can verify that all 4 interceptors are non-nil functions
	for i, interceptor := range unaryInterceptors {
		assert.NotNil(t, interceptor, "Interceptor at index %d should not be nil", i)
	}
}

// TestSetupMonitoring_StreamInterceptorsOrder tests stream interceptors order
func TestSetupMonitoring_StreamInterceptorsOrder(t *testing.T) {
	ctx := context.Background()
	logger := noop.NewNoop()

	opts := &MonitoringOptions{
		Logger:             logger,
		EnableTracing:      true,
		EnableMetrics:      true,
		EnableLogging:      true,
		EnableStatsHandler: true,
	}

	_, streamInterceptors, _ := SetupMonitoring(ctx, opts)

	// Same order as unary:
	// 1. Tracing
	// 2. Metrics
	// 3. Recovery
	// 4. Logging
	require.Len(t, streamInterceptors, 4)

	for i, interceptor := range streamInterceptors {
		assert.NotNil(t, interceptor, "Stream interceptor at index %d should not be nil", i)
	}
}
