package storage

import (
	"errors"
	"fmt"
)

// Common storage errors.
var (
	ErrNotFound       = errors.New("object not found")
	ErrAccessDenied   = errors.New("access denied")
	ErrBucketNotFound = errors.New("bucket not found")
)

// ErrorCode represents a storage error code.
type ErrorCode string

const (
	CodeNotFound       ErrorCode = "NotFound"
	CodeAccessDenied   ErrorCode = "AccessDenied"
	CodeBucketNotFound ErrorCode = "BucketNotFound"
	CodeInternalError  ErrorCode = "InternalError"
)

// StorageError wraps storage operation errors.
type StorageError struct {
	Code    ErrorCode
	Message string
	Err     error
	Bucket  string
	Key     string
}

func (e *StorageError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("storage.%s: %s (bucket=%s, key=%s): %v", e.Code, e.Message, e.Bucket, e.Key, e.Err)
	}
	return fmt.Sprintf("storage.%s: %s (bucket=%s, key=%s)", e.Code, e.Message, e.Bucket, e.Key)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

// IsNotFound checks if error is a "not found" error.
func IsNotFound(err error) bool {
	var storageErr *StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Code == CodeNotFound
	}
	return errors.Is(err, ErrNotFound)
}

// IsAccessDenied checks if error is an "access denied" error.
func IsAccessDenied(err error) bool {
	var storageErr *StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Code == CodeAccessDenied
	}
	return errors.Is(err, ErrAccessDenied)
}

// IsBucketNotFound checks if error is a "bucket not found" error.
func IsBucketNotFound(err error) bool {
	var storageErr *StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Code == CodeBucketNotFound
	}
	return errors.Is(err, ErrBucketNotFound)
}
