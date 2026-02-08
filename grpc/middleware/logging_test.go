package middleware

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

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

// mockHandler is a simple handler that returns a response and error
type mockHandler struct {
	resp interface{}
	err  error
}

func (m *mockHandler) handle(ctx context.Context, req interface{}) (interface{}, error) {
	return m.resp, m.err
}

// TestLoggingInterceptor_Success tests successful request logging
func TestLoggingInterceptor_Success(t *testing.T) {
	// Create a logger that captures log output
	var logAttrs []slog.Attr
	logger := slog.New(&attrHandler{attrs: &logAttrs})

	interceptor := LoggingInterceptor(logger)

	handlerCalled := false
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "success", nil
	}

	resp, err := interceptor(context.Background(), "request", info, handler)

	require.True(t, handlerCalled)
	assert.Equal(t, "success", resp)
	assert.NoError(t, err)

	// Verify log attributes were captured
	assert.Greater(t, len(logAttrs), 0, "Expected log attributes to be captured")
}

// TestLoggingInterceptor_WithError tests error request logging
func TestLoggingInterceptor_WithError(t *testing.T) {
	var logAttrs []slog.Attr
	logger := slog.New(&attrHandler{attrs: &logAttrs})

	interceptor := LoggingInterceptor(logger)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/TestMethod",
	}

	expectedErr := status.Error(codes.NotFound, "resource not found")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	resp, err := interceptor(context.Background(), "request", info, handler)

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))

	// Verify error was logged
	assert.Greater(t, len(logAttrs), 0, "Expected log attributes to be captured")

	// Check for error-related attributes
	foundErrorCode := false
	for _, attr := range logAttrs {
		if attr.Key == "status_code" {
			foundErrorCode = true
			assert.Equal(t, "NotFound", attr.Value.String())
		}
	}
	assert.True(t, foundErrorCode, "Expected status_code attribute in log")
}

// TestLoggingInterceptor_WithPanic tests panic recovery via RecoveryInterceptor
func TestLoggingInterceptor_WithPanic(t *testing.T) {
	var logAttrs []slog.Attr
	logger := slog.New(&attrHandler{attrs: &logAttrs})

	// Chain RecoveryInterceptor before LoggingInterceptor
	recovery := RecoveryInterceptor(logger)
	logging := LoggingInterceptor(logger)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/PanicMethod",
	}

	panicMsg := "test panic"
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic(panicMsg)
	}

	// First test recovery alone
	t.Run("RecoveryInterceptor catches panic", func(t *testing.T) {
		resp, err := recovery(context.Background(), "request", info, handler)

		assert.Nil(t, resp)
		assert.Error(t, err)

		// Status code should be UNAVAILABLE (14)
		st := status.Convert(err)
		assert.Equal(t, codes.Unavailable, st.Code())
	})

	// Then test combined with logging
	t.Run("Recovery followed by Logging", func(t *testing.T) {
		comboHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			// Recovery interceptor returns error, logging logs it
			return recovery(ctx, "request", info, handler)
		}

		logAttrs = logAttrs[:0] // Clear previous logs
		resp, err := logging(context.Background(), "request", info, comboHandler)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})
}

// TestLoggingInterceptor_DifferentErrorCodes tests logging with various gRPC status codes
func TestLoggingInterceptor_DifferentErrorCodes(t *testing.T) {
	testCases := []struct {
		name       string
		code       codes.Code
		err        error
		wantLogged bool
	}{
		{
			name:       "InvalidArgument",
			code:       codes.InvalidArgument,
			err:        status.Error(codes.InvalidArgument, "invalid argument"),
			wantLogged: true,
		},
		{
			name:       "PermissionDenied",
			code:       codes.PermissionDenied,
			err:        status.Error(codes.PermissionDenied, "permission denied"),
			wantLogged: true,
		},
		{
			name:       "Unauthenticated",
			code:       codes.Unauthenticated,
			err:        status.Error(codes.Unauthenticated, "unauthenticated"),
			wantLogged: true,
		},
		{
			name:       "Internal",
			code:       codes.Internal,
			err:        status.Error(codes.Internal, "internal error"),
			wantLogged: true,
		},
		{
			name:       "Unavailable",
			code:       codes.Unavailable,
			err:        status.Error(codes.Unavailable, "unavailable"),
			wantLogged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var logAttrs []slog.Attr
			logger := slog.New(&attrHandler{attrs: &logAttrs})

			interceptor := LoggingInterceptor(logger)

			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.service/TestMethod",
			}

			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, tc.err
			}

			resp, err := interceptor(context.Background(), "request", info, handler)

			assert.Nil(t, resp)
			assert.Error(t, err)
			assert.Equal(t, tc.code, status.Code(err))
		})
	}
}

// TestRecoveryInterceptor_CatchesPanic tests that RecoveryInterceptor catches panics
func TestRecoveryInterceptor_CatchesPanic(t *testing.T) {
	var logAttrs []slog.Attr
	logger := slog.New(&attrHandler{attrs: &logAttrs})

	interceptor := RecoveryInterceptor(logger)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/PanicMethod",
	}

	// Test with string panic
	t.Run("string panic", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			panic("panic string")
		}

		resp, err := interceptor(context.Background(), "request", info, handler)

		assert.Nil(t, resp)
		assert.Error(t, err)

		st := status.Convert(err)
		assert.Equal(t, codes.Unavailable, st.Code())
		assert.Contains(t, st.Message(), "internal server error")
	})

	// Test with error panic
	t.Run("error panic", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			panic(errors.New("panic error"))
		}

		resp, err := interceptor(context.Background(), "request", info, handler)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
	})

	// Test with int panic
	t.Run("int panic", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			panic(42)
		}

		resp, err := interceptor(context.Background(), "request", info, handler)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
	})

	// Test no panic
	t.Run("no panic", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "success", nil
		}

		resp, err := interceptor(context.Background(), "request", info, handler)

		assert.Equal(t, "success", resp)
		assert.NoError(t, err)
	})
}

// mockServerStream is a mock implementation of grpc.ServerStream for testing
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

// TestLoggingStreamInterceptor_Success tests successful stream logging
func TestLoggingStreamInterceptor_Success(t *testing.T) {
	var logAttrs []slog.Attr
	logger := slog.New(&attrHandler{attrs: &logAttrs})

	interceptor := LoggingStreamInterceptor(logger)

	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.service/TestStream",
		IsClientStream: true,
		IsServerStream: false,
	}

	ss := &mockServerStream{ctx: context.Background()}

	handlerCalled := false
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	err := interceptor(nil, ss, info, handler)

	assert.True(t, handlerCalled)
	assert.NoError(t, err)
	assert.Greater(t, len(logAttrs), 0, "Expected log attributes to be captured")
}

// TestLoggingStreamInterceptor_WithError tests error stream logging
func TestLoggingStreamInterceptor_WithError(t *testing.T) {
	var logAttrs []slog.Attr
	logger := slog.New(&attrHandler{attrs: &logAttrs})

	interceptor := LoggingStreamInterceptor(logger)

	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.service/TestStream",
		IsClientStream: false,
		IsServerStream: true,
	}

	ss := &mockServerStream{ctx: context.Background()}

	expectedErr := status.Error(codes.Aborted, "stream aborted")
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return expectedErr
	}

	err := interceptor(nil, ss, info, handler)

	assert.Error(t, err)
	assert.Equal(t, codes.Aborted, status.Code(err))

	// Verify error was logged
	assert.Greater(t, len(logAttrs), 0, "Expected log attributes to be captured")
}

// TestLoggingStreamInterceptor_StreamTypes tests logging with different stream types
func TestLoggingStreamInterceptor_StreamTypes(t *testing.T) {
	var logAttrs []slog.Attr
	logger := slog.New(&attrHandler{attrs: &logAttrs})

	interceptor := LoggingStreamInterceptor(logger)

	testCases := []struct {
		name string
		info *grpc.StreamServerInfo
	}{
		{
			name: "client streaming",
			info: &grpc.StreamServerInfo{
				FullMethod:     "/test.service/ClientStream",
				IsClientStream: true,
				IsServerStream: false,
			},
		},
		{
			name: "server streaming",
			info: &grpc.StreamServerInfo{
				FullMethod:     "/test.service/ServerStream",
				IsClientStream: false,
				IsServerStream: true,
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
			logAttrs = logAttrs[:0] // Clear logs

			ss := &mockServerStream{ctx: context.Background()}

			handler := func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			}

			err := interceptor(nil, ss, tc.info, handler)
			assert.NoError(t, err)
		})
	}
}

// TestRecoveryStreamInterceptor_CatchesPanic tests that RecoveryStreamInterceptor catches panics
func TestRecoveryStreamInterceptor_CatchesPanic(t *testing.T) {
	var logAttrs []slog.Attr
	logger := slog.New(&attrHandler{attrs: &logAttrs})

	interceptor := RecoveryStreamInterceptor(logger)

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/PanicStream",
	}

	ss := &mockServerStream{ctx: context.Background()}

	// Test with string panic
	t.Run("string panic", func(t *testing.T) {
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			panic("stream panic")
		}

		err := interceptor(nil, ss, info, handler)

		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.Unavailable, st.Code())
	})

	// Test with error panic
	t.Run("error panic", func(t *testing.T) {
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			panic(errors.New("stream error panic"))
		}

		err := interceptor(nil, ss, info, handler)

		assert.Error(t, err)
		assert.Equal(t, codes.Unavailable, status.Code(err))
	})

	// Test no panic
	t.Run("no panic", func(t *testing.T) {
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			return nil
		}

		err := interceptor(nil, ss, info, handler)

		assert.NoError(t, err)
	})
}

// TestLoggingInterceptor_Context tests that context is properly passed through
func TestLoggingInterceptor_Context(t *testing.T) {
	logger := noop.NewNoop().With("test", "value")
	interceptor := LoggingInterceptor(logger)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/TestMethod",
	}

	ctxWithValue := context.WithValue(context.Background(), "test_key", "test_value")
	var capturedCtx context.Context

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedCtx = ctx
		return "success", nil
	}

	interceptor(ctxWithValue, "request", info, handler)

	// Verify context was passed through
	assert.Equal(t, "test_value", capturedCtx.Value("test_key"))
}

// TestLoggingStreamInterceptor_Context tests that context is properly passed through streams
func TestLoggingStreamInterceptor_Context(t *testing.T) {
	logger := noop.NewNoop()
	interceptor := LoggingStreamInterceptor(logger)

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.service/TestStream",
	}

	ctxWithValue := context.WithValue(context.Background(), "stream_key", "stream_value")
	ss := &mockServerStream{ctx: ctxWithValue}

	var capturedCtx context.Context

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		capturedCtx = stream.Context()
		return nil
	}

	interceptor(nil, ss, info, handler)

	// Verify context was passed through
	assert.Equal(t, "stream_value", capturedCtx.Value("stream_key"))
}

// TestLoggingInterceptor_WithTextLogger tests that interceptor works with text logger
func TestLoggingInterceptor_WithTextLogger(t *testing.T) {
	// Use text logger for simpler testing
	var buf []byte
	logger := slog.New(slog.NewTextHandler(&bufWriter{&buf}, nil))

	interceptor := LoggingInterceptor(logger)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	// Should not panic
	resp, err := interceptor(context.Background(), "request", info, handler)

	assert.Equal(t, "success", resp)
	assert.NoError(t, err)
}

// bufWriter is a simple writer that appends to a byte slice
type bufWriter struct {
	buf *[]byte
}

func (b *bufWriter) Write(p []byte) (n int, err error) {
	*b.buf = append(*b.buf, p...)
	return len(p), nil
}

// attrHandler is a test handler that captures log attributes
type attrHandler struct {
	attrs *[]slog.Attr
}

func (h *attrHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Attrs(func(a slog.Attr) bool {
		*h.attrs = append(*h.attrs, a)
		return true
	})
	return nil
}

func (h *attrHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *attrHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *attrHandler) WithGroup(name string) slog.Handler {
	return h
}

// TestLoggingInterceptor_WithBufConn tests using buffer connection for gRPC testing
func TestLoggingInterceptor_WithBufConn(t *testing.T) {
	logger := noop.NewNoop()
	interceptor := LoggingInterceptor(logger)

	// Create a buffer listener
	lis := bufconn.Listen(1024)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.service/BufConnTest",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "bufconn_success", nil
	}

	resp, err := interceptor(context.Background(), "request", info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "bufconn_success", resp)

	lis.Close() // Clean up
}
