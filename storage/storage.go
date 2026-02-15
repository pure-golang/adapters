package storage

import (
	"context"
	"io"
	"time"
)

// ObjectInfo represents metadata about a stored object.
type ObjectInfo struct {
	Key          string            // Object key/path
	Size         int64             // Object size in bytes
	LastModified time.Time         // Last modification time
	ETag         string            // Entity tag for versioning
	ContentType  string            // Content type
	Metadata     map[string]string // User-defined metadata
}

// PutOptions contains optional parameters for Put operation.
type PutOptions struct {
	ContentType string            // MIME type
	Metadata    map[string]string // User metadata
}

// ListOptions contains optional parameters for List operation.
type ListOptions struct {
	Prefix    string // Object key prefix
	Recursive bool   // Whether to list recursively
	MaxKeys   int    // Maximum number of keys to return
}

// ListResult contains the result of a List operation.
type ListResult struct {
	Objects     []ObjectInfo // List of objects
	IsTruncated bool         // Whether more results are available
}

// PresignedURLOptions contains options for generating presigned URLs.
type PresignedURLOptions struct {
	Method string        // HTTP method (GET, PUT, DELETE)
	Expiry time.Duration // URL expiration time
}

// MultipartUpload represents an active multipart upload.
type MultipartUpload struct {
	UploadID  string    // Unique upload ID
	Key       string    // Object key
	Bucket    string    // Bucket name
	Initiated time.Time // Upload initiation time
}

// UploadedPart represents a successfully uploaded part.
type UploadedPart struct {
	PartNumber int32  // Part number
	ETag       string // ETag of the uploaded part
	Size       int64  // Size of the part
}

// CompleteMultipartUploadOptions contains options for completing multipart upload.
type CompleteMultipartUploadOptions struct {
	Parts []UploadedPart // List of uploaded parts in order
}

// Storage is the interface for object storage operations.
type Storage interface {
	// Put stores an object in the specified bucket.
	Put(ctx context.Context, bucket, key string, reader io.Reader, opts *PutOptions) error

	// Get retrieves an object from the specified bucket.
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, *ObjectInfo, error)

	// Delete removes an object from the specified bucket.
	Delete(ctx context.Context, bucket, key string) error

	// Exists checks if an object exists in the specified bucket.
	Exists(ctx context.Context, bucket, key string) (bool, error)

	// List lists objects in the specified bucket with optional prefix.
	List(ctx context.Context, bucket string, opts *ListOptions) (*ListResult, error)

	// GetPresignedURL generates a presigned URL for direct access.
	GetPresignedURL(ctx context.Context, bucket, key string, opts *PresignedURLOptions) (string, error)

	// GetFileHeader retrieves the first 4096 bytes of an object using range request.
	GetFileHeader(ctx context.Context, bucket, key string) ([]byte, error)

	// CreateMultipartUpload initiates a multipart upload.
	CreateMultipartUpload(ctx context.Context, bucket, key string, opts *PutOptions) (*MultipartUpload, error)

	// UploadPart uploads a part in a multipart upload.
	UploadPart(ctx context.Context, bucket, key, uploadID string, partNumber int32, reader io.Reader) (*UploadedPart, error)

	// CompleteMultipartUpload completes a multipart upload.
	CompleteMultipartUpload(ctx context.Context, bucket, key, uploadID string, opts *CompleteMultipartUploadOptions) (*ObjectInfo, error)

	// AbortMultipartUpload aborts a multipart upload.
	AbortMultipartUpload(ctx context.Context, bucket, key, uploadID string) error

	// ListMultipartUploads lists active multipart uploads.
	ListMultipartUploads(ctx context.Context, bucket string) ([]MultipartUpload, error)

	io.Closer
}
