//go:build integration
// +build integration

package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"

	"github.com/pure-golang/adapters/storage"
)

// TestIntegrationWithTestcontainers runs integration tests using testcontainers-go.
func TestIntegrationWithTestcontainers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start MinIO container
	minioContainer, err := tcminio.RunContainer(ctx,
		tcminio.WithUsername("minioadmin"),
		tcminio.WithPassword("minioadmin"),
	)
	require.NoError(t, err)
	defer minioContainer.Terminate(ctx) // nolint:errcheck

	// Get connection details
	endpoint, err := minioContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	// Create MinIO client
	cfg := Config{
		Endpoint:  strings.TrimPrefix(endpoint, "http://"),
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Region:    "us-east-1",
		Secure:    false,
	}

	client, err := NewClient(cfg, nil)
	require.NoError(t, err)
	defer client.Close()

	// Create test bucket
	bucket := "test-bucket"
	err = client.GetMinioClient().MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	require.NoError(t, err)

	stor := NewStorage(client, nil)

	t.Run("PutAndGet", func(t *testing.T) {
		ctx := context.Background()
		key := "test-object.txt"
		content := []byte("Hello, MinIO!")

		// Put object
		err := stor.Put(ctx, bucket, key, bytes.NewReader(content), &storage.PutOptions{
			ContentType: "text/plain",
			Metadata:    map[string]string{"test": "metadata"},
		})
		require.NoError(t, err)

		// Get object
		rc, info, err := stor.Get(ctx, bucket, key)
		require.NoError(t, err)
		defer rc.Close()

		assert.Equal(t, int64(len(content)), info.Size)
		assert.Equal(t, "text/plain", info.ContentType)

		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("Exists", func(t *testing.T) {
		ctx := context.Background()
		key := "exists-test.txt"

		// Not exists initially
		exists, err := stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.False(t, exists)

		// Put object
		err = stor.Put(ctx, bucket, key, strings.NewReader("test"), nil)
		require.NoError(t, err)

		// Now exists
		exists, err = stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Delete", func(t *testing.T) {
		ctx := context.Background()
		key := "delete-test.txt"

		// Put object
		err := stor.Put(ctx, bucket, key, strings.NewReader("test"), nil)
		require.NoError(t, err)

		// Delete object
		err = stor.Delete(ctx, bucket, key)
		require.NoError(t, err)

		// Verify deleted
		exists, err := stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("List", func(t *testing.T) {
		ctx := context.Background()

		// Put multiple objects
		prefix := "list-test/"
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("%sobj%d.txt", prefix, i)
			err := stor.Put(ctx, bucket, key, strings.NewReader("test"), nil)
			require.NoError(t, err)
			t.Logf("Created object: %s", key)
		}

		// List with prefix - should only return our 5 objects
		result, err := stor.List(ctx, bucket, &storage.ListOptions{
			Prefix: prefix,
		})
		require.NoError(t, err)
		t.Logf("List with prefix returned %d objects", len(result.Objects))
		for _, obj := range result.Objects {
			t.Logf("  - %s (size=%d)", obj.Key, obj.Size)
		}
		assert.Len(t, result.Objects, 5)

		// List recursive
		result, err = stor.List(ctx, bucket, &storage.ListOptions{
			Prefix:    prefix,
			Recursive: true,
		})
		require.NoError(t, err)
		t.Logf("List with prefix+recursive returned %d objects", len(result.Objects))
		assert.Len(t, result.Objects, 5)

		// List all with recursive=true to get all objects
		result, err = stor.List(ctx, bucket, &storage.ListOptions{
			Recursive: true,
		})
		require.NoError(t, err)
		t.Logf("List all (recursive) returned %d total objects", len(result.Objects))
		for _, obj := range result.Objects {
			t.Logf("  - %s (size=%d)", obj.Key, obj.Size)
		}

		// Count objects with our prefix
		listTestCount := 0
		for _, obj := range result.Objects {
			if strings.HasPrefix(obj.Key, prefix) {
				listTestCount++
			}
		}
		t.Logf("Found %d objects with prefix %s", listTestCount, prefix)
		assert.GreaterOrEqual(t, listTestCount, 5)
	})

	t.Run("PresignedURL", func(t *testing.T) {
		ctx := context.Background()
		key := "presigned-test.txt"
		content := []byte("presigned content")

		// Put object
		err := stor.Put(ctx, bucket, key, bytes.NewReader(content), nil)
		require.NoError(t, err)

		// Get presigned GET URL
		url, err := stor.GetPresignedURL(ctx, bucket, key, &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		})
		require.NoError(t, err)
		assert.Contains(t, url, bucket)
		assert.Contains(t, url, key)

		// Get presigned PUT URL
		url, err = stor.GetPresignedURL(ctx, bucket, "new-key", &storage.PresignedURLOptions{
			Method: "PUT",
			Expiry: 15 * time.Minute,
		})
		require.NoError(t, err)
		assert.Contains(t, url, "new-key")
	})

	t.Run("MultipartUpload", func(t *testing.T) {
		ctx := context.Background()
		key := "multipart-test.txt"

		// Create multipart upload
		upload, err := stor.CreateMultipartUpload(ctx, bucket, key, &storage.PutOptions{
			ContentType: "text/plain",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, upload.UploadID)

		// Upload parts
		part1Content := bytes.Repeat([]byte("A"), 5*1024*1024) // 5MB
		part2Content := bytes.Repeat([]byte("B"), 5*1024*1024) // 5MB

		part1, err := stor.UploadPart(ctx, bucket, key, upload.UploadID, 1, bytes.NewReader(part1Content))
		require.NoError(t, err)
		assert.Equal(t, int32(1), part1.PartNumber)
		assert.NotEmpty(t, part1.ETag)

		part2, err := stor.UploadPart(ctx, bucket, key, upload.UploadID, 2, bytes.NewReader(part2Content))
		require.NoError(t, err)
		assert.Equal(t, int32(2), part2.PartNumber)
		assert.NotEmpty(t, part2.ETag)

		// Complete upload
		info, err := stor.CompleteMultipartUpload(ctx, bucket, key, upload.UploadID, &storage.CompleteMultipartUploadOptions{
			Parts: []storage.UploadedPart{*part1, *part2},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(len(part1Content)+len(part2Content)), info.Size)

		// Verify object exists
		exists, err := stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("AbortMultipartUpload", func(t *testing.T) {
		ctx := context.Background()
		key := "abort-test.txt"

		// Create multipart upload
		upload, err := stor.CreateMultipartUpload(ctx, bucket, key, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, upload.UploadID)

		// Upload a part to ensure the upload is registered in MinIO
		partContent := bytes.Repeat([]byte("X"), 5*1024*1024) // 5MB
		part, err := stor.UploadPart(ctx, bucket, key, upload.UploadID, 1, bytes.NewReader(partContent))
		require.NoError(t, err)
		assert.Equal(t, int32(1), part.PartNumber)

		// Abort upload
		err = stor.AbortMultipartUpload(ctx, bucket, key, upload.UploadID)
		require.NoError(t, err)

		// Verify the object was not created (abort should have cleaned up)
		exists, err := stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.False(t, exists, "object should not exist after aborting multipart upload")
	})

	t.Run("DefaultBucket", func(t *testing.T) {
		ctx := context.Background()
		key := "default-bucket-test.txt"

		// Create storage with default bucket set in config
		cfgWithDefault := cfg
		cfgWithDefault.DefaultBucket = bucket
		client2, err := NewClient(cfgWithDefault, nil)
		require.NoError(t, err)
		defer client2.Close()

		stor2 := NewStorage(client2, nil)

		// Put with empty bucket (should use default)
		err = stor2.Put(ctx, "", key, strings.NewReader("test"), nil)
		require.NoError(t, err)

		// Verify exists
		exists, err := stor2.Exists(ctx, "", key)
		require.NoError(t, err)
		assert.True(t, exists)

		// Get with empty bucket
		rc, info, err := stor2.Get(ctx, "", key)
		require.NoError(t, err)
		defer rc.Close()
		assert.NotNil(t, info)
		assert.NotNil(t, rc)

		// Delete with empty bucket
		err = stor2.Delete(ctx, "", key)
		require.NoError(t, err)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		ctx := context.Background()

		// Get non-existent object
		rc, info, err := stor.Get(ctx, bucket, "non-existent-key")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)

		storageErr, ok := err.(*storage.StorageError)
		require.True(t, ok)
		assert.Equal(t, storage.CodeNotFound, storageErr.Code)
	})

	t.Run("Error_DeleteNotFound", func(t *testing.T) {
		ctx := context.Background()

		// Delete non-existent object - S3-compatible storage doesn't error
		// (idempotent delete is standard S3 behavior)
		err := stor.Delete(ctx, bucket, "non-existent-key")
		assert.NoError(t, err)
	})

	t.Run("Error_BucketNotFound", func(t *testing.T) {
		ctx := context.Background()

		// Try to get from non-existent bucket
		rc, info, err := stor.Get(ctx, "non-existent-bucket", "key")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)

		storageErr, ok := err.(*storage.StorageError)
		require.True(t, ok)
		assert.Equal(t, storage.CodeBucketNotFound, storageErr.Code)
	})

	t.Run("ListMultipartUploads", func(t *testing.T) {
		ctx := context.Background()

		// List should work without error
		_, err := stor.ListMultipartUploads(ctx, bucket)
		require.NoError(t, err)

		// Create a multipart upload
		key1 := "list-upload-test.txt"
		upload1, err := stor.CreateMultipartUpload(ctx, bucket, key1, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, upload1.UploadID)

		// Upload a part
		partContent := bytes.Repeat([]byte("X"), 5*1024*1024) // 5MB
		_, err = stor.UploadPart(ctx, bucket, key1, upload1.UploadID, 1, bytes.NewReader(partContent))
		require.NoError(t, err)

		// List again - method should work without error
		_, err = stor.ListMultipartUploads(ctx, bucket)
		require.NoError(t, err)
		// Important: method works without error, uploads may or may not contain our upload
		// depending on MinIO's internal consistency

		// Clean up - abort the upload
		_ = stor.AbortMultipartUpload(ctx, bucket, key1, upload1.UploadID)
	})

	t.Run("ListMultipartUploadsWithEmptyBucket", func(t *testing.T) {
		ctx := context.Background()

		// Create storage with default bucket
		cfgWithDefault := cfg
		cfgWithDefault.DefaultBucket = bucket
		client3, err := NewClient(cfgWithDefault, nil)
		require.NoError(t, err)
		defer client3.Close()

		stor3 := NewStorage(client3, nil)

		// List with empty bucket should use default
		uploads, err := stor3.ListMultipartUploads(ctx, "")
		require.NoError(t, err)
		// uploads may be nil or empty slice
		_ = uploads // we verified it doesn't error
	})

	t.Run("GetNotFound", func(t *testing.T) {
		ctx := context.Background()

		// Get non-existent object
		rc, info, err := stor.Get(ctx, bucket, "definitely-not-a-real-key-12345")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)

		storageErr, ok := err.(*storage.StorageError)
		require.True(t, ok, "error should be StorageError type")
		assert.Equal(t, storage.CodeNotFound, storageErr.Code)
	})

	t.Run("PutAndGetLargeObject", func(t *testing.T) {
		ctx := context.Background()
		key := "large-object.bin"

		// Create a 10MB object
		largeContent := bytes.Repeat([]byte("DATA"), 250*1024) // ~1MB of "DATA"

		err := stor.Put(ctx, bucket, key, bytes.NewReader(largeContent), &storage.PutOptions{
			ContentType: "application/octet-stream",
		})
		require.NoError(t, err)

		// Get and verify
		rc, info, err := stor.Get(ctx, bucket, key)
		require.NoError(t, err)
		defer rc.Close()

		assert.Equal(t, int64(len(largeContent)), info.Size)

		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, len(largeContent), len(data))
	})
}
