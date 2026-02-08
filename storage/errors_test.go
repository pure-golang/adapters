package storage

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStorageError_Error tests the Error method of StorageError.
func TestStorageError_Error(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		baseErr := errors.New("underlying error")
		err := &StorageError{
			Code:    CodeNotFound,
			Message: "object not found",
			Err:     baseErr,
			Bucket:  "my-bucket",
			Key:     "my-key",
		}

		expected := "storage.NotFound: object not found (bucket=my-bucket, key=my-key): underlying error"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("without wrapped error", func(t *testing.T) {
		err := &StorageError{
			Code:    CodeAccessDenied,
			Message: "access is denied",
			Bucket:  "test-bucket",
			Key:     "test-key",
		}

		expected := "storage.AccessDenied: access is denied (bucket=test-bucket, key=test-key)"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("with empty bucket and key", func(t *testing.T) {
		err := &StorageError{
			Code:    CodeInternalError,
			Message: "internal error occurred",
		}

		expected := "storage.InternalError: internal error occurred (bucket=, key=)"
		assert.Equal(t, expected, err.Error())
	})
}

// TestStorageError_Unwrap tests the Unwrap method of StorageError.
func TestStorageError_Unwrap(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := &StorageError{
			Code: CodeNotFound,
			Err:  baseErr,
		}

		unwrapped := errors.Unwrap(err)
		assert.Same(t, baseErr, unwrapped)
	})

	t.Run("without wrapped error", func(t *testing.T) {
		err := &StorageError{
			Code: CodeNotFound,
		}

		unwrapped := errors.Unwrap(err)
		assert.Nil(t, unwrapped)
	})
}

// TestIsNotFound tests the IsNotFound helper function.
func TestIsNotFound(t *testing.T) {
	t.Run("returns true for NotFound StorageError", func(t *testing.T) {
		err := &StorageError{
			Code: CodeNotFound,
		}

		assert.True(t, IsNotFound(err))
	})

	t.Run("returns false for other StorageError codes", func(t *testing.T) {
		err := &StorageError{
			Code: CodeAccessDenied,
		}

		assert.False(t, IsNotFound(err))
	})

	t.Run("returns true for ErrNotFound", func(t *testing.T) {
		assert.True(t, IsNotFound(ErrNotFound))
	})

	t.Run("returns true for wrapped ErrNotFound", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", ErrNotFound)
		assert.True(t, IsNotFound(err))
	})

	t.Run("returns false for generic error", func(t *testing.T) {
		err := errors.New("some other error")
		assert.False(t, IsNotFound(err))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, IsNotFound(nil))
	})
}

// TestIsAccessDenied tests the IsAccessDenied helper function.
func TestIsAccessDenied(t *testing.T) {
	t.Run("returns true for AccessDenied StorageError", func(t *testing.T) {
		err := &StorageError{
			Code: CodeAccessDenied,
		}

		assert.True(t, IsAccessDenied(err))
	})

	t.Run("returns false for other StorageError codes", func(t *testing.T) {
		err := &StorageError{
			Code: CodeNotFound,
		}

		assert.False(t, IsAccessDenied(err))
	})

	t.Run("returns true for ErrAccessDenied", func(t *testing.T) {
		assert.True(t, IsAccessDenied(ErrAccessDenied))
	})

	t.Run("returns true for wrapped ErrAccessDenied", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", ErrAccessDenied)
		assert.True(t, IsAccessDenied(err))
	})

	t.Run("returns false for generic error", func(t *testing.T) {
		err := errors.New("some other error")
		assert.False(t, IsAccessDenied(err))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, IsAccessDenied(nil))
	})
}

// TestIsBucketNotFound tests the IsBucketNotFound helper function.
func TestIsBucketNotFound(t *testing.T) {
	t.Run("returns true for BucketNotFound StorageError", func(t *testing.T) {
		err := &StorageError{
			Code: CodeBucketNotFound,
		}

		assert.True(t, IsBucketNotFound(err))
	})

	t.Run("returns false for other StorageError codes", func(t *testing.T) {
		err := &StorageError{
			Code: CodeNotFound,
		}

		assert.False(t, IsBucketNotFound(err))
	})

	t.Run("returns true for ErrBucketNotFound", func(t *testing.T) {
		assert.True(t, IsBucketNotFound(ErrBucketNotFound))
	})

	t.Run("returns true for wrapped ErrBucketNotFound", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", ErrBucketNotFound)
		assert.True(t, IsBucketNotFound(err))
	})

	t.Run("returns false for generic error", func(t *testing.T) {
		err := errors.New("some other error")
		assert.False(t, IsBucketNotFound(err))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, IsBucketNotFound(nil))
	})
}

// TestErrorCode_values tests that ErrorCode constants have expected values.
func TestErrorCode_values(t *testing.T) {
	assert.Equal(t, ErrorCode("NotFound"), CodeNotFound)
	assert.Equal(t, ErrorCode("AccessDenied"), CodeAccessDenied)
	assert.Equal(t, ErrorCode("BucketNotFound"), CodeBucketNotFound)
	assert.Equal(t, ErrorCode("InternalError"), CodeInternalError)
}

// TestNewStorageError tests creating StorageError instances.
func TestNewStorageError(t *testing.T) {
	err := &StorageError{
		Code:    CodeNotFound,
		Message: "test message",
		Err:     errors.New("test"),
		Bucket:  "bucket",
		Key:     "key",
	}

	assert.Equal(t, CodeNotFound, err.Code)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, "bucket", err.Bucket)
	assert.Equal(t, "key", err.Key)
	assert.NotNil(t, err.Err)
}

// TestStorageError_As tests errors.As with StorageError.
func TestStorageError_As(t *testing.T) {
	t.Run("can extract StorageError from wrapped error", func(t *testing.T) {
		storageErr := &StorageError{
			Code:    CodeNotFound,
			Message: "not found",
		}
		wrappedErr := fmt.Errorf("wrapped: %w", storageErr)

		var extracted *StorageError
		assert.True(t, errors.As(wrappedErr, &extracted))
		assert.Equal(t, CodeNotFound, extracted.Code)
		assert.Equal(t, "not found", extracted.Message)
	})

	t.Run("returns false for non-StorageError", func(t *testing.T) {
		err := errors.New("plain error")
		var storageErr *StorageError
		assert.False(t, errors.As(err, &storageErr))
		assert.Nil(t, storageErr)
	})
}
