package minio

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"github.com/pure-golang/adapters/storage"
	"github.com/minio/minio-go/v7"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ storage.Storage = (*Storage)(nil)

var tracer = otel.Tracer("github.com/pure-golang/adapters/storage/s3")

// Storage implements storage.Storage interface for S3-compatible storage.
// Supports MinIO, Yandex Cloud Storage, AWS S3, and other S3-compatible providers.
type Storage struct {
	client *Client
	cfg    Config
	logger *slog.Logger
}

// StorageOptions contains options for Storage creation.
type StorageOptions struct {
	Logger *slog.Logger
}

// NewStorage creates a new S3 Storage instance.
func NewStorage(client *Client, opts *StorageOptions) *Storage {
	if opts == nil {
		opts = &StorageOptions{}
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	return &Storage{
		client: client,
		cfg:    client.cfg,
		logger: opts.Logger.WithGroup("storage").With("backend", "s3"),
	}
}

// NewDefault creates a Storage with a new client.
func NewDefault(cfg Config) (*Storage, error) {
	client, err := NewDefaultClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewStorage(client, nil), nil
}

// getClient returns the underlying minio client with validation.
func (s *Storage) getClient() (*minio.Client, error) {
	if s.client == nil || s.client.client == nil {
		return nil, &storage.StorageError{
			Code:    storage.CodeInternalError,
			Message: "minio client is not initialized",
		}
	}
	return s.client.client, nil
}

// Put stores an object in S3-compatible storage.
func (s *Storage) Put(ctx context.Context, bucket, key string, reader io.Reader, opts *storage.PutOptions) error {
	ctx, span := tracer.Start(ctx, "S3.Put", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if opts == nil {
		opts = &storage.PutOptions{}
	}

	// Set default bucket if not specified
	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
		attribute.String("content_type", opts.ContentType),
	)

	// Convert storage.PutOptions to minio.PutObjectOptions
	minioOpts := minio.PutObjectOptions{
		ContentType:  opts.ContentType,
		UserMetadata: opts.Metadata,
	}

	// Get the minio client
	client, err := s.getClient()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Upload the object
	info, err := client.PutObject(ctx, bucket, key, reader, -1, minioOpts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrapf(err, "failed to put object %s/%s", bucket, key)
	}

	span.SetAttributes(
		attribute.Int64("size", info.Size),
		attribute.String("etag", info.ETag),
	)
	span.SetStatus(codes.Ok, "")

	s.logger.Debug("Object stored", "bucket", bucket, "key", key, "size", info.Size)
	return nil
}

// Get retrieves an object from S3-compatible storage.
func (s *Storage) Get(ctx context.Context, bucket, key string) (io.ReadCloser, *storage.ObjectInfo, error) {
	ctx, span := tracer.Start(ctx, "S3.Get", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
	)

	// Get the minio client
	client, err := s.getClient()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, err
	}

	// Get the object
	obj, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, toStorageError(err, bucket, key)
	}

	// Get object info to return metadata
	stat, err := obj.Stat()
	if err != nil {
		closeErr := obj.Close()
		if closeErr != nil {
			s.logger.With("error", closeErr).Error("failed to close object after stat error")
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, toStorageError(err, bucket, key)
	}

	info := &storage.ObjectInfo{
		Key:          key,
		Size:         stat.Size,
		LastModified: stat.LastModified,
		ETag:         stat.ETag,
		ContentType:  stat.ContentType,
		Metadata:     stat.UserMetadata,
	}

	span.SetAttributes(
		attribute.Int64("size", stat.Size),
		attribute.String("etag", stat.ETag),
	)
	span.SetStatus(codes.Ok, "")

	return obj, info, nil
}

// Delete removes an object from S3-compatible storage.
func (s *Storage) Delete(ctx context.Context, bucket, key string) error {
	ctx, span := tracer.Start(ctx, "S3.Delete", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
	)

	// Get the minio client
	client, err := s.getClient()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	err = client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return toStorageError(err, bucket, key)
	}

	span.SetStatus(codes.Ok, "")
	s.logger.Debug("Object deleted", "bucket", bucket, "key", key)
	return nil
}

// Exists checks if an object exists in S3-compatible storage.
func (s *Storage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	ctx, span := tracer.Start(ctx, "S3.Exists", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
	)

	// Get the minio client
	client, err := s.getClient()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	_, err = client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNotFoundError(err) {
			span.SetStatus(codes.Ok, "")
			return false, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, toStorageError(err, bucket, key)
	}

	span.SetStatus(codes.Ok, "")
	return true, nil
}

// List lists objects in the specified bucket.
func (s *Storage) List(ctx context.Context, bucket string, opts *storage.ListOptions) (*storage.ListResult, error) {
	ctx, span := tracer.Start(ctx, "S3.List", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	if opts == nil {
		opts = &storage.ListOptions{}
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("prefix", opts.Prefix),
		attribute.Bool("recursive", opts.Recursive),
	)

	// Convert storage.ListOptions to minio.ListObjectsOptions
	minioOpts := minio.ListObjectsOptions{
		Prefix:       opts.Prefix,
		Recursive:    opts.Recursive,
		MaxKeys:      opts.MaxKeys,
		WithMetadata: true,
	}

	// Get the minio client
	client, err := s.getClient()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// List objects
	objectCh := client.ListObjects(ctx, bucket, minioOpts)

	var objects []storage.ObjectInfo

	for object := range objectCh {
		if object.Err != nil {
			span.RecordError(object.Err)
			span.SetStatus(codes.Error, object.Err.Error())
			return nil, errors.Wrap(object.Err, "failed to list objects")
		}

		// Skip directory markers (objects ending with "/" with size 0)
		if strings.HasSuffix(object.Key, "/") && object.Size == 0 {
			continue
		}

		objects = append(objects, storage.ObjectInfo{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			ETag:         object.ETag,
			ContentType:  object.ContentType,
			Metadata:     object.UserMetadata,
		})
	}

	result := &storage.ListResult{
		Objects:     objects,
		IsTruncated: false, // minio-go v7 doesn't provide this info directly
	}

	span.SetAttributes(
		attribute.Int("object_count", len(objects)),
	)
	span.SetStatus(codes.Ok, "")

	return result, nil
}

// Close closes the storage connection.
func (s *Storage) Close() error {
	return s.client.Close()
}
