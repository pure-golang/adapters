package minio

import (
	"context"
	"net/url"
	"time"

	"github.com/pure-golang/adapters/storage"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// GetPresignedURL generates a presigned URL for S3 object access.
func (s *Storage) GetPresignedURL(ctx context.Context, bucket, key string, opts *storage.PresignedURLOptions) (string, error) {
	ctx, span := tracer.Start(ctx, "S3.GetPresignedURL", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	if bucket == "" {
		bucket = s.cfg.DefaultBucket
	}

	if opts == nil {
		opts = &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
	}

	span.SetAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
		attribute.String("method", opts.Method),
		attribute.Int("expiry_seconds", int(opts.Expiry.Seconds())),
	)

	// Set default expiry
	if opts.Expiry == 0 {
		opts.Expiry = 15 * time.Minute
	}

	// Set default method
	if opts.Method == "" {
		opts.Method = "GET"
	}

	var presignedURL *url.URL
	var err error

	// Validate method first (before client validation)
	if opts.Method != "GET" && opts.Method != "PUT" {
		err = errors.Errorf("unsupported HTTP method: %s", opts.Method)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	// Validate client is initialized before generating presigned URL
	client, clientErr := s.getClient()
	if clientErr != nil {
		span.RecordError(clientErr)
		span.SetStatus(codes.Error, clientErr.Error())
		return "", clientErr
	}

	switch opts.Method {
	case "GET":
		presignedURL, err = client.PresignedGetObject(ctx, bucket, key, opts.Expiry, nil)
	case "PUT":
		presignedURL, err = client.PresignedPutObject(ctx, bucket, key, opts.Expiry)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", errors.Wrapf(err, "failed to generate presigned URL for %s/%s", bucket, key)
	}

	span.SetStatus(codes.Ok, "")
	s.logger.Debug("Presigned URL generated", "bucket", bucket, "key", key, "method", opts.Method, "expiry", opts.Expiry)

	return presignedURL.String(), nil
}
