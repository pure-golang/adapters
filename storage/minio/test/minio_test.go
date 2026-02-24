package minio_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"

	"github.com/pure-golang/adapters/storage"
	"github.com/pure-golang/adapters/storage/minio"
)

func TestIntegrationWithTestcontainers(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()

	minioContainer, err := tcminio.RunContainer(ctx,
		tcminio.WithUsername("minioadmin"),
		tcminio.WithPassword("minioadmin"),
	)
	require.NoError(t, err)
	defer minioContainer.Terminate(ctx) // nolint:errcheck

	endpoint, err := minioContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	cfg := minio.Config{
		Endpoint:  strings.TrimPrefix(endpoint, "http://"),
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Region:    "us-east-1",
		Secure:    false,
	}

	client, err := minio.NewClient(cfg)
	require.NoError(t, err)
	defer client.Close()

	bucket := "test-bucket"
	err = client.GetMinioClient().MakeBucket(ctx, bucket, miniogo.MakeBucketOptions{})
	require.NoError(t, err)

	stor := minio.NewStorage(client, nil)

	t.Run("PutAndGet", func(t *testing.T) {
		ctx := context.Background()
		key := "test-object.txt"
		content := []byte("Hello, MinIO!")

		err := stor.Put(ctx, bucket, key, bytes.NewReader(content), &storage.PutOptions{
			ContentType: "text/plain",
			Metadata:    map[string]string{"test": "metadata"},
		})
		require.NoError(t, err)

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

		exists, err := stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.False(t, exists)

		err = stor.Put(ctx, bucket, key, strings.NewReader("test"), nil)
		require.NoError(t, err)

		exists, err = stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Delete", func(t *testing.T) {
		ctx := context.Background()
		key := "delete-test.txt"

		err := stor.Put(ctx, bucket, key, strings.NewReader("test"), nil)
		require.NoError(t, err)

		err = stor.Delete(ctx, bucket, key)
		require.NoError(t, err)

		exists, err := stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("List", func(t *testing.T) {
		ctx := context.Background()

		prefix := "list-test/"
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("%sobj%d.txt", prefix, i)
			err := stor.Put(ctx, bucket, key, strings.NewReader("test"), nil)
			require.NoError(t, err)
			t.Logf("Created object: %s", key)
		}

		result, err := stor.List(ctx, bucket, &storage.ListOptions{
			Prefix: prefix,
		})
		require.NoError(t, err)
		t.Logf("List with prefix returned %d objects", len(result.Objects))
		assert.Len(t, result.Objects, 5)

		result, err = stor.List(ctx, bucket, &storage.ListOptions{
			Prefix:    prefix,
			Recursive: true,
		})
		require.NoError(t, err)
		assert.Len(t, result.Objects, 5)

		result, err = stor.List(ctx, bucket, &storage.ListOptions{
			Recursive: true,
		})
		require.NoError(t, err)
		listTestCount := 0
		for _, obj := range result.Objects {
			if strings.HasPrefix(obj.Key, prefix) {
				listTestCount++
			}
		}
		assert.GreaterOrEqual(t, listTestCount, 5)
	})

	t.Run("PresignedURL", func(t *testing.T) {
		ctx := context.Background()
		key := "presigned-test.txt"
		content := []byte("presigned content")

		err := stor.Put(ctx, bucket, key, bytes.NewReader(content), nil)
		require.NoError(t, err)

		url, err := stor.GetPresignedURL(ctx, bucket, key, &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		})
		require.NoError(t, err)
		assert.Contains(t, url, bucket)
		assert.Contains(t, url, key)

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

		upload, err := stor.CreateMultipartUpload(ctx, bucket, key, &storage.PutOptions{
			ContentType: "text/plain",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, upload.UploadID)

		part1Content := bytes.Repeat([]byte("A"), 5*1024*1024)
		part2Content := bytes.Repeat([]byte("B"), 5*1024*1024)

		part1, err := stor.UploadPart(ctx, bucket, key, upload.UploadID, 1, bytes.NewReader(part1Content))
		require.NoError(t, err)
		assert.Equal(t, int32(1), part1.PartNumber)
		assert.NotEmpty(t, part1.ETag)

		part2, err := stor.UploadPart(ctx, bucket, key, upload.UploadID, 2, bytes.NewReader(part2Content))
		require.NoError(t, err)
		assert.Equal(t, int32(2), part2.PartNumber)
		assert.NotEmpty(t, part2.ETag)

		info, err := stor.CompleteMultipartUpload(ctx, bucket, key, upload.UploadID, &storage.CompleteMultipartUploadOptions{
			Parts: []storage.UploadedPart{*part1, *part2},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(len(part1Content)+len(part2Content)), info.Size)

		exists, err := stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("AbortMultipartUpload", func(t *testing.T) {
		ctx := context.Background()
		key := "abort-test.txt"

		upload, err := stor.CreateMultipartUpload(ctx, bucket, key, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, upload.UploadID)

		partContent := bytes.Repeat([]byte("X"), 5*1024*1024)
		part, err := stor.UploadPart(ctx, bucket, key, upload.UploadID, 1, bytes.NewReader(partContent))
		require.NoError(t, err)
		assert.Equal(t, int32(1), part.PartNumber)

		err = stor.AbortMultipartUpload(ctx, bucket, key, upload.UploadID)
		require.NoError(t, err)

		exists, err := stor.Exists(ctx, bucket, key)
		require.NoError(t, err)
		assert.False(t, exists, "object should not exist after aborting multipart upload")
	})

	t.Run("DefaultBucket", func(t *testing.T) {
		ctx := context.Background()
		key := "default-bucket-test.txt"

		cfgWithDefault := cfg
		cfgWithDefault.DefaultBucket = bucket
		client2, err := minio.NewClient(cfgWithDefault)
		require.NoError(t, err)
		defer client2.Close()

		stor2 := minio.NewStorage(client2, nil)

		err = stor2.Put(ctx, "", key, strings.NewReader("test"), nil)
		require.NoError(t, err)

		exists, err := stor2.Exists(ctx, "", key)
		require.NoError(t, err)
		assert.True(t, exists)

		rc, info, err := stor2.Get(ctx, "", key)
		require.NoError(t, err)
		defer rc.Close()
		assert.NotNil(t, info)
		assert.NotNil(t, rc)

		err = stor2.Delete(ctx, "", key)
		require.NoError(t, err)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		ctx := context.Background()

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
		err := stor.Delete(ctx, bucket, "non-existent-key")
		assert.NoError(t, err)
	})

	t.Run("Error_BucketNotFound", func(t *testing.T) {
		ctx := context.Background()

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

		_, err := stor.ListMultipartUploads(ctx, bucket)
		require.NoError(t, err)

		key1 := "list-upload-test.txt"
		upload1, err := stor.CreateMultipartUpload(ctx, bucket, key1, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, upload1.UploadID)

		partContent := bytes.Repeat([]byte("X"), 5*1024*1024)
		_, err = stor.UploadPart(ctx, bucket, key1, upload1.UploadID, 1, bytes.NewReader(partContent))
		require.NoError(t, err)

		_, err = stor.ListMultipartUploads(ctx, bucket)
		require.NoError(t, err)

		_ = stor.AbortMultipartUpload(ctx, bucket, key1, upload1.UploadID)
	})

	t.Run("ListMultipartUploadsWithEmptyBucket", func(t *testing.T) {
		ctx := context.Background()

		cfgWithDefault := cfg
		cfgWithDefault.DefaultBucket = bucket
		client3, err := minio.NewClient(cfgWithDefault)
		require.NoError(t, err)
		defer client3.Close()

		stor3 := minio.NewStorage(client3, nil)

		uploads, err := stor3.ListMultipartUploads(ctx, "")
		require.NoError(t, err)
		_ = uploads
	})

	t.Run("GetNotFound", func(t *testing.T) {
		ctx := context.Background()

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

		largeContent := bytes.Repeat([]byte("DATA"), 250*1024)

		err := stor.Put(ctx, bucket, key, bytes.NewReader(largeContent), &storage.PutOptions{
			ContentType: "application/octet-stream",
		})
		require.NoError(t, err)

		rc, info, err := stor.Get(ctx, bucket, key)
		require.NoError(t, err)
		defer rc.Close()

		assert.Equal(t, int64(len(largeContent)), info.Size)

		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, len(largeContent), len(data))
	})

	t.Run("GetFileHeader", func(t *testing.T) {
		ctx := context.Background()
		key := "header-test.txt"
		content := []byte("Hello, MinIO! This is a test file for GetFileHeader.")

		err := stor.Put(ctx, bucket, key, bytes.NewReader(content), &storage.PutOptions{
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		header, err := stor.GetFileHeader(ctx, bucket, key)
		require.NoError(t, err)
		assert.Len(t, header, len(content))
		assert.Equal(t, content, header)
	})

	t.Run("GetFileHeaderLargeFile", func(t *testing.T) {
		ctx := context.Background()
		key := "header-large-test.bin"

		largeContent := bytes.Repeat([]byte("X"), 10*1024)

		err := stor.Put(ctx, bucket, key, bytes.NewReader(largeContent), &storage.PutOptions{
			ContentType: "application/octet-stream",
		})
		require.NoError(t, err)

		header, err := stor.GetFileHeader(ctx, bucket, key)
		require.NoError(t, err)
		assert.Len(t, header, 4096)
		assert.Equal(t, largeContent[:4096], header)
	})

	t.Run("GetFileHeaderNotFound", func(t *testing.T) {
		ctx := context.Background()

		header, err := stor.GetFileHeader(ctx, bucket, "non-existent-file.txt")
		assert.Error(t, err)
		assert.Nil(t, header)

		storageErr, ok := err.(*storage.StorageError)
		require.True(t, ok, "error should be StorageError type")
		assert.Equal(t, storage.CodeNotFound, storageErr.Code)
	})
}
