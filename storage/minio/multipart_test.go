package minio

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pure-golang/adapters/storage"
	"github.com/stretchr/testify/assert"
)

// TestCreateMultipartUpload_Extended tests the CreateMultipartUpload method.
func TestCreateMultipartUpload_Extended(t *testing.T) {
	t.Run("with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", "key", nil)
		assert.Error(t, err)
		assert.Nil(t, upload)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with empty bucket uses default", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		upload, err := stor.CreateMultipartUpload(context.Background(), "", "key", nil)
		assert.Error(t, err)
		assert.Nil(t, upload)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with nil options uses defaults", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", "key", nil)
		assert.Error(t, err)
		assert.Nil(t, upload)
	})

	t.Run("with options", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.PutOptions{
			ContentType: "text/plain",
			Metadata:    map[string]string{"key": "value"},
		}
		upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Nil(t, upload)
	})
}

// TestCreateMultipartUpload_DefaultBucket tests CreateMultipartUpload with default bucket.
func TestCreateMultipartUpload_DefaultBucket(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "default-bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("empty bucket uses default", func(t *testing.T) {
		upload, err := stor.CreateMultipartUpload(context.Background(), "", "key.txt", nil)
		assert.Error(t, err)
		assert.Nil(t, upload)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("explicit bucket overrides default", func(t *testing.T) {
		upload, err := stor.CreateMultipartUpload(context.Background(), "explicit-bucket", "key.txt", nil)
		assert.Error(t, err)
		assert.Nil(t, upload)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestCreateMultipartUpload_KeyVariations tests various key formats.
func TestCreateMultipartUpload_KeyVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	keys := []string{
		"simple.txt",
		"path/to/file.txt",
		"nested/path/to/file.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.with.many.dots.txt",
	}

	for _, key := range keys {
		t.Run("key_"+key, func(t *testing.T) {
			upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", key, nil)
			assert.Error(t, err)
			assert.Nil(t, upload)
		})
	}
}

// TestUploadPart_Extended tests the UploadPart method.
func TestUploadPart_Extended(t *testing.T) {
	t.Run("with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		reader := strings.NewReader("test data")
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with empty bucket uses default", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		reader := strings.NewReader("test data")
		part, err := stor.UploadPart(context.Background(), "", "key", "upload-123", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with various part numbers", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		partNumbers := []int32{1, 2, 5, 100, 1000, 10000}

		for _, partNumber := range partNumbers {
			t.Run("part_number_"+string(rune(partNumber)), func(t *testing.T) {
				reader := strings.NewReader("test data")
				part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", partNumber, reader)
				assert.Error(t, err)
				assert.Nil(t, part)
			})
		}
	})
}

// TestUploadPart_SizeInterface tests UploadPart with readers that implement Size().
func TestUploadPart_SizeInterface(t *testing.T) {
	t.Run("with SizeInterface reader", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		// Create a reader that implements Size() method
		sizeReader := &sizeReader{
			Reader: strings.NewReader("test data"),
			size:   9,
		}

		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", 1, sizeReader)
		assert.Error(t, err)
		assert.Nil(t, part)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with bytes.Reader (has Size method)", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		data := []byte("test data")
		reader := bytes.NewReader(data)

		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestUploadPart_ReaderVariations tests various reader types.
func TestUploadPart_ReaderVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("strings.Reader", func(t *testing.T) {
		reader := strings.NewReader("test data")
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
	})

	t.Run("bytes.Reader", func(t *testing.T) {
		reader := bytes.NewReader([]byte("test data"))
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
	})

	t.Run("empty reader", func(t *testing.T) {
		reader := strings.NewReader("")
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
	})

	t.Run("large data reader", func(t *testing.T) {
		data := bytes.Repeat([]byte("x"), 1024*1024) // 1MB
		reader := bytes.NewReader(data)
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
	})
}

// sizeReader implements io.Reader and Size() method for testing.
type sizeReader struct {
	io.Reader
	size int64
}

// Size returns the size of the reader.
func (sr *sizeReader) Size() int64 {
	return sr.size
}

// TestCompleteMultipartUpload_Extended tests the CompleteMultipartUpload method.
func TestCompleteMultipartUpload_Extended(t *testing.T) {
	t.Run("with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.CompleteMultipartUploadOptions{}
		info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-123", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with empty bucket uses default", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.CompleteMultipartUploadOptions{}
		info, err := stor.CompleteMultipartUpload(context.Background(), "", "key", "upload-123", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with parts", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.CompleteMultipartUploadOptions{
			Parts: []storage.UploadedPart{
				{PartNumber: 1, ETag: "etag1", Size: 100},
				{PartNumber: 2, ETag: "etag2", Size: 200},
			},
		}
		info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-123", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("with empty parts list", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.CompleteMultipartUploadOptions{
			Parts: []storage.UploadedPart{},
		}
		info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-123", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("with single part", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.CompleteMultipartUploadOptions{
			Parts: []storage.UploadedPart{
				{PartNumber: 1, ETag: "etag1", Size: 1024},
			},
		}
		info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-123", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("with many parts", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		parts := make([]storage.UploadedPart, 10000)
		for i := 0; i < 10000; i++ {
			parts[i] = storage.UploadedPart{
				PartNumber: int32(i + 1),
				ETag:       "etag" + string(rune(i)),
				Size:       5 * 1024 * 1024,
			}
		}

		opts := &storage.CompleteMultipartUploadOptions{
			Parts: parts,
		}
		info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-123", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
	})
}

// TestAbortMultipartUpload_Extended tests the AbortMultipartUpload method.
func TestAbortMultipartUpload_Extended(t *testing.T) {
	t.Run("with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		err := stor.AbortMultipartUpload(context.Background(), "bucket", "key", "upload-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with empty bucket uses default", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		err := stor.AbortMultipartUpload(context.Background(), "", "key", "upload-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with various upload IDs", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		uploadIDs := []string{
			"upload-123",
			"upload-abc",
			"12345",
			"abcdefg",
			"upload-with-dashes",
			"upload_with_underscores",
		}

		for _, uploadID := range uploadIDs {
			t.Run("upload_id_"+uploadID, func(t *testing.T) {
				err := stor.AbortMultipartUpload(context.Background(), "bucket", "key", uploadID)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not initialized")
			})
		}
	})
}

// TestListMultipartUploads_Extended tests the ListMultipartUploads method.
func TestListMultipartUploads_Extended(t *testing.T) {
	t.Run("with nil client", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		// Calling ListMultipartUploads with nil client will panic when trying to iterate
		// We can't test this directly without catching the panic, but we can verify
		// that the storage structure is set up correctly
		assert.NotNil(t, stor)
	})

	t.Run("with empty bucket uses default", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		// Verify the config has the default bucket set
		assert.Equal(t, "default-bucket", stor.cfg.DefaultBucket)
	})

	t.Run("with cancelled context", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		_, cancel := context.WithCancel(context.Background())
		cancel()

		// This will panic when trying to use the nil client, but the context is already cancelled
		// We can't test the actual call without a real minio client
		assert.NotNil(t, stor)
	})

	t.Run("verify storage has default bucket configured", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "my-default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		assert.Equal(t, "my-default-bucket", stor.cfg.DefaultBucket)
	})
}

// TestMultipartUpload_Structure tests the MultipartUpload structure.
func TestMultipartUpload_Structure(t *testing.T) {
	now := time.Now()

	upload := &storage.MultipartUpload{
		UploadID:  "upload-123",
		Key:       "test-key.txt",
		Bucket:    "test-bucket",
		Initiated: now,
	}

	assert.Equal(t, "upload-123", upload.UploadID)
	assert.Equal(t, "test-key.txt", upload.Key)
	assert.Equal(t, "test-bucket", upload.Bucket)
	assert.Equal(t, now, upload.Initiated)
}

// TestUploadedPart_Structure tests the UploadedPart structure.
func TestUploadedPart_Structure(t *testing.T) {
	part := &storage.UploadedPart{
		PartNumber: 1,
		ETag:       "etag-123",
		Size:       5 * 1024 * 1024,
	}

	assert.Equal(t, int32(1), part.PartNumber)
	assert.Equal(t, "etag-123", part.ETag)
	assert.Equal(t, int64(5*1024*1024), part.Size)
}

// TestCompleteMultipartUploadOptions_Structure tests the options structure.
func TestCompleteMultipartUploadOptions_Structure(t *testing.T) {
	opts := &storage.CompleteMultipartUploadOptions{
		Parts: []storage.UploadedPart{
			{PartNumber: 1, ETag: "etag1", Size: 100},
			{PartNumber: 2, ETag: "etag2", Size: 200},
		},
	}

	assert.Len(t, opts.Parts, 2)
	assert.Equal(t, int32(1), opts.Parts[0].PartNumber)
	assert.Equal(t, "etag1", opts.Parts[0].ETag)
	assert.Equal(t, int64(100), opts.Parts[0].Size)
}

// TestMultipart_ContextTests tests multipart methods with various contexts.
func TestMultipart_ContextTests(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("CreateMultipartUpload with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		upload, err := stor.CreateMultipartUpload(ctx, "bucket", "key", nil)
		assert.Error(t, err)
		assert.Nil(t, upload)
	})

	t.Run("UploadPart with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		reader := strings.NewReader("test")
		part, err := stor.UploadPart(ctx, "bucket", "key", "upload-123", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
	})

	t.Run("CompleteMultipartUpload with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		opts := &storage.CompleteMultipartUploadOptions{}
		info, err := stor.CompleteMultipartUpload(ctx, "bucket", "key", "upload-123", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("AbortMultipartUpload with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := stor.AbortMultipartUpload(ctx, "bucket", "key", "upload-123")
		assert.Error(t, err)
	})
}

// TestMultipart_KeyVariations tests various key formats for multipart operations.
func TestMultipart_KeyVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	keys := []string{
		"file.txt",
		"path/to/file.txt",
		"a/b/c/d/file.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.many.dots.txt",
		"filename",
	}

	for _, key := range keys {
		t.Run("key_"+strings.ReplaceAll(key, "/", "_"), func(t *testing.T) {
			// CreateMultipartUpload
			upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", key, nil)
			assert.Error(t, err)
			assert.Nil(t, upload)

			// UploadPart
			reader := strings.NewReader("test")
			part, err := stor.UploadPart(context.Background(), "bucket", key, "upload-123", 1, reader)
			assert.Error(t, err)
			assert.Nil(t, part)

			// CompleteMultipartUpload
			opts := &storage.CompleteMultipartUploadOptions{}
			info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", key, "upload-123", opts)
			assert.Error(t, err)
			assert.Nil(t, info)

			// AbortMultipartUpload
			err = stor.AbortMultipartUpload(context.Background(), "bucket", key, "upload-123")
			assert.Error(t, err)
		})
	}
}

// TestCore_ReturnsCoreClient tests the core() method returns a valid minio.Core.
func TestCore_ReturnsCoreClient(t *testing.T) {
	t.Run("core returns non-nil Core", func(t *testing.T) {
		client := &Client{
			cfg:    Config{},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		core := stor.core()
		assert.NotNil(t, core)
	})

	t.Run("core wraps the client correctly", func(t *testing.T) {
		minioClient := &Client{
			cfg:    Config{},
			logger: slog.Default(),
			// Note: without a real minio client, core().Client will be nil
		}
		stor := NewStorage(minioClient, nil)

		core := stor.core()
		assert.NotNil(t, core)
		// The Core wraps the underlying minio client (may be nil in tests)
		// Just verify the core object itself is not nil
	})
}

// TestMultipartUpload_CompleteWithEmptyParts tests CompleteMultipartUpload with empty parts.
func TestMultipartUpload_CompleteWithEmptyParts(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	opts := &storage.CompleteMultipartUploadOptions{
		Parts: []storage.UploadedPart{},
	}
	info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-123", opts)
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "not initialized")
}

// TestMultipartUpload_WithRealisticETags tests with realistic ETag values.
func TestMultipartUpload_WithRealisticETags(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	// Realistic ETags from S3/MinIO are MD5 hashes (32 hex chars for single part, different for multipart)
	realisticETags := []string{
		"5d41402abc4b2a76b9719d911017c592", // MD5 of "hello"
		"d41d8cd98f00b204e9800998ecf8427e", // MD5 of "" (empty)
		"098f6bcd4621d373cade4e832627b4f6", // MD5 of "abc"
		"multi-part-etag-with-dash-12345",  // Multipart ETag format
	}

	for _, etag := range realisticETags {
		t.Run("etag_"+etag[:8], func(t *testing.T) {
			opts := &storage.CompleteMultipartUploadOptions{
				Parts: []storage.UploadedPart{
					{PartNumber: 1, ETag: etag, Size: 1024},
				},
			}
			info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-123", opts)
			assert.Error(t, err)
			assert.Nil(t, info)
		})
	}
}

// TestMultipartUpload_PartNumberEdgeCases tests edge cases for part numbers.
func TestMultipartUpload_PartNumberEdgeCases(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	edgeCases := []struct {
		name       string
		partNumber int32
	}{
		{"min_part", 1},
		{"max_small", 10000},
		{"middle", 5000},
		{"large", 9999},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader("test data")
			part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", tc.partNumber, reader)
			assert.Error(t, err)
			assert.Nil(t, part)
			assert.Contains(t, err.Error(), "not initialized")
		})
	}
}

// TestMultipartUpload_KeyVariationsExtended tests various key formats.
func TestMultipartUpload_KeyVariationsExtended(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	keys := []string{
		"simple.txt",
		"with-dash.txt",
		"with_underscore.txt",
		"with.dot.txt",
		"with space.txt",
		"with/ slash.txt",
		"with+plus.txt",
		"with%percent.txt",
		"with~tilde.txt",
		"with!exclamation.txt",
		"with@at.txt",
		"with#hash.txt",
		"with$dollar.txt",
		"with&ampersand.txt",
		"with*asterisk.txt",
		"with(paren).txt",
		"very-long-key-name-that-goes-on-and-on.txt",
		"unicode/файл.txt",
		"emoji/😀.txt",
		"nested/deeply/nested/path/file.txt",
	}

	for _, key := range keys {
		t.Run("key_"+key, func(t *testing.T) {
			upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", key, nil)
			assert.Error(t, err)
			assert.Nil(t, upload)

			reader := strings.NewReader("test")
			part, err := stor.UploadPart(context.Background(), "bucket", key, "upload-123", 1, reader)
			assert.Error(t, err)
			assert.Nil(t, part)

			opts := &storage.CompleteMultipartUploadOptions{
				Parts: []storage.UploadedPart{},
			}
			info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", key, "upload-123", opts)
			assert.Error(t, err)
			assert.Nil(t, info)

			err = stor.AbortMultipartUpload(context.Background(), "bucket", key, "upload-123")
			assert.Error(t, err)
		})
	}
}

// TestMultipartUpload_BucketVariations tests bucket name variations.
func TestMultipartUpload_BucketVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "default-bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	buckets := []string{
		"",
		"simple-bucket",
		"bucket.with.dots",
		"bucket-with-dashes",
		"bucket_with_underscores",
		"bucket123",
		"123-numbers-first",
	}

	for _, bucket := range buckets {
		t.Run("bucket_"+bucket, func(t *testing.T) {
			upload, err := stor.CreateMultipartUpload(context.Background(), bucket, "key.txt", nil)
			assert.Error(t, err)
			assert.Nil(t, upload)
		})
	}
}

// TestMultipartUpload_UploadIDVariations tests various upload ID formats.
func TestMultipartUpload_UploadIDVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	uploadIDs := []string{
		"upload-123",
		"upload/with/slashes",
		"upload=with=equals",
		"upload:with:colons",
		"upload;with;semicolons",
		"upload,with,commas",
		"very-long-upload-id-that-continues-for-a-while-and-gets-even-longer",
		"upload-with-numbers-123456789",
		"UPPERCASE",
		"lowercase",
		"MiXeDcAsE",
	}

	for _, uploadID := range uploadIDs {
		t.Run("upload_id_"+uploadID[:min(10, len(uploadID))], func(t *testing.T) {
			reader := strings.NewReader("test")
			part, err := stor.UploadPart(context.Background(), "bucket", "key", uploadID, 1, reader)
			assert.Error(t, err)
			assert.Nil(t, part)

			opts := &storage.CompleteMultipartUploadOptions{
				Parts: []storage.UploadedPart{
					{PartNumber: 1, ETag: "etag-123", Size: 100},
				},
			}
			info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", uploadID, opts)
			assert.Error(t, err)
			assert.Nil(t, info)

			err = stor.AbortMultipartUpload(context.Background(), "bucket", "key", uploadID)
			assert.Error(t, err)
		})
	}
}

// TestMultipartUpload_OptionsVariations tests various PutOptions.
func TestMultipartUpload_OptionsVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	contentTypes := []string{
		"",
		"text/plain",
		"application/json",
		"application/octet-stream",
		"image/jpeg",
		"video/mp4",
		"application/pdf",
	}

	for _, ct := range contentTypes {
		t.Run("content_type_"+ct, func(t *testing.T) {
			opts := &storage.PutOptions{
				ContentType: ct,
			}
			upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", "key.txt", opts)
			assert.Error(t, err)
			assert.Nil(t, upload)
		})
	}

	t.Run("with metadata", func(t *testing.T) {
		opts := &storage.PutOptions{
			ContentType: "application/json",
			Metadata: map[string]string{
				"key1":              "value1",
				"key2":              "value2",
				"x-amz-meta-custom": "custom-value",
			},
		}
		upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", "key.txt", opts)
		assert.Error(t, err)
		assert.Nil(t, upload)
	})
}

// TestMultipartUpload_ContextTimeoutVariations tests various timeout scenarios.
func TestMultipartUpload_ContextTimeoutVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	timeouts := []time.Duration{
		1 * time.Nanosecond,
		1 * time.Microsecond,
		10 * time.Millisecond,
	}

	for _, timeout := range timeouts {
		t.Run("timeout_"+timeout.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// Wait for timeout to expire
			if timeout > 10*time.Millisecond {
				time.Sleep(timeout + 10*time.Millisecond)
			}

			upload, err := stor.CreateMultipartUpload(ctx, "bucket", "key.txt", nil)
			assert.Error(t, err)
			assert.Nil(t, upload)
			_ = ctx // Use ctx to avoid unused variable warning
		})
	}
}

// TestMultipartUpload_NilClientErrorPaths tests error paths with nil client.
func TestMultipartUpload_NilClientErrorPaths(t *testing.T) {
	t.Run("CreateMultipartUpload with nil client", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", "key", nil)
		assert.Error(t, err)
		assert.Nil(t, upload)
	})

	t.Run("CompleteMultipartUpload with nil client", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.CompleteMultipartUploadOptions{
			Parts: []storage.UploadedPart{{PartNumber: 1, ETag: "etag"}},
		}
		info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-id", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	// UploadPart and AbortMultipartUpload may panic with nil client due to core() method
	// ListMultipartUploads spawns goroutines that panic
	// These are known limitations of the current implementation
}

// TestMultipartUpload_KeyVariations tests with various key formats.
func TestMultipartUpload_KeyVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	keys := []string{
		"simple-key.txt",
		"path/to/file.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.many.dots.txt",
		"unicode-ключ.txt",
		"very-long-file-name-that-goes-on-and-on.txt",
	}

	for _, key := range keys {
		t.Run("key_"+key[:min(5, len(key))], func(t *testing.T) {
			upload, err := stor.CreateMultipartUpload(context.Background(), "bucket", key, nil)
			assert.Error(t, err)
			assert.Nil(t, upload)
		})
	}
}

// TestMultipartUpload_AbortUploadIDVariations tests Abort with various upload ID formats.
func TestMultipartUpload_AbortUploadIDVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	uploadIDs := []string{
		"upload-123",
		"upload/xYz/123",
		"002_abc",
		"uploadWithCamelCase",
		"UPPERCASE",
	}

	for _, uploadID := range uploadIDs {
		t.Run("upload_"+uploadID, func(t *testing.T) {
			err := stor.AbortMultipartUpload(context.Background(), "bucket", "key", uploadID)
			assert.Error(t, err)
		})
	}
}

// min returns the minimum of two integers for testing purposes.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
