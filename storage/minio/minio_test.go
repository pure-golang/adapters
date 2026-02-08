package minio

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pure-golang/adapters/storage"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig tests Config struct.
func TestConfig(t *testing.T) {
	cfg := Config{
		Endpoint:  "localhost:9000",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Region:    "us-east-1",
		Secure:    true,
	}

	assert.Equal(t, "localhost:9000", cfg.Endpoint)
	assert.Equal(t, "minioadmin", cfg.AccessKey)
	assert.Equal(t, "minioadmin", cfg.SecretKey)
	assert.Equal(t, "us-east-1", cfg.Region)
	assert.True(t, cfg.Secure)
}

// TestToStorageError tests the toStorageError function.
func TestToStorageError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		bucket       string
		key          string
		expectedCode storage.ErrorCode
	}{
		{
			name:         "nil error",
			err:          nil,
			bucket:       "bucket",
			key:          "key",
			expectedCode: "",
		},
		{
			name:         "generic error",
			err:          errors.New("some generic error"),
			bucket:       "bucket",
			key:          "key",
			expectedCode: storage.CodeInternalError,
		},
		{
			name:         "NoSuchKey error",
			err:          errors.New("The specified key does not exist: NoSuchKey"),
			bucket:       "bucket",
			key:          "key",
			expectedCode: storage.CodeNotFound,
		},
		{
			name:         "not found error",
			err:          errors.New("object not found"),
			bucket:       "bucket",
			key:          "key",
			expectedCode: storage.CodeNotFound,
		},
		{
			name:         "NotFound error",
			err:          errors.New("BucketNotFound: something NotFound"),
			bucket:       "bucket",
			key:          "key",
			expectedCode: storage.CodeNotFound,
		},
		{
			name:         "NoSuchBucket error",
			err:          errors.New("NoSuchBucket: the bucket does not exist"),
			bucket:       "bucket",
			key:          "key",
			expectedCode: storage.CodeBucketNotFound,
		},
		{
			name:         "AccessDenied error",
			err:          errors.New("AccessDenied: access denied"),
			bucket:       "bucket",
			key:          "key",
			expectedCode: storage.CodeAccessDenied,
		},
		{
			name:         "Forbidden error",
			err:          errors.New("403 Forbidden"),
			bucket:       "bucket",
			key:          "key",
			expectedCode: storage.CodeAccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := toStorageError(tt.err, tt.bucket, tt.key)
			if tt.expectedCode == "" {
				assert.Nil(t, err)
			} else {
				require.NotNil(t, err)
				storageErr, ok := err.(*storage.StorageError)
				require.True(t, ok)
				assert.Equal(t, tt.expectedCode, storageErr.Code)
				assert.Equal(t, tt.bucket, storageErr.Bucket)
				assert.Equal(t, tt.key, storageErr.Key)
			}
		})
	}
}

// TestIsNotFoundError tests the isNotFoundError function.
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some generic error"),
			expected: false,
		},
		{
			name:     "NoSuchKey error",
			err:      errors.New("NoSuchKey: key not found"),
			expected: true,
		},
		{
			name:     "NotFound error",
			err:      errors.New("object NotFound"),
			expected: true,
		},
		{
			name:     "not found error",
			err:      errors.New("not found"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isNotFoundError(tt.err))
		})
	}
}

// TestNewClient tests the NewClient function.
func TestNewClient(t *testing.T) {
	t.Run("with nil options fails to connect", func(t *testing.T) {
		cfg := Config{
			Endpoint:  "invalid-endpoint:9999",
			AccessKey: "test",
			SecretKey: "test",
		}

		client, err := NewClient(cfg, nil)
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

// TestNewDefaultClient tests the NewDefaultClient function.
func TestNewDefaultClient(t *testing.T) {
	cfg := Config{
		Endpoint:  "invalid-endpoint:9999",
		AccessKey: "test",
		SecretKey: "test",
	}

	client, err := NewDefaultClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
}

// TestClient_Close tests the Client Close method.
func TestClient_Close(t *testing.T) {
	t.Run("close once", func(t *testing.T) {
		client := &Client{
			logger: slog.Default(),
			closed: false,
		}

		err := client.Close()
		assert.NoError(t, err)
		assert.True(t, client.IsClosed())
	})

	t.Run("close twice returns nil", func(t *testing.T) {
		client := &Client{
			logger: slog.Default(),
			closed: false,
		}

		err1 := client.Close()
		err2 := client.Close()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.True(t, client.IsClosed())
	})

	t.Run("IsClosed before close", func(t *testing.T) {
		client := &Client{
			logger: slog.Default(),
			closed: false,
		}

		assert.False(t, client.IsClosed())
		if err := client.Close(); err != nil {
			t.Logf("Warning: failed to close client: %v", err)
		}
		assert.True(t, client.IsClosed())
	})
}

// TestGetMinioClient tests the GetMinioClient method.
func TestGetMinioClient(t *testing.T) {
	minioClient := &minio.Client{}
	client := &Client{
		client: minioClient,
	}

	assert.Equal(t, minioClient, client.GetMinioClient())
}

// TestNewStorage tests the NewStorage function.
func TestNewStorage(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		client := &Client{
			cfg: Config{DefaultBucket: "test-bucket"},
		}

		stor := NewStorage(client, nil)
		require.NotNil(t, stor)

		assert.Equal(t, client, stor.client)
		assert.Equal(t, "test-bucket", stor.cfg.DefaultBucket)
		assert.NotNil(t, stor.logger)
	})

	t.Run("with nil logger in options", func(t *testing.T) {
		client := &Client{
			cfg: Config{DefaultBucket: "test-bucket"},
		}

		stor := NewStorage(client, &StorageOptions{})
		require.NotNil(t, stor)
		assert.NotNil(t, stor.logger)
	})

	t.Run("with custom logger", func(t *testing.T) {
		client := &Client{
			cfg: Config{DefaultBucket: "test-bucket"},
		}
		logger := slog.Default()

		stor := NewStorage(client, &StorageOptions{Logger: logger})
		require.NotNil(t, stor)
		assert.NotNil(t, stor.logger)
	})
}

// TestNewDefault tests the NewDefault function.
func TestNewDefault(t *testing.T) {
	t.Run("with invalid config", func(t *testing.T) {
		cfg := Config{
			Endpoint:  "invalid-endpoint:9999",
			AccessKey: "test",
			SecretKey: "test",
		}

		storage, err := NewDefault(cfg)
		assert.Error(t, err)
		assert.Nil(t, storage)
	})
}

// TestStorage_Close tests the Storage Close method.
func TestStorage_Close(t *testing.T) {
	t.Run("close calls client close", func(t *testing.T) {
		client := &Client{
			cfg:    Config{},
			logger: slog.Default(),
			closed: false,
		}

		stor := NewStorage(client, nil)
		err := stor.Close()

		assert.NoError(t, err)
		assert.True(t, client.IsClosed())
	})
}

// TestStorageInterface verifies Storage implements storage.Storage.
func TestStorageInterface(t *testing.T) {
	// Compile-time check that Storage implements storage.Storage
	var _ storage.Storage = (*Storage)(nil)
	var _ io.Closer = (*Storage)(nil)
}

// TestCloserInterface verifies Client implements Closer.
func TestCloserInterface(t *testing.T) {
	// Compile-time check that Client implements Closer
	var _ Closer = (*Client)(nil)
}

// TestStorage_Put tests the Put method with nil client.
func TestStorage_Put(t *testing.T) {
	t.Run("put with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		err := stor.Put(context.Background(), "bucket", "key", nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_Get tests the Get method with nil client.
func TestStorage_Get(t *testing.T) {
	t.Run("get with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		rc, info, err := stor.Get(context.Background(), "bucket", "key")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_Delete tests the Delete method with nil client.
func TestStorage_Delete(t *testing.T) {
	t.Run("delete with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		err := stor.Delete(context.Background(), "bucket", "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_Exists tests the Exists method with nil client.
func TestStorage_Exists(t *testing.T) {
	t.Run("exists with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		exists, err := stor.Exists(context.Background(), "bucket", "key")
		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_List tests the List method.
func TestStorage_List(t *testing.T) {
	t.Run("list with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		result, err := stor.List(context.Background(), "bucket", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestGetPresignedURL tests the GetPresignedURL method.
func TestGetPresignedURL(t *testing.T) {
	t.Run("unsupported method returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.PresignedURLOptions{
			Method: "DELETE",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "unsupported HTTP method")
	})

	t.Run("GET method with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("PUT method with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		opts := &storage.PresignedURLOptions{
			Method: "PUT",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestCore tests the core method.
func TestCore(t *testing.T) {
	client := &Client{
		cfg:    Config{},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	core := stor.core()
	assert.NotNil(t, core)
}

// TestCreateMultipartUpload tests the CreateMultipartUpload method.
func TestCreateMultipartUpload(t *testing.T) {
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
}

// TestUploadPart tests the UploadPart method.
func TestUploadPart(t *testing.T) {
	t.Run("with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		// Provide a non-nil reader to avoid panic in io.ReadAll
		reader := strings.NewReader("test data")
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-123", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestCompleteMultipartUpload tests the CompleteMultipartUpload method.
func TestCompleteMultipartUpload(t *testing.T) {
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
}

// TestAbortMultipartUpload tests the AbortMultipartUpload method.
func TestAbortMultipartUpload(t *testing.T) {
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
}

// TestListMultipartUploads tests the ListMultipartUploads method.
func TestListMultipartUploads(t *testing.T) {
	t.Run("with nil client panics", func(t *testing.T) {
		t.Skip("ListIncompleteUploads spawns goroutines that panic cannot be caught")
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		// nolint:errcheck // Test is skipped, error intentionally ignored
		_, _ = stor.ListMultipartUploads(context.Background(), "bucket")
	})
}

// Example usage
func ExampleStorage() {
	// This example demonstrates how to use the MinIO adapter
	cfg := Config{
		Endpoint:  "localhost:9000",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Secure:    false,
	}

	storage, err := NewDefault(cfg)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			// Log error in production
		}
	}()

	_ = storage.Put(context.Background(), "bucket", "key", nil, nil)
}
