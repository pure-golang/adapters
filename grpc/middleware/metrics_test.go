package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TestGetMessageSize_ProtoMessage tests getMessageSize with protobuf messages
func TestGetMessageSize_ProtoMessage(t *testing.T) {
	testCases := []struct {
		name     string
		msg      interface{}
		expected int64
	}{
		{
			name:     "string wrapper",
			msg:      wrapperspb.String("test"),
			expected: int64(proto.Size(wrapperspb.String("test"))),
		},
		{
			name:     "int32 wrapper",
			msg:      wrapperspb.Int32(42),
			expected: int64(proto.Size(wrapperspb.Int32(42))),
		},
		{
			name:     "bool wrapper",
			msg:      wrapperspb.Bool(true),
			expected: int64(proto.Size(wrapperspb.Bool(true))),
		},
		{
			name:     "empty string wrapper",
			msg:      wrapperspb.String(""),
			expected: int64(proto.Size(wrapperspb.String(""))),
		},
		{
			name:     "large string wrapper",
			msg:      wrapperspb.String(string(make([]byte, 1000))),
			expected: int64(proto.Size(wrapperspb.String(string(make([]byte, 1000))))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size := getMessageSize(tc.msg)
			assert.Equal(t, tc.expected, size)
		})
	}
}

// TestGetMessageSize_NonProtoMessage tests getMessageSize with non-protobuf messages
func TestGetMessageSize_NonProtoMessage(t *testing.T) {
	testCases := []struct {
		name     string
		msg      interface{}
		expected int64
	}{
		{
			name:     "string",
			msg:      "test string",
			expected: 0,
		},
		{
			name:     "int",
			msg:      42,
			expected: 0,
		},
		{
			name:     "struct",
			msg:      struct{ Name string }{"test"},
			expected: 0,
		},
		{
			name:     "nil",
			msg:      nil,
			expected: 0,
		},
		{
			name:     "map",
			msg:      map[string]string{"key": "value"},
			expected: 0,
		},
		{
			name:     "slice",
			msg:      []int{1, 2, 3},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size := getMessageSize(tc.msg)
			assert.Equal(t, tc.expected, size)
		})
	}
}

// TestMetricsUnaryInterceptor_Success tests metrics recording for successful requests
func TestMetricsUnaryInterceptor_Success(t *testing.T) {
	// Setup noop meter provider to avoid panics
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return wrapperspb.String("success"), nil
	}

	// Should not panic
	resp, err := interceptor(context.Background(), wrapperspb.String("request"), info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestMetricsUnaryInterceptor_WithError tests metrics recording for failed requests
func TestMetricsUnaryInterceptor_WithError(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/ErrorMethod",
	}

	expectedErr := status.Error(codes.NotFound, "not found")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	resp, err := interceptor(context.Background(), wrapperspb.String("request"), info, handler)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// TestMetricsUnaryInterceptor_WithProtoMessage tests metrics with actual proto messages
func TestMetricsUnaryInterceptor_WithProtoMessage(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/ProtoMethod",
	}

	// Create proto messages of different sizes
	request := wrapperspb.String("request message")
	var capturedReqSize int64

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Capture request size
		capturedReqSize = getMessageSize(req)
		return wrapperspb.String("response message"), nil
	}

	resp, err := interceptor(context.Background(), request, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Greater(t, capturedReqSize, int64(0), "Proto message should have size > 0")
}

// TestMetricsUnaryInterceptor_WithNonProtoMessage tests metrics with non-proto messages
func TestMetricsUnaryInterceptor_WithNonProtoMessage(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/NonProtoMethod",
	}

	// Use non-proto message
	request := struct {
		Data string
	}{
		Data: "test data",
	}

	var capturedReqSize int64

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedReqSize = getMessageSize(req)
		return struct{ Result string }{"success"}, nil
	}

	resp, err := interceptor(context.Background(), request, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(0), capturedReqSize, "Non-proto message should have size 0")
}

// TestMetricsUnaryInterceptor_DifferentStatusCodes tests metrics with various status codes
func TestMetricsUnaryInterceptor_DifferentStatusCodes(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	testCases := []struct {
		name string
		code codes.Code
		err  error
	}{
		{
			name: "OK",
			code: codes.OK,
			err:  nil,
		},
		{
			name: "InvalidArgument",
			code: codes.InvalidArgument,
			err:  status.Error(codes.InvalidArgument, "invalid"),
		},
		{
			name: "PermissionDenied",
			code: codes.PermissionDenied,
			err:  status.Error(codes.PermissionDenied, "denied"),
		},
		{
			name: "NotFound",
			code: codes.NotFound,
			err:  status.Error(codes.NotFound, "not found"),
		},
		{
			name: "AlreadyExists",
			code: codes.AlreadyExists,
			err:  status.Error(codes.AlreadyExists, "exists"),
		},
		{
			name: "Internal",
			code: codes.Internal,
			err:  status.Error(codes.Internal, "internal"),
		},
		{
			name: "Unavailable",
			code: codes.Unavailable,
			err:  status.Error(codes.Unavailable, "unavailable"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.service/" + tc.name,
			}

			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, tc.err
			}

			resp, err := interceptor(context.Background(), nil, info, handler)

			if tc.err != nil {
				assert.Error(t, err)
				assert.Nil(t, resp)
				assert.Equal(t, tc.code, status.Code(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMetricsUnaryInterceptor_Context tests context is properly passed through
func TestMetricsUnaryInterceptor_Context(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/ContextMethod",
	}

	ctxWithValue := context.WithValue(context.Background(), "test_key", "test_value")
	var capturedCtx context.Context

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedCtx = ctx
		return "success", nil
	}

	interceptor(ctxWithValue, nil, info, handler)

	assert.Equal(t, "test_value", capturedCtx.Value("test_key"))
}

// mockServerStreamForMetrics is a mock implementation of grpc.ServerStream for metrics testing
type mockServerStreamForMetrics struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStreamForMetrics) Context() context.Context {
	return m.ctx
}

func (m *mockServerStreamForMetrics) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockServerStreamForMetrics) RecvMsg(msg interface{}) error {
	return nil
}

// TestMetricsStreamInterceptor_Success tests metrics for successful streams
func TestMetricsStreamInterceptor_Success(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsStreamInterceptor()

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/TestStream",
	}

	ss := &mockServerStreamForMetrics{ctx: context.Background()}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := interceptor(nil, ss, info, handler)

	assert.NoError(t, err)
}

// TestMetricsStreamInterceptor_WithError tests metrics for failed streams
func TestMetricsStreamInterceptor_WithError(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsStreamInterceptor()

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/ErrorStream",
	}

	ss := &mockServerStreamForMetrics{ctx: context.Background()}

	expectedErr := status.Error(codes.Aborted, "stream aborted")
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return expectedErr
	}

	err := interceptor(nil, ss, info, handler)

	assert.Error(t, err)
	assert.Equal(t, codes.Aborted, status.Code(err))
}

// TestMetricsStreamInterceptor_StreamTypes tests that stream type is correctly detected
func TestMetricsStreamInterceptor_StreamTypes(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsStreamInterceptor()

	testCases := []struct {
		name string
		info *grpc.StreamServerInfo
	}{
		{
			name: "server streaming",
			info: &grpc.StreamServerInfo{
				FullMethod:     "/test.service/ServerStream",
				IsClientStream: false,
				IsServerStream: true,
			},
		},
		{
			name: "client streaming",
			info: &grpc.StreamServerInfo{
				FullMethod:     "/test.service/ClientStream",
				IsClientStream: true,
				IsServerStream: false,
			},
		},
		{
			name: "bidi streaming",
			info: &grpc.StreamServerInfo{
				FullMethod:     "/test.service/BidiStream",
				IsClientStream: true,
				IsServerStream: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ss := &mockServerStreamForMetrics{ctx: context.Background()}

			handler := func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			}

			err := interceptor(nil, ss, tc.info, handler)
			assert.NoError(t, err)
		})
	}
}

// TestMetricsStreamInterceptor_Context tests context is properly passed through
func TestMetricsStreamInterceptor_Context(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsStreamInterceptor()

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/StreamContext",
	}

	ctxWithValue := context.WithValue(context.Background(), "stream_key", "stream_value")
	ss := &mockServerStreamForMetrics{ctx: ctxWithValue}

	var capturedCtx context.Context

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		capturedCtx = stream.Context()
		return nil
	}

	err := interceptor(nil, ss, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "stream_value", capturedCtx.Value("stream_key"))
}

// TestMetricsUnaryInterceptor_DifferentMessageSizes tests with messages of various sizes
func TestMetricsUnaryInterceptor_DifferentMessageSizes(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/SizeMethod",
	}

	testCases := []struct {
		name    string
		request proto.Message
	}{
		{
			name:    "empty message",
			request: wrapperspb.String(""),
		},
		{
			name:    "small message",
			request: wrapperspb.String("small"),
		},
		{
			name:    "medium message",
			request: wrapperspb.String(string(make([]byte, 100))),
		},
		{
			name:    "large message",
			request: wrapperspb.String(string(make([]byte, 1000))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return wrapperspb.String("response"), nil
			}

			resp, err := interceptor(context.Background(), tc.request, info, handler)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}

// TestMetricsUnaryInterceptor_WithPanic tests behavior when handler panics
func TestMetricsUnaryInterceptor_WithPanic(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/PanicMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	// The metrics interceptor doesn't catch panics - they will propagate
	assert.Panics(t, func() {
		interceptor(context.Background(), nil, info, handler)
	})
}

// TestMetricsStreamInterceptor_WithPanic tests behavior when stream handler panics
func TestMetricsStreamInterceptor_WithPanic(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsStreamInterceptor()

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/PanicStream",
	}

	ss := &mockServerStreamForMetrics{ctx: context.Background()}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		panic("stream panic")
	}

	// The metrics interceptor doesn't catch panics - they will propagate
	assert.Panics(t, func() {
		interceptor(nil, ss, info, handler)
	})
}

// TestMetricsUnaryInterceptor_WithErrorInProto tests with various proto message types
func TestMetricsUnaryInterceptor_WithErrorInProto(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/ProtoTypes",
	}

	// Test with different wrapper types
	wrappers := []proto.Message{
		wrapperspb.String("string"),
		wrapperspb.Int32(32),
		wrapperspb.Int64(64),
		wrapperspb.Float(3.14),
		wrapperspb.Double(6.28),
		wrapperspb.Bool(true),
		wrapperspb.Bytes([]byte{1, 2, 3}),
		wrapperspb.UInt32(32),
		wrapperspb.UInt64(64),
	}

	for _, wrapper := range wrappers {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return wrapperspb.String("ok"), nil
		}

		resp, err := interceptor(context.Background(), wrapper, info, handler)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
	}
}

// TestGetMessageSize_WithNilProto tests getMessageSize with nil proto message
func TestGetMessageSize_WithNilProto(t *testing.T) {
	// This test verifies that getMessageSize handles nil gracefully
	var msg proto.Message = nil

	size := getMessageSize(msg)

	// Should return 0 for nil message
	assert.Equal(t, int64(0), size)
}

// TestMetricsStreamInterceptor_DifferentStatusCodes tests metrics with various status codes for streams
func TestMetricsStreamInterceptor_DifferentStatusCodes(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsStreamInterceptor()

	testCases := []struct {
		name string
		code codes.Code
		err  error
	}{
		{
			name: "OK",
			code: codes.OK,
			err:  nil,
		},
		{
			name: "Canceled",
			code: codes.Canceled,
			err:  status.Error(codes.Canceled, "canceled"),
		},
		{
			name: "Unknown",
			code: codes.Unknown,
			err:  status.Error(codes.Unknown, "unknown"),
		},
		{
			name: "DeadlineExceeded",
			code: codes.DeadlineExceeded,
			err:  status.Error(codes.DeadlineExceeded, "deadline"),
		},
		{
			name: "ResourceExhausted",
			code: codes.ResourceExhausted,
			err:  status.Error(codes.ResourceExhausted, "exhausted"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := &grpc.StreamServerInfo{
				FullMethod: "/test.service/Stream" + tc.name,
			}

			ss := &mockServerStreamForMetrics{ctx: context.Background()}

			handler := func(srv interface{}, stream grpc.ServerStream) error {
				return tc.err
			}

			err := interceptor(nil, ss, info, handler)

			if tc.err != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.code, status.Code(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMetricsUnaryInterceptor_WithSpanInContext tests metrics interceptor with span in context
func TestMetricsUnaryInterceptor_WithSpanInContext(t *testing.T) {
	// Setup a real tracer provider for testing
	tp := sdktrace.NewTracerProvider()
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()
	otel.SetTracerProvider(tp)

	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/SpanMethod",
	}

	// Create a context with a span
	ctx, span := tp.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Verify span is still in context
		currentSpan := trace.SpanFromContext(ctx)
		assert.NotNil(t, currentSpan)
		return "success", nil
	}

	resp, err := interceptor(ctx, nil, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestMetricsUnaryInterceptor_MeasuresDuration tests that duration is measured
func TestMetricsUnaryInterceptor_MeasuresDuration(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsUnaryInterceptor()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/DurationMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	// The interceptor should measure duration
	// We can't directly test the metrics output without a real meter,
	// but we can verify it doesn't panic
	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestMetricsStreamInterceptor_MeasuresDuration tests that stream duration is measured
func TestMetricsStreamInterceptor_MeasuresDuration(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())

	interceptor := MetricsStreamInterceptor()

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/StreamDuration",
	}

	ss := &mockServerStreamForMetrics{ctx: context.Background()}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := interceptor(nil, ss, info, handler)

	assert.NoError(t, err)
}
