# minio

Package minio provides a unified S3-compatible adapter for the storage interface.

Supports:
- **MinIO** - Local/development S3-compatible storage
- **Yandex Cloud Storage** - Production cloud storage
- **AWS S3** - Amazon Simple Storage Service
- **Any S3-compatible storage** - Other providers that implement the S3 API

## Usage

### MinIO (local development)

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

### Yandex Cloud Storage

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

### AWS S3

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

### Operations

```go
// Put an object
err = storage.Put(ctx, "my-bucket", "my-key", bytes.NewReader([]byte("hello")), nil)

// Get an object
reader, info, err := storage.Get(ctx, "my-bucket", "my-key")
defer reader.Close()

// List objects
result, err := storage.List(ctx, "my-bucket", &storage.ListOptions{
    Prefix: "prefix/",
})

// Check if object exists
exists, err := storage.Exists(ctx, "my-bucket", "my-key")

// Delete an object
err = storage.Delete(ctx, "my-bucket", "my-key")

// Generate presigned URL
url, err := storage.GetPresignedURL(ctx, "my-bucket", "my-key", &storage.PresignedURLOptions{
    Method: "GET",
    Expiry: 15 * time.Minute,
})
```

## Environment Variables

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

## Configuration

### Config

```go
type Config struct {
    Endpoint           string  // S3 endpoint (defaults to storage.yandexcloud.net)
    AccessKey          string  // Access key (required)
    SecretKey          string  // Secret key (required)
    Region             string  // Region (default: us-east-1)
    DefaultBucket      string  // Default bucket name
    Secure             bool    // Use HTTPS (default: true)
    Timeout            int     // Connection timeout in seconds (default: 30)
    InsecureSkipVerify bool    // Skip TLS verification (default: false)
}
```

### ClientOptions

```go
type ClientOptions struct {
    Logger         *slog.Logger         // Custom logger
    TracerProvider trace.TracerProvider // Custom tracer provider
}
```

### StorageOptions

```go
type StorageOptions struct {
    Logger *slog.Logger // Custom logger
}
```

## Constructors

- `NewClient(cfg Config, options *ClientOptions) (*Client, error)` - Create a new client with options
- `NewDefaultClient(cfg Config) (*Client, error)` - Create a client with default options
- `NewStorage(client *Client, opts *StorageOptions) *Storage` - Create storage from client
- `NewDefault(cfg Config) (*Storage, error)` - Create storage with new client and default options

## Features

- Full S3-compatible API support via minio-go
- Multipart upload for large files
- Presigned URL generation
- OpenTelemetry tracing
- Structured logging
- Context-aware operations
- Works with any S3-compatible storage provider
