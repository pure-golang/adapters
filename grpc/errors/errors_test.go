package errors

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestFromError_Nil(t *testing.T) {
	// Test FromError with nil error
	err := FromError(nil)
	assert.Nil(t, err, "FromError(nil) should return nil")
}

func TestFromError_ContextCanceled(t *testing.T) {
	// Test FromError with context.Canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := FromError(ctx.Err())
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Equal(t, codes.Canceled, st.Code(), "code should be Canceled")
	assert.Equal(t, "request canceled", st.Message(), "message should match")
}

func TestFromError_ContextDeadlineExceeded(t *testing.T) {
	// Test FromError with context.DeadlineExceeded
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	<-ctx.Done()
	cancel()

	err := FromError(ctx.Err())
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Equal(t, codes.DeadlineExceeded, st.Code(), "code should be DeadlineExceeded")
	assert.Equal(t, "deadline exceeded", st.Message(), "message should match")
}

func TestFromError_ExistingGRPCStatus(t *testing.T) {
	// Test FromError with existing gRPC status (preserves it)
	originalErr := status.Error(codes.NotFound, "resource not found")
	err := FromError(originalErr)

	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Equal(t, codes.NotFound, st.Code(), "code should be preserved")
	assert.Equal(t, "resource not found", st.Message(), "message should be preserved")
	assert.Same(t, originalErr, err, "should return the same error instance")
}

func TestFromError_GenericError(t *testing.T) {
	// Test FromError with generic error (converts to Internal)
	genericErr := errors.New("something went wrong")
	err := FromError(genericErr)

	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Equal(t, codes.Internal, st.Code(), "code should be Internal")
	assert.Equal(t, "something went wrong", st.Message(), "message should match original error")
}

func TestFromError_WrappedContextErrors(t *testing.T) {
	// Test FromError with wrapped context errors
	wrappedCanceled := fmt.Errorf("wrapped: %w", context.Canceled)
	err := FromError(wrappedCanceled)

	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Equal(t, codes.Canceled, st.Code(), "wrapped canceled should be detected")

	wrappedDeadline := fmt.Errorf("operation failed: %w", context.DeadlineExceeded)
	err = FromError(wrappedDeadline)

	st, ok = status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Equal(t, codes.DeadlineExceeded, st.Code(), "wrapped deadline exceeded should be detected")
}

func TestWrapError_Nil(t *testing.T) {
	// Test WrapError with nil error
	err := WrapError(nil, codes.Internal, "wrapped message")
	assert.Nil(t, err, "WrapError with nil error should return nil")
}

func TestWrapError_ValidError(t *testing.T) {
	// Test WrapError with valid error
	originalErr := errors.New("base error")
	err := WrapError(originalErr, codes.NotFound, "resource lookup failed")

	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Equal(t, codes.NotFound, st.Code(), "code should be NotFound")
	assert.Contains(t, st.Message(), "resource lookup failed", "message should contain wrapper")
	assert.Contains(t, st.Message(), "base error", "message should contain original error")
}

func TestWrapError_MessageFormatting(t *testing.T) {
	// Test WrapError message formatting
	tests := []struct {
		name        string
		baseErr     error
		code        codes.Code
		msg         string
		expectedMsg string
	}{
		{
			name:        "simple wrap",
			baseErr:     errors.New("inner error"),
			code:        codes.Internal,
			msg:         "operation failed",
			expectedMsg: "operation failed: inner error",
		},
		{
			name:        "empty prefix",
			baseErr:     errors.New("db error"),
			code:        codes.Aborted,
			msg:         "",
			expectedMsg: ": db error",
		},
		{
			name:        "multi-word prefix",
			baseErr:     errors.New("connection lost"),
			code:        codes.Unavailable,
			msg:         "database operation unsuccessful due to",
			expectedMsg: "database operation unsuccessful due to: connection lost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapError(tt.baseErr, tt.code, tt.msg)
			require.Error(t, err)

			st, ok := status.FromError(err)
			require.True(t, ok, "error should be a gRPC status")
			assert.Equal(t, tt.code, st.Code(), "code should match")
			assert.Equal(t, tt.expectedMsg, st.Message(), "formatted message should match")
		})
	}
}

func TestWrapError_DifferentCodes(t *testing.T) {
	// Test WrapError with different status codes
	codesList := []codes.Code{
		codes.Canceled,
		codes.Unknown,
		codes.InvalidArgument,
		codes.DeadlineExceeded,
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

	baseErr := errors.New("test error")
	for _, code := range codesList {
		t.Run(code.String(), func(t *testing.T) {
			err := WrapError(baseErr, code, "wrapped")
			require.Error(t, err)

			st, ok := status.FromError(err)
			require.True(t, ok, "error should be a gRPC status")
			assert.Equal(t, code, st.Code(), "code should be %s", code)
		})
	}
}

func TestWrapError_OKCode(t *testing.T) {
	// Test WrapError with codes.OK - gRPC returns nil for OK
	baseErr := errors.New("test error")
	err := WrapError(baseErr, codes.OK, "wrapped")
	// codes.OK results in nil error from status.Errorf
	assert.Nil(t, err, "codes.OK should result in nil error")
}

func TestNewError_DifferentCodes(t *testing.T) {
	// Test NewError with different codes
	tests := []struct {
		name        string
		code        codes.Code
		msg         string
		expectedMsg string
	}{
		{
			name:        "not found",
			code:        codes.NotFound,
			msg:         "user not found",
			expectedMsg: "user not found",
		},
		{
			name:        "invalid argument",
			code:        codes.InvalidArgument,
			msg:         "invalid email format",
			expectedMsg: "invalid email format",
		},
		{
			name:        "permission denied",
			code:        codes.PermissionDenied,
			msg:         "access denied",
			expectedMsg: "access denied",
		},
		{
			name:        "internal",
			code:        codes.Internal,
			msg:         "database connection failed",
			expectedMsg: "database connection failed",
		},
		{
			name:        "unavailable",
			code:        codes.Unavailable,
			msg:         "service temporarily unavailable",
			expectedMsg: "service temporarily unavailable",
		},
		{
			name:        "empty message",
			code:        codes.Unknown,
			msg:         "",
			expectedMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, tt.msg)
			require.Error(t, err)

			st, ok := status.FromError(err)
			require.True(t, ok, "error should be a gRPC status")
			assert.Equal(t, tt.code, st.Code(), "code should match")
			assert.Equal(t, tt.expectedMsg, st.Message(), "message should match")
		})
	}
}

func TestNewError_AllCodes(t *testing.T) {
	// Test NewError with all possible gRPC codes
	codesList := []codes.Code{
		codes.Canceled,
		codes.Unknown,
		codes.InvalidArgument,
		codes.DeadlineExceeded,
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

	for _, code := range codesList {
		t.Run(code.String(), func(t *testing.T) {
			err := NewError(code, "test message")
			require.Error(t, err)

			st, ok := status.FromError(err)
			require.True(t, ok, "error should be a gRPC status")
			assert.Equal(t, code, st.Code(), "code should be %s", code)
			assert.Equal(t, "test message", st.Message(), "message should match")
		})
	}
}

func TestNewError_OKCode(t *testing.T) {
	// Test NewError with codes.OK - gRPC returns nil for OK
	err := NewError(codes.OK, "test message")
	// codes.OK results in nil error from status.Error
	assert.Nil(t, err, "codes.OK should result in nil error")
}

func TestIntegration_FromErrorRoundtrip(t *testing.T) {
	// Test that errors created by NewError are handled correctly by FromError
	originalErr := NewError(codes.NotFound, "item not found")
	resultErr := FromError(originalErr)

	// FromError should preserve gRPC status errors
	st, ok := status.FromError(resultErr)
	require.True(t, ok, "result should be a gRPC status")
	assert.Equal(t, codes.NotFound, st.Code(), "code should be preserved")
	assert.Equal(t, "item not found", st.Message(), "message should be preserved")
	assert.Same(t, originalErr, resultErr, "should return same error instance")
}

func TestIntegration_WrapAndFrom(t *testing.T) {
	// Test wrapping an error then passing through FromError
	baseErr := errors.New("connection failed")
	wrappedErr := WrapError(baseErr, codes.Unavailable, "database error")
	resultErr := FromError(wrappedErr)

	// FromError should preserve the wrapped gRPC status
	st, ok := status.FromError(resultErr)
	require.True(t, ok, "result should be a gRPC status")
	assert.Equal(t, codes.Unavailable, st.Code(), "code should be Unavailable")
	assert.Contains(t, st.Message(), "database error", "should contain wrap message")
	assert.Contains(t, st.Message(), "connection failed", "should contain base error")
}
