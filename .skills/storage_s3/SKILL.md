---
name: "storage_s3"
description: "Паттерны S3-совместимого хранилища (MinIO, Yandex Cloud, AWS S3): CRUD, range, presigned URLs, multipart"
modes: [Code, Ask]
---
# Skill: S3/MinIO Storage Patterns

## Tactical Instructions

### Basic Operations
```go
cfg := minio.Config{
    Endpoint:  "localhost:9000",
    AccessKey: "minioadmin",
    SecretKey: "minioadmin",
    Secure:    false,
}
storage, err := minio.NewDefault(cfg)
defer storage.Close()

// Put object
err = storage.Put(ctx, "my-bucket", "my-key", bytes.NewReader(data), &storage.PutOptions{
    ContentType: "application/json",
    Metadata:    map[string]string{"author": "user1"},
})

// Get object
reader, info, err := storage.Get(ctx, "my-bucket", "my-key")
defer reader.Close()

// Check existence
exists, err := storage.Exists(ctx, "my-bucket", "my-key")

// Delete object
err = storage.Delete(ctx, "my-bucket", "my-key")
```

### Range Requests (partial reads)
```go
// GetFileHeader retrieves first 4096 bytes using range request (opts.SetRange(0, 4095))
// Useful for file type detection without downloading entire file
header, err := storage.GetFileHeader(ctx, "my-bucket", "large-file.bin")
if err != nil {
    return err
}

fileType := http.DetectContentType(header)
```

### Presigned URLs
```go
// Generate temporary URL for direct client access (bypasses your server)
url, err := storage.GetPresignedURL(ctx, "my-bucket", "my-key", &storage.PresignedURLOptions{
    Method: "GET",
    Expiry: 15 * time.Minute,
})
```

### Multipart Upload (files > 5MB)
```go
upload, err := storage.CreateMultipartUpload(ctx, "bucket", "large-file.bin", nil)

parts := make([]storage.UploadedPart, 0)
for i := 0; i < numParts; i++ {
    part, err := storage.UploadPart(ctx, "bucket", "large-file.bin", upload.UploadID,
        int32(i+1), partReader)
    if err != nil {
        storage.AbortMultipartUpload(ctx, "bucket", "large-file.bin", upload.UploadID)
        return err
    }
    parts = append(parts, *part)
}

info, err := storage.CompleteMultipartUpload(ctx, "bucket", "large-file.bin",
    upload.UploadID, &storage.CompleteMultipartUploadOptions{Parts: parts})
```

### List Objects
```go
result, err := storage.List(ctx, "my-bucket", &storage.ListOptions{
    Prefix:    "photos/2024/",
    Recursive: true,
    MaxKeys:   1000,
})

for _, obj := range result.Objects {
    fmt.Printf("%s (%d bytes)\n", obj.Key, obj.Size)
}
```

### Notes
- Span naming pattern: `S3.operation` (e.g., `S3.GetFileHeader`, `S3.Put`, `S3.Get`)
- All operations support context cancellation and OpenTelemetry tracing
- Multipart upload recommended for files > 5MB
- Use presigned URLs for direct client access to avoid proxying through your server
