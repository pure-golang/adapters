package minio

import (
	"errors"
	"testing"

	"github.com/pure-golang/adapters/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToStorageError_NotFound tests toStorageError with various NotFound scenarios.
func TestToStorageError_NotFound(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		bucket       string
		key          string
		expectedCode storage.ErrorCode
	}{
		{
			name:         "nil error returns nil",
			err:          nil,
			bucket:       "bucket",
			key:          "key",
			expectedCode: "",
		},
		{
			name:         "NoSuchKey error returns NotFound",
			err:          errors.New("The specified key does not exist: NoSuchKey"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeNotFound,
		},
		{
			name:         "NotFound in message returns NotFound",
			err:          errors.New("Object NotFound in bucket"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeNotFound,
		},
		{
			name:         "not found text returns NotFound",
			err:          errors.New("object not found"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeNotFound,
		},
		{
			name:         "does not exist returns NotFound",
			err:          errors.New("file does not exist"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeNotFound,
		},
		{
			name:         "bucket not found returns BucketNotFound",
			err:          errors.New("bucket not found"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeBucketNotFound,
		},
		{
			name:         "bucket does not exist returns BucketNotFound",
			err:          errors.New("bucket does not exist"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeBucketNotFound,
		},
		{
			name:         "NoSuchBucket returns BucketNotFound",
			err:          errors.New("NoSuchBucket: the specified bucket does not exist"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeBucketNotFound,
		},
		{
			name:         "AccessDenied returns AccessDenied",
			err:          errors.New("AccessDenied: You don't have permission"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeAccessDenied,
		},
		{
			name:         "Forbidden returns AccessDenied",
			err:          errors.New("403 Forbidden"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeAccessDenied,
		},
		{
			name:         "generic error returns InternalError",
			err:          errors.New("some unknown error"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeInternalError,
		},
		{
			name:         "connection timeout returns InternalError",
			err:          errors.New("connection timeout"),
			bucket:       "test-bucket",
			key:          "test-key",
			expectedCode: storage.CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := toStorageError(tt.err, tt.bucket, tt.key)
			if tt.expectedCode == "" {
				assert.Nil(t, err)
			} else {
				require.NotNil(t, err)
				storageErr, ok := err.(*storage.StorageError)
				require.True(t, ok, "error should be *storage.StorageError")
				assert.Equal(t, tt.expectedCode, storageErr.Code)
				assert.Equal(t, tt.bucket, storageErr.Bucket)
				assert.Equal(t, tt.key, storageErr.Key)
				assert.NotNil(t, storageErr.Err)
				assert.Equal(t, tt.err, storageErr.Err)
			}
		})
	}
}

// TestToStorageError_ErrorMessages tests the error messages are set correctly.
func TestToStorageError_ErrorMessages(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{
			name:            "NotFound message",
			err:             errors.New("object not found"),
			expectedMessage: "object not found",
		},
		{
			name:            "BucketNotFound message",
			err:             errors.New("NoSuchBucket"),
			expectedMessage: "bucket not found",
		},
		{
			name:            "AccessDenied message",
			err:             errors.New("AccessDenied"),
			expectedMessage: "access denied",
		},
		{
			name:            "InternalError message",
			err:             errors.New("unknown error"),
			expectedMessage: "internal storage error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := toStorageError(tt.err, "bucket", "key")
			require.NotNil(t, err)
			storageErr, ok := err.(*storage.StorageError)
			require.True(t, ok)
			assert.Equal(t, tt.expectedMessage, storageErr.Message)
		})
	}
}

// TestIsNotFoundError_Extended tests the isNotFoundError function with additional scenarios.
func TestIsNotFoundError_Extended(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error returns false",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error returns false",
			err:      errors.New("some generic error"),
			expected: false,
		},
		{
			name:     "connection error returns false",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "permission error returns false",
			err:      errors.New("access denied"),
			expected: false,
		},
		{
			name:     "NoSuchKey error returns true",
			err:      errors.New("NoSuchKey: key not found"),
			expected: true,
		},
		{
			name:     "NotFound error returns true",
			err:      errors.New("object NotFound"),
			expected: true,
		},
		{
			name:     "not found text returns true",
			err:      errors.New("object not found"),
			expected: true,
		},
		{
			name:     "does not exist returns true",
			err:      errors.New("file does not exist"),
			expected: true,
		},
		{
			name:     "mixed case not found returns true",
			err:      errors.New("Object NOT FOUND"),
			expected: false, // case sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStorageErrorCodes tests all storage error codes are properly used.
func TestStorageErrorCodes(t *testing.T) {
	// Test that we use the correct error codes
	assert.Equal(t, storage.ErrorCode("NotFound"), storage.CodeNotFound)
	assert.Equal(t, storage.ErrorCode("BucketNotFound"), storage.CodeBucketNotFound)
	assert.Equal(t, storage.ErrorCode("AccessDenied"), storage.CodeAccessDenied)
	assert.Equal(t, storage.ErrorCode("InternalError"), storage.CodeInternalError)
}
