package minio

import (
	"strings"

	"github.com/pure-golang/adapters/storage"
)

// toStorageError converts minio errors to storage errors.
func toStorageError(err error, bucket, key string) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for specific S3 error types by error message
	// Order matters: check for more specific patterns first
	switch {
	case strings.Contains(errMsg, "NoSuchBucket"):
		return &storage.StorageError{
			Code:    storage.CodeBucketNotFound,
			Message: "bucket not found",
			Err:     err,
			Bucket:  bucket,
			Key:     key,
		}
	case strings.Contains(errMsg, "bucket") && (strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "does not exist")):
		// Error mentions both bucket and not found/does not exist
		return &storage.StorageError{
			Code:    storage.CodeBucketNotFound,
			Message: "bucket not found",
			Err:     err,
			Bucket:  bucket,
			Key:     key,
		}
	case strings.Contains(errMsg, "NoSuchKey") ||
		strings.Contains(errMsg, "NotFound"):
		return &storage.StorageError{
			Code:    storage.CodeNotFound,
			Message: "object not found",
			Err:     err,
			Bucket:  bucket,
			Key:     key,
		}
	case strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "does not exist"):
		return &storage.StorageError{
			Code:    storage.CodeNotFound,
			Message: "object not found",
			Err:     err,
			Bucket:  bucket,
			Key:     key,
		}
	case strings.Contains(errMsg, "AccessDenied") ||
		strings.Contains(errMsg, "Forbidden"):
		return &storage.StorageError{
			Code:    storage.CodeAccessDenied,
			Message: "access denied",
			Err:     err,
			Bucket:  bucket,
			Key:     key,
		}
	}

	// Generic error wrapping
	return &storage.StorageError{
		Code:    storage.CodeInternalError,
		Message: "internal storage error",
		Err:     err,
		Bucket:  bucket,
		Key:     key,
	}
}

// isNotFoundError checks if error is a "not found" type error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	return strings.Contains(errMsg, "NoSuchKey") ||
		strings.Contains(errMsg, "NotFound") ||
		strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "does not exist")
}
