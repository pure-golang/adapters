# storage

Package storage provides a generic interface for S3-compatible object storage operations.

## Interface

The `Storage` interface defines operations for object storage:

- `Put` - Store an object
- `Get` - Retrieve an object
- `Delete` - Remove an object
- `Exists` - Check if an object exists
- `List` - List objects in a bucket
- `GetPresignedURL` - Generate a presigned URL for direct access
- `CreateMultipartUpload` - Initiate a multipart upload
- `UploadPart` - Upload a part in a multipart upload
- `CompleteMultipartUpload` - Complete a multipart upload
- `AbortMultipartUpload` - Abort a multipart upload
- `ListMultipartUploads` - List active multipart uploads

## S3 Adapter

The `minio` package provides a unified S3-compatible adapter that works with:

- **MinIO** - Local/development S3-compatible storage
- **Yandex Cloud Storage** - Production cloud storage
- **AWS S3** - Amazon Simple Storage Service
- **Any S3-compatible storage** - Other providers that implement the S3 API

### Usage

#### MinIO (local development)

```go
import "github.com/pure-golang/adapters/storage/minio"

cfg := minio.Config{
    Endpoint:  "localhost:9000",
    AccessKey: "minioadmin",
    SecretKey: "minioadmin",
    Secure:    false,
}

storage, err := minio.NewDefault(cfg)
if err != nil {
    log.Fatal(err)
}
defer storage.Close()
```

#### Yandex Cloud Storage

```go
import "github.com/pure-golang/adapters/storage/minio"

cfg := minio.Config{
    Endpoint:  "storage.yandexcloud.net",  // or leave empty for default
    AccessKey: "your-access-key-id",
    SecretKey: "your-secret-access-key",
    Region:    "ru-central1",
}

storage, err := minio.NewDefault(cfg)
if err != nil {
    log.Fatal(err)
}
defer storage.Close()
```

#### AWS S3

```go
import "github.com/pure-golang/adapters/storage/minio"

cfg := minio.Config{
    Endpoint:  "s3.amazonaws.com",
    AccessKey: "your-access-key-id",
    SecretKey: "your-secret-access-key",
    Region:    "us-east-1",
    Secure:    true,
}

storage, err := minio.NewDefault(cfg)
if err != nil {
    log.Fatal(err)
}
defer storage.Close()
```

## Environment Variables

The adapter supports configuration via environment variables using `envconfig` tags.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| S3_ENDPOINT | No* | storage.yandexcloud.net | S3 endpoint URL |
| S3_ACCESS_KEY | Yes | - | Access key ID |
| S3_SECRET_KEY | Yes | - | Secret access key |
| S3_REGION | No | us-east-1 | Region name |
| S3_BUCKET | No | - | Default bucket name |
| S3_SECURE | No | true | Use HTTPS |
| S3_TIMEOUT | No | 30 | Connection timeout (seconds) |
| S3_INSECURE_SKIP_VERIFY | No | false | Skip TLS verification |

*For local MinIO, you typically need to specify an endpoint like `localhost:9000`

### Example with environment variables

```bash
# MinIO (local)
export S3_ENDPOINT=localhost:9000
export S3_ACCESS_KEY=minioadmin
export S3_SECRET_KEY=minioadmin
export S3_SECURE=false

# Yandex Cloud
export S3_ENDPOINT=storage.yandexcloud.net
export S3_ACCESS_KEY=your-access-key-id
export S3_SECRET_KEY=your-secret-access-key
export S3_REGION=ru-central1

# AWS S3
export S3_ENDPOINT=s3.amazonaws.com
export S3_ACCESS_KEY=your-aws-access-key
export S3_SECRET_KEY=your-aws-secret-key
export S3_REGION=us-east-1
```

## Features

- **S3-Compatible API** - Works with any S3-compatible storage provider
- **Context Support** - All operations accept `context.Context` for cancellation and timeout
- **OpenTelemetry Tracing** - Built-in distributed tracing
- **Structured Logging** - Uses `log/slog` for structured logging
- **Multipart Uploads** - Support for large file uploads
- **Presigned URLs** - Generate direct access URLs for GET/PUT operations
- **Error Handling** - Typed errors with proper error codes
