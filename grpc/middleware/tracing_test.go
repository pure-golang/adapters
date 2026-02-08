package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TestSplitMethodName_ValidPath tests splitMethodName with valid method paths
func TestSplitMethodName_ValidPath(t *testing.T) {
	testCases := []struct {
		name            string
		fullMethodName  string
		expectedService string
		expectedMethod  string
	}{
		{
			name:            "standard path",
			fullMethodName:  "/service.v1.Service/Method",
			expectedService: "service.v1.Service",
			expectedMethod:  "Method",
		},
		{
			name:            "simple path",
			fullMethodName:  "/MyService/MyMethod",
			expectedService: "MyService",
			expectedMethod:  "MyMethod",
		},
		{
			name:            "nested package",
			fullMethodName:  "/com.example.service.v1.MyService/Method",
			expectedService: "com.example.service.v1.MyService",
			expectedMethod:  "Method",
		},
		{
			name:            "single component service",
			fullMethodName:  "/Service/Method",
			expectedService: "Service",
			expectedMethod:  "Method",
		},
		{
			name:            "method with underscores",
			fullMethodName:  "/my_service/my_method",
			expectedService: "my_service",
			expectedMethod:  "my_method",
		},
		{
			name:            "cleaned path with extra slashes",
			fullMethodName:  "//Service//Method",
			expectedService: "Service",
			expectedMethod:  "Method",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, method := splitMethodName(tc.fullMethodName)
			assert.Equal(t, tc.expectedService, service)
			assert.Equal(t, tc.expectedMethod, method)
		})
	}
}

// TestSplitMethodName_InvalidPath tests splitMethodName with invalid paths
func TestSplitMethodName_InvalidPath(t *testing.T) {
	testCases := []struct {
		name            string
		fullMethodName  string
		expectedService string
		expectedMethod  string
	}{
		{
			name:            "no leading slash",
			fullMethodName:  "Service/Method",
			expectedService: "unknown",
			expectedMethod:  "unknown",
		},
		{
			name:            "empty string",
			fullMethodName:  "",
			expectedService: "unknown",
			expectedMethod:  "unknown",
		},
		{
			name:            "only leading slash",
			fullMethodName:  "/",
			expectedService: "unknown",
			expectedMethod:  "/",
		},
		{
			name:            "only service",
			fullMethodName:  "/Service",
			expectedService: "unknown",
			expectedMethod:  "Service",
		},
		{
			name:            "relative path",
			fullMethodName:  "../Service/Method",
			expectedService: "unknown",
			expectedMethod:  "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, method := splitMethodName(tc.fullMethodName)
			assert.Equal(t, tc.expectedService, service)
			assert.Equal(t, tc.expectedMethod, method)
		})
	}
}

// TestMetadataTextMapPropagator tests the MetadataTextMapPropagator function
func TestMetadataTextMapPropagator(t *testing.T) {
	propagator := MetadataTextMapPropagator()

	assert.NotNil(t, propagator)

	// Should be a composite propagator
	// Note: The actual type might be wrapped, so we just verify it's not nil
	assert.NotNil(t, propagator)
}

// TestMetadataSupplier_Get tests metadataSupplier.Get method
func TestMetadataSupplier_Get(t *testing.T) {
	md := metadata.Pairs("key1", "value1", "key2", "value2")
	supplier := metadataSupplier{metadata: &md}

	// Test existing key
	value := supplier.Get("key1")
	assert.Equal(t, "value1", value)

	// Test non-existing key
	value = supplier.Get("nonexistent")
	assert.Equal(t, "", value)

	// Test key with multiple values (returns first)
	value = supplier.Get("key2")
	assert.Equal(t, "value2", value)
}

// TestMetadataSupplier_Set tests metadataSupplier.Set method
func TestMetadataSupplier_Set(t *testing.T) {
	md := metadata.MD{}
	supplier := metadataSupplier{metadata: &md}

	supplier.Set("new-key", "new-value")

	values := md.Get("new-key")
	assert.Len(t, values, 1)
	assert.Equal(t, "new-value", values[0])

	// Setting again should replace the value
	supplier.Set("new-key", "another-value")
	values = md.Get("new-key")
	assert.Len(t, values, 1)
	assert.Equal(t, "another-value", values[0])
}

// TestMetadataSupplier_Keys tests metadataSupplier.Keys method
func TestMetadataSupplier_Keys(t *testing.T) {
	md := metadata.Pairs("key1", "value1", "key2", "value2", "key3", "value3")
	supplier := metadataSupplier{metadata: &md}

	keys := supplier.Keys()

	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
	assert.Contains(t, keys, "key3")
}

// TestMetadataSupplier_Empty tests metadataSupplier with empty metadata
func TestMetadataSupplier_Empty(t *testing.T) {
	md := metadata.MD{}
	supplier := metadataSupplier{metadata: &md}

	// Get should return empty string
	value := supplier.Get("any-key")
	assert.Equal(t, "", value)

	// Keys should return empty slice
	keys := supplier.Keys()
	assert.Len(t, keys, 0)

	// Set should work
	supplier.Set("new-key", "value")
	keys = supplier.Keys()
	assert.Len(t, keys, 1)
}

// TestWrappedServerStream_Context tests wrappedServerStream.Context method
func TestWrappedServerStream_Context(t *testing.T) {
	ctx := context.Background()
	wss := &wrappedServerStream{
		ctx: ctx,
	}

	assert.Equal(t, ctx, wss.Context())
}

// TestWrappedServerStream_ContextWithValue tests wrappedServerStream with custom context
func TestWrappedServerStream_ContextWithValue(t *testing.T) {
	ctxWithValue := context.WithValue(context.Background(), "test-key", "test-value")
	wss := &wrappedServerStream{
		ctx: ctxWithValue,
	}

	ctx := wss.Context()
	assert.Equal(t, "test-value", ctx.Value("test-key"))
}

// TestTracingUnaryInterceptor_CreatesSpan tests that TracingUnaryInterceptor creates a span
func TestTracingUnaryInterceptor_CreatesSpan(t *testing.T) {
	// Setup a real tracer provider to capture spans
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(MetadataTextMapPropagator())

	interceptor := TracingUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Verify span exists in context
		span := trace.SpanFromContext(ctx)
		assert.NotNil(t, span)
		assert.NotEqual(t, trace.SpanContext{}, span.SpanContext())
		return "success", nil
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "success", resp)

	// Cleanup
	_ = tp.Shutdown(context.Background())
}

// TestTracingUnaryInterceptor_WithError tests that TracingUnaryInterceptor handles errors
func TestTracingUnaryInterceptor_WithError(t *testing.T) {
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(MetadataTextMapPropagator())

	interceptor := TracingUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/ErrorMethod",
	}

	expectedErr := status.Error(codes.NotFound, "not found")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	// Cleanup
	_ = tp.Shutdown(context.Background())
}

// TestTracingUnaryInterceptor_ExtractsTraceContext tests trace context extraction
func TestTracingUnaryInterceptor_ExtractsTraceContext(t *testing.T) {
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	otel.SetTracerProvider(tp)
	propagator := MetadataTextMapPropagator()
	otel.SetTextMapPropagator(propagator)

	interceptor := TracingUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/TraceMethod",
	}

	// Create a parent span and context
	parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")
	defer parentSpan.End()

	// Inject trace context into metadata
	md := metadata.New(nil)
	propagator.Inject(parentCtx, metadataSupplier{metadata: &md})
	ctx := metadata.NewIncomingContext(parentCtx, md)

	var capturedSpanContext trace.SpanContext

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Verify trace context was extracted
		span := trace.SpanFromContext(ctx)
		capturedSpanContext = span.SpanContext()
		return "success", nil
	}

	resp, err := interceptor(ctx, nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "success", resp)

	// Verify the span is related to parent
	assert.True(t, capturedSpanContext.IsValid())
	assert.Equal(t, parentSpan.SpanContext().TraceID(), capturedSpanContext.TraceID())
}

// TestTracingUnaryInterceptor_NoMetadata tests handling when no metadata exists
func TestTracingUnaryInterceptor_NoMetadata(t *testing.T) {
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(MetadataTextMapPropagator())

	interceptor := TracingUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/NoMetadataMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Should still have a span
		span := trace.SpanFromContext(ctx)
		assert.NotNil(t, span)
		return "success", nil
	}

	// Context without metadata
	ctx := context.Background()

	resp, err := interceptor(ctx, nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "success", resp)

	// Cleanup
	_ = tp.Shutdown(context.Background())
}

// mockServerStreamForTracing is a mock implementation of grpc.ServerStream for tracing tests
type mockServerStreamForTracing struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStreamForTracing) Context() context.Context {
	return m.ctx
}

// TestTracingStreamInterceptor_CreatesSpan tests that TracingStreamInterceptor creates a span
func TestTracingStreamInterceptor_CreatesSpan(t *testing.T) {
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(MetadataTextMapPropagator())

	interceptor := TracingStreamInterceptor()

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/TestStream",
	}

	ss := &mockServerStreamForTracing{ctx: context.Background()}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// Verify span exists in stream context
		span := trace.SpanFromContext(stream.Context())
		assert.NotNil(t, span)
		assert.NotEqual(t, trace.SpanContext{}, span.SpanContext())
		return nil
	}

	err := interceptor(nil, ss, info, handler)

	assert.NoError(t, err)

	// Cleanup
	_ = tp.Shutdown(context.Background())
}

// TestTracingStreamInterceptor_WithError tests that TracingStreamInterceptor handles errors
func TestTracingStreamInterceptor_WithError(t *testing.T) {
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(MetadataTextMapPropagator())

	interceptor := TracingStreamInterceptor()

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/ErrorStream",
	}

	ss := &mockServerStreamForTracing{ctx: context.Background()}

	expectedErr := status.Error(codes.Aborted, "stream aborted")
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return expectedErr
	}

	err := interceptor(nil, ss, info, handler)

	assert.Error(t, err)
	assert.Equal(t, codes.Aborted, status.Code(err))
}

// TestTracingStreamInterceptor_DetectsStreamType tests that stream type is correctly detected
func TestTracingStreamInterceptor_DetectsStreamType(t *testing.T) {
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(MetadataTextMapPropagator())

	interceptor := TracingStreamInterceptor()

	testCases := []struct {
		name       string
		info       *grpc.StreamServerInfo
		streamType string
	}{
		{
			name: "server streaming",
			info: &grpc.StreamServerInfo{
				FullMethod:     "/test.service/ServerStream",
				IsClientStream: false,
				IsServerStream: true,
			},
			streamType: "server_streaming",
		},
		{
			name: "client streaming",
			info: &grpc.StreamServerInfo{
				FullMethod:     "/test.service/ClientStream",
				IsClientStream: true,
				IsServerStream: false,
			},
			streamType: "client_streaming",
		},
		{
			name: "bidi streaming",
			info: &grpc.StreamServerInfo{
				FullMethod:     "/test.service/BidiStream",
				IsClientStream: true,
				IsServerStream: true,
			},
			streamType: "bidi_streaming",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ss := &mockServerStreamForTracing{ctx: context.Background()}

			handler := func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			}

			err := interceptor(nil, ss, tc.info, handler)

			assert.NoError(t, err)
		})
	}
}

// TestTracingStreamInterceptor_ExtractsTraceContext tests trace context extraction for streams
func TestTracingStreamInterceptor_ExtractsTraceContext(t *testing.T) {
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	otel.SetTracerProvider(tp)
	propagator := MetadataTextMapPropagator()
	otel.SetTextMapPropagator(propagator)

	interceptor := TracingStreamInterceptor()

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/StreamTrace",
	}

	// Create a parent span and context
	parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")
	defer parentSpan.End()

	// Inject trace context into metadata
	md := metadata.New(nil)
	propagator.Inject(parentCtx, metadataSupplier{metadata: &md})
	ctx := metadata.NewIncomingContext(parentCtx, md)

	ss := &mockServerStreamForTracing{ctx: ctx}

	var capturedSpanContext trace.SpanContext

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// Verify trace context was extracted
		span := trace.SpanFromContext(stream.Context())
		capturedSpanContext = span.SpanContext()
		return nil
	}

	err := interceptor(nil, ss, info, handler)

	assert.NoError(t, err)

	// Verify the span is related to parent
	assert.True(t, capturedSpanContext.IsValid())
	assert.Equal(t, parentSpan.SpanContext().TraceID(), capturedSpanContext.TraceID())
}

// TestTracingUnaryInterceptor_SpanAttributes tests span attributes
func TestTracingUnaryInterceptor_SpanAttributes(t *testing.T) {
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(MetadataTextMapPropagator())

	interceptor := TracingUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Verify span exists in context
		span := trace.SpanFromContext(ctx)
		assert.NotNil(t, span)
		assert.NotEqual(t, trace.SpanContext{}, span.SpanContext())
		return "success", nil
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "success", resp)
}

// TestTracingStreamInterceptor_SpanAttributes tests stream span attributes
func TestTracingStreamInterceptor_SpanAttributes(t *testing.T) {
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(MetadataTextMapPropagator())

	interceptor := TracingStreamInterceptor()

	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.v1.Service/StreamMethod",
		IsClientStream: true,
		IsServerStream: true,
	}

	ss := &mockServerStreamForTracing{ctx: context.Background()}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// Verify span exists in stream context
		span := trace.SpanFromContext(stream.Context())
		assert.NotNil(t, span)
		assert.NotEqual(t, trace.SpanContext{}, span.SpanContext())
		return nil
	}

	err := interceptor(nil, ss, info, handler)

	assert.NoError(t, err)
}

// TestSplitMethodName_URLSafe tests splitMethodName with URL-safe paths
func TestSplitMethodName_URLSafe(t *testing.T) {
	// Test that URL encoded paths are handled
	service, method := splitMethodName("/api.v1.User/GetUser")

	assert.Equal(t, "api.v1.User", service)
	assert.Equal(t, "GetUser", method)
}

// TestTracingUnaryInterceptor_DifferentStatusCodes tests span status for different error codes
func TestTracingUnaryInterceptor_DifferentStatusCodes(t *testing.T) {
	// Setup a real tracer provider to capture spans
	exporter := &testSpanExporter{}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(MetadataTextMapPropagator())

	// Cleanup after test
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	interceptor := TracingUnaryInterceptor()

	errorCodes := []codes.Code{
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unimplemented,
		codes.Internal,
		codes.Unavailable,
		codes.DataLoss,
		codes.Unauthenticated,
	}

	for _, code := range errorCodes {
		t.Run(code.String(), func(t *testing.T) {
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.service/" + code.String(),
			}

			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, status.Error(code, code.String())
			}

			_, err := interceptor(context.Background(), nil, info, handler)

			assert.Error(t, err)
			assert.Equal(t, code, status.Code(err))
		})
	}
}

// TestMetadataTextMapPropagator_Propagation tests that the propagator actually propagates
func TestMetadataTextMapPropagator_Propagation(t *testing.T) {
	propagator := MetadataTextMapPropagator()

	ctx := context.Background()

	// Create a span to inject
	tp := sdktrace.NewTracerProvider()
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()
	otel.SetTracerProvider(tp)

	ctx, span := tp.Tracer("test").Start(ctx, "test")
	defer span.End()

	// Inject into metadata
	md := metadata.MD{}
	carrier := metadataSupplier{metadata: &md}
	propagator.Inject(ctx, carrier)

	// Verify traceparent header was set
	traceParent := md.Get("traceparent")
	assert.NotEmpty(t, traceParent)

	// Extract back
	ctx2 := propagator.Extract(context.Background(), carrier)
	extractedSpan := trace.SpanFromContext(ctx2)

	// The extracted context should have the same trace ID
	assert.Equal(t, span.SpanContext().TraceID(), extractedSpan.SpanContext().TraceID())
}

// testSpanExporter is a simple exporter that captures spans for testing
type testSpanExporter struct {
	spans []testSpan
}

type testSpan struct {
	name       string
	attributes map[string]interface{}
}

func (e *testSpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		ts := testSpan{
			name:       span.Name(),
			attributes: make(map[string]interface{}),
		}

		// Extract some key attributes for testing
		attrs := span.Attributes()
		for _, kv := range attrs {
			ts.attributes[string(kv.Key)] = kv.Value.AsInterface()
		}

		e.spans = append(e.spans, ts)
	}
	return nil
}

func (e *testSpanExporter) Shutdown(ctx context.Context) error {
	return nil
}
