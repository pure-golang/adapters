package minio

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/pure-golang/adapters/storage"
	"github.com/minio/minio-go/v7"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// core returns the minio.Core client for low-level operations.
func (s *Storage) core() *minio.Core {
	return &minio.Core{Client: s.client.client}
}

// CreateMultipartUpload initiates a multipart upload.
func (s *Storage) CreateMultipartUpload(ctx context.Context, bucket, key string, opts *storage.PutOptions) (*storage.MultipartUpload, error) {
	ctx, span := tracer.Start(ctx, "S3.CreateMultipartUpload", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	if opts == nil {
		opts = &storage.PutOptions{}
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
	)

	// Validate client is initialized
	if _, err := s.getClient(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Create multipart upload
	minioOpts := minio.PutObjectOptions{
		ContentType:  opts.ContentType,
		UserMetadata: opts.Metadata,
	}

	uploadID, err := s.core().NewMultipartUpload(ctx, bucket, key, minioOpts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrapf(err, "failed to create multipart upload %s/%s", bucket, key)
	}

	result := &storage.MultipartUpload{
		UploadID:  uploadID,
		Key:       key,
		Bucket:    bucket,
		Initiated: time.Now(),
	}

	span.SetAttributes(
		attribute.String("upload_id", uploadID),
	)
	span.SetStatus(codes.Ok, "")

	s.logger.Debug("Multipart upload created", "bucket", bucket, "key", key, "upload_id", uploadID)
	return result, nil
}

// UploadPart uploads a part in a multipart upload.
func (s *Storage) UploadPart(ctx context.Context, bucket, key, uploadID string, partNumber int32, reader io.Reader) (*storage.UploadedPart, error) {
	ctx, span := tracer.Start(ctx, "S3.UploadPart", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
		attribute.String("upload_id", uploadID),
		attribute.Int("part_number", int(partNumber)),
	)

	// Determine size of the reader
	var size int64
	type sizer interface {
		Size() int64
	}
	if sz, ok := reader.(sizer); ok {
		size = sz.Size()
	} else {
		// Read into buffer to get size
		buf, err := io.ReadAll(reader)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, errors.Wrap(err, "failed to read part data")
		}
		size = int64(len(buf))
		reader = bytes.NewReader(buf)
	}

	// Validate client is initialized
	if _, err := s.getClient(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Upload the part using Core.PutObjectPart
	putOpts := minio.PutObjectPartOptions{}
	info, err := s.core().PutObjectPart(ctx, bucket, key, uploadID, int(partNumber), reader, size, putOpts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrapf(err, "failed to upload part %d of %s/%s", partNumber, bucket, key)
	}

	result := &storage.UploadedPart{
		PartNumber: partNumber,
		ETag:       info.ETag,
		Size:       info.Size,
	}

	span.SetAttributes(
		attribute.String("etag", info.ETag),
		attribute.Int64("size", info.Size),
	)
	span.SetStatus(codes.Ok, "")

	s.logger.Debug("Part uploaded", "bucket", bucket, "key", key, "part_number", partNumber, "size", info.Size)
	return result, nil
}

// CompleteMultipartUpload completes a multipart upload.
func (s *Storage) CompleteMultipartUpload(ctx context.Context, bucket, key, uploadID string, opts *storage.CompleteMultipartUploadOptions) (*storage.ObjectInfo, error) {
	ctx, span := tracer.Start(ctx, "S3.CompleteMultipartUpload", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
		attribute.String("upload_id", uploadID),
		attribute.Int("part_count", len(opts.Parts)),
	)

	// Convert parts to minio format
	minioParts := make([]minio.CompletePart, len(opts.Parts))
	for i, p := range opts.Parts {
		minioParts[i] = minio.CompletePart{
			PartNumber: int(p.PartNumber),
			ETag:       p.ETag,
		}
	}

	// Validate client is initialized
	if _, err := s.getClient(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Complete the upload using Core.CompleteMultipartUpload
	minioOpts := minio.PutObjectOptions{}
	info, err := s.core().CompleteMultipartUpload(ctx, bucket, key, uploadID, minioParts, minioOpts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrapf(err, "failed to complete multipart upload %s/%s", bucket, key)
	}

	// Calculate total size from uploaded parts since info.Size might be 0
	var totalSize int64
	for _, p := range opts.Parts {
		totalSize += p.Size
	}

	// If calculated size is 0 or info.Size is non-zero, use info.Size
	if totalSize == 0 && info.Size > 0 {
		totalSize = info.Size
	}

	// Get object stat to fetch accurate metadata
	client, err := s.getClient()
	if err != nil {
		// If client validation fails, return the info we have
		result := &storage.ObjectInfo{
			Key:          key,
			Size:         totalSize,
			ETag:         info.ETag,
			ContentType:  "",
			LastModified: info.LastModified,
		}
		span.SetAttributes(
			attribute.Int64("size", totalSize),
			attribute.String("etag", info.ETag),
		)
		span.SetStatus(codes.Ok, "")
		s.logger.Info("Multipart upload completed", "bucket", bucket, "key", key, "size", totalSize)
		return result, nil
	}

	stat, err := client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		// If stat fails, use the info we have
		result := &storage.ObjectInfo{
			Key:          key,
			Size:         totalSize,
			ETag:         info.ETag,
			ContentType:  "",
			LastModified: info.LastModified,
		}
		span.SetAttributes(
			attribute.Int64("size", totalSize),
			attribute.String("etag", info.ETag),
		)
		span.SetStatus(codes.Ok, "")
		s.logger.Info("Multipart upload completed", "bucket", bucket, "key", key, "size", totalSize)
		return result, nil
	}

	result := &storage.ObjectInfo{
		Key:          key,
		Size:         stat.Size,
		ETag:         stat.ETag,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified,
		Metadata:     stat.UserMetadata,
	}

	span.SetAttributes(
		attribute.Int64("size", stat.Size),
		attribute.String("etag", stat.ETag),
	)
	span.SetStatus(codes.Ok, "")

	s.logger.Info("Multipart upload completed", "bucket", bucket, "key", key, "size", stat.Size)
	return result, nil
}

// AbortMultipartUpload aborts a multipart upload.
func (s *Storage) AbortMultipartUpload(ctx context.Context, bucket, key, uploadID string) error {
	ctx, span := tracer.Start(ctx, "S3.AbortMultipartUpload", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
		attribute.String("upload_id", uploadID),
	)

	// Validate client is initialized
	if _, err := s.getClient(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	err := s.core().AbortMultipartUpload(ctx, bucket, key, uploadID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrapf(err, "failed to abort multipart upload %s/%s", bucket, key)
	}

	span.SetStatus(codes.Ok, "")
	s.logger.Debug("Multipart upload aborted", "bucket", bucket, "key", key, "upload_id", uploadID)
	return nil
}

// ListMultipartUploads lists active multipart uploads.
func (s *Storage) ListMultipartUploads(ctx context.Context, bucket string) ([]storage.MultipartUpload, error) {
	ctx, span := tracer.Start(ctx, "S3.ListMultipartUploads", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
	)

	// Use the minio client's ListIncompleteUploads with recursive=true
	// This uses the high-level API which should be more reliable
	ch := s.client.client.ListIncompleteUploads(ctx, bucket, "", true)

	var uploads []storage.MultipartUpload
	for upload := range ch {
		if upload.Err != nil {
			span.RecordError(upload.Err)
			span.SetStatus(codes.Error, upload.Err.Error())
			return nil, errors.Wrap(upload.Err, "failed to list multipart uploads")
		}

		uploads = append(uploads, storage.MultipartUpload{
			UploadID:  upload.UploadID,
			Key:       upload.Key,
			Bucket:    bucket,
			Initiated: upload.Initiated,
		})
	}

	s.logger.Debug("ListMultipartUploads result", "bucket", bucket, "upload_count", len(uploads))

	span.SetAttributes(
		attribute.Int("upload_count", len(uploads)),
	)
	span.SetStatus(codes.Ok, "")

	return uploads, nil
}
