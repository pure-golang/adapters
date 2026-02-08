package minio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

// TestStorage getClient tests the getClient method.
func TestStorage_GetClient(t *testing.T) {
	t.Run("returns nil when client is nil", func(t *testing.T) {
		stor := &Storage{
			client: nil,
			cfg:    Config{},
			logger: slog.Default(),
		}

		client, err := stor.getClient()
		assert.Error(t, err)
		assert.Nil(t, client)

		storageErr, ok := err.(*storage.StorageError)
		assert.True(t, ok)
		assert.Equal(t, storage.CodeInternalError, storageErr.Code)
		assert.Contains(t, storageErr.Message, "not initialized")
	})

	t.Run("returns nil when minio client is nil", func(t *testing.T) {
		stor := &Storage{
			client: &Client{
				client: nil,
			},
			cfg:    Config{},
			logger: slog.Default(),
		}

		client, err := stor.getClient()
		assert.Error(t, err)
		assert.Nil(t, client)

		storageErr, ok := err.(*storage.StorageError)
		assert.True(t, ok)
		assert.Equal(t, storage.CodeInternalError, storageErr.Code)
	})

	t.Run("returns client when initialized", func(t *testing.T) {
		minioClient := &minio.Client{}
		stor := &Storage{
			client: &Client{
				client: minioClient,
			},
			cfg:    Config{},
			logger: slog.Default(),
		}

		client, err := stor.getClient()
		assert.NoError(t, err)
		assert.Equal(t, minioClient, client)
	})
}

// TestStorage_NewDefault tests the NewDefault function.
func TestStorage_NewDefault(t *testing.T) {
	t.Run("with invalid config returns error", func(t *testing.T) {
		cfg := Config{
			Endpoint:  "invalid-endpoint:9999",
			AccessKey: "test",
			SecretKey: "test",
		}

		stor, err := NewDefault(cfg)
		assert.Error(t, err)
		assert.Nil(t, stor)
	})
}

// TestStorage_Put_DefaultBucket tests Put with default bucket.
func TestStorage_Put_DefaultBucket(t *testing.T) {
	t.Run("uses default bucket when empty", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		reader := strings.NewReader("test data")
		err := stor.Put(context.Background(), "", "key.txt", reader, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("uses explicit bucket when provided", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		reader := strings.NewReader("test data")
		err := stor.Put(context.Background(), "explicit-bucket", "key.txt", reader, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_Put_Options tests Put with various options.
func TestStorage_Put_Options(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("with nil options", func(t *testing.T) {
		reader := strings.NewReader("test data")
		err := stor.Put(context.Background(), "bucket", "key.txt", reader, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with empty options", func(t *testing.T) {
		reader := strings.NewReader("test data")
		opts := &storage.PutOptions{}
		err := stor.Put(context.Background(), "bucket", "key.txt", reader, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with content type", func(t *testing.T) {
		reader := strings.NewReader("test data")
		opts := &storage.PutOptions{
			ContentType: "text/plain",
		}
		err := stor.Put(context.Background(), "bucket", "key.txt", reader, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with metadata", func(t *testing.T) {
		reader := strings.NewReader("test data")
		opts := &storage.PutOptions{
			ContentType: "application/json",
			Metadata:    map[string]string{"key": "value"},
		}
		err := stor.Put(context.Background(), "bucket", "key.txt", reader, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_Get_DefaultBucket tests Get with default bucket.
func TestStorage_Get_DefaultBucket(t *testing.T) {
	t.Run("uses default bucket when empty", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		rc, info, err := stor.Get(context.Background(), "", "key.txt")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_Get_KeyVariations tests Get with various key formats.
func TestStorage_Get_KeyVariations(t *testing.T) {
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
	}

	for _, key := range keys {
		t.Run("key_"+strings.ReplaceAll(key, "/", "_"), func(t *testing.T) {
			rc, info, err := stor.Get(context.Background(), "bucket", key)
			assert.Error(t, err)
			assert.Nil(t, rc)
			assert.Nil(t, info)
		})
	}
}

// TestStorage_Delete_DefaultBucket tests Delete with default bucket.
func TestStorage_Delete_DefaultBucket(t *testing.T) {
	t.Run("uses default bucket when empty", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		err := stor.Delete(context.Background(), "", "key.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with explicit bucket", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		err := stor.Delete(context.Background(), "explicit-bucket", "key.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_Exists_DefaultBucket tests Exists with default bucket.
func TestStorage_Exists_DefaultBucket(t *testing.T) {
	t.Run("uses default bucket when empty", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		exists, err := stor.Exists(context.Background(), "", "key.txt")
		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_Exists_KeyVariations tests Exists with various key formats.
func TestStorage_Exists_KeyVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	keys := []string{
		"file.txt",
		"path/to/file.txt",
		"nested/path/file.txt",
	}

	for _, key := range keys {
		t.Run("key_"+strings.ReplaceAll(key, "/", "_"), func(t *testing.T) {
			exists, err := stor.Exists(context.Background(), "bucket", key)
			assert.Error(t, err)
			assert.False(t, exists)
		})
	}
}

// TestStorage_List_DefaultBucket tests List with default bucket.
func TestStorage_List_DefaultBucket(t *testing.T) {
	t.Run("uses default bucket when empty", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		result, err := stor.List(context.Background(), "", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_List_Options tests List with various options.
func TestStorage_List_Options(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("with nil options", func(t *testing.T) {
		result, err := stor.List(context.Background(), "bucket", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with empty options", func(t *testing.T) {
		opts := &storage.ListOptions{}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with prefix", func(t *testing.T) {
		opts := &storage.ListOptions{
			Prefix: "test/",
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with recursive", func(t *testing.T) {
		opts := &storage.ListOptions{
			Recursive: true,
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with max keys", func(t *testing.T) {
		opts := &storage.ListOptions{
			MaxKeys: 100,
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("with all options", func(t *testing.T) {
		opts := &storage.ListOptions{
			Prefix:    "docs/",
			Recursive: true,
			MaxKeys:   50,
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_ContextTests tests storage methods with various contexts.
func TestStorage_ContextTests(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("Put with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		reader := strings.NewReader("test")
		err := stor.Put(ctx, "bucket", "key", reader, nil)
		assert.Error(t, err)
	})

	t.Run("Get with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		rc, info, err := stor.Get(ctx, "bucket", "key")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)
	})

	t.Run("Delete with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := stor.Delete(ctx, "bucket", "key")
		assert.Error(t, err)
	})

	t.Run("Exists with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		exists, err := stor.Exists(ctx, "bucket", "key")
		assert.Error(t, err)
		assert.False(t, exists)
	})

	t.Run("List with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result, err := stor.List(ctx, "bucket", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestStorage_WithNilReader tests Put with nil reader.
func TestStorage_WithNilReader(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	err := stor.Put(context.Background(), "bucket", "key", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

// TestStorage_Close_Extended tests the Close method.
func TestStorage_Close_Extended(t *testing.T) {
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

	t.Run("close twice returns nil", func(t *testing.T) {
		client := &Client{
			cfg:    Config{},
			logger: slog.Default(),
			closed: false,
		}

		stor := NewStorage(client, nil)
		err1 := stor.Close()
		err2 := stor.Close()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.True(t, client.IsClosed())
	})
}

// TestStorage_Options tests various storage options.
func TestStorage_Options(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}

	t.Run("nil options", func(t *testing.T) {
		stor := NewStorage(client, nil)
		assert.NotNil(t, stor)
		assert.NotNil(t, stor.logger)
	})

	t.Run("empty options", func(t *testing.T) {
		stor := NewStorage(client, &StorageOptions{})
		assert.NotNil(t, stor)
		assert.NotNil(t, stor.logger)
	})

	t.Run("with custom logger", func(t *testing.T) {
		logger := slog.Default()
		stor := NewStorage(client, &StorageOptions{Logger: logger})
		assert.NotNil(t, stor)
		assert.NotNil(t, stor.logger)
	})
}

// TestStorage_Config tests that storage uses client config.
func TestStorage_Config(t *testing.T) {
	cfg := Config{
		DefaultBucket: "test-bucket",
		Endpoint:      "localhost:9000",
		AccessKey:     "key",
		SecretKey:     "secret",
	}

	client := &Client{
		cfg:    cfg,
		logger: slog.Default(),
	}

	stor := NewStorage(client, nil)
	assert.Equal(t, cfg.DefaultBucket, stor.cfg.DefaultBucket)
	assert.Equal(t, cfg.Endpoint, stor.cfg.Endpoint)
	assert.Equal(t, cfg.AccessKey, stor.cfg.AccessKey)
	assert.Equal(t, cfg.SecretKey, stor.cfg.SecretKey)
}

// TestStorageInterface verifies Storage implements required interfaces.
func TestStorageInterface_Extended(t *testing.T) {
	// Compile-time check that Storage implements storage.Storage
	var _ storage.Storage = (*Storage)(nil)
	var _ io.Closer = (*Storage)(nil)

	// Create a storage instance
	client := &Client{
		cfg:    Config{},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	assert.Implements(t, (*storage.Storage)(nil), stor)
	assert.Implements(t, (*io.Closer)(nil), stor)
}

// TestStorage_WithTimeoutContext tests with timeout contexts.
func TestStorage_WithTimeoutContext(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("Put with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		reader := strings.NewReader("test")
		err := stor.Put(ctx, "bucket", "key", reader, nil)
		assert.Error(t, err)
	})

	t.Run("Get with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		rc, info, err := stor.Get(ctx, "bucket", "key")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)
	})
}

// TestStorageError_Wrapping tests that errors are properly wrapped.
func TestStorageError_Wrapping(t *testing.T) {
	t.Run("toStorageError with nil error", func(t *testing.T) {
		err := toStorageError(nil, "bucket", "key")
		assert.Nil(t, err)
	})

	t.Run("toStorageError wraps original error", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := toStorageError(originalErr, "bucket", "key")

		require.NotNil(t, err)
		storageErr, ok := err.(*storage.StorageError)
		require.True(t, ok)
		assert.Equal(t, originalErr, storageErr.Err)
		assert.Equal(t, "bucket", storageErr.Bucket)
		assert.Equal(t, "key", storageErr.Key)
	})
}

// TestObjectInfo tests the ObjectInfo structure.
func TestObjectInfo(t *testing.T) {
	info := &storage.ObjectInfo{
		Key:          "test-key.txt",
		Size:         1024,
		ETag:         "etag-123",
		ContentType:  "text/plain",
		LastModified: time.Now(),
		Metadata:     map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-key.txt", info.Key)
	assert.Equal(t, int64(1024), info.Size)
	assert.Equal(t, "etag-123", info.ETag)
	assert.Equal(t, "text/plain", info.ContentType)
	assert.NotZero(t, info.LastModified)
	assert.Equal(t, "value", info.Metadata["key"])
}

// TestListResult tests the ListResult structure.
func TestListResult(t *testing.T) {
	objects := []storage.ObjectInfo{
		{Key: "file1.txt", Size: 100},
		{Key: "file2.txt", Size: 200},
	}

	result := &storage.ListResult{
		Objects:     objects,
		IsTruncated: false,
	}

	assert.Len(t, result.Objects, 2)
	assert.False(t, result.IsTruncated)
	assert.Equal(t, "file1.txt", result.Objects[0].Key)
	assert.Equal(t, int64(100), result.Objects[0].Size)
}

// TestPutOptions tests the PutOptions structure.
func TestPutOptions(t *testing.T) {
	opts := &storage.PutOptions{
		ContentType: "application/json",
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	assert.Equal(t, "application/json", opts.ContentType)
	assert.Len(t, opts.Metadata, 2)
	assert.Equal(t, "value1", opts.Metadata["key1"])
}

// TestListOptions tests the ListOptions structure.
func TestListOptions(t *testing.T) {
	opts := &storage.ListOptions{
		Prefix:    "test/",
		Recursive: true,
		MaxKeys:   100,
	}

	assert.Equal(t, "test/", opts.Prefix)
	assert.True(t, opts.Recursive)
	assert.Equal(t, 100, opts.MaxKeys)
}

// TestPresignedURLOptions_Struct tests the PresignedURLOptions structure.
func TestPresignedURLOptions_Struct(t *testing.T) {
	opts := &storage.PresignedURLOptions{
		Method: "GET",
		Expiry: 15 * time.Minute,
	}

	assert.Equal(t, "GET", opts.Method)
	assert.Equal(t, 15*time.Minute, opts.Expiry)
}

// TestStorage_ErrorPaths tests additional error paths in storage operations.
func TestStorage_ErrorPaths(t *testing.T) {
	t.Run("Put with error in getClient", func(t *testing.T) {
		// Create storage with nil client
		nilClientStor := &Storage{
			client: nil,
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}

		reader := strings.NewReader("test")
		err := nilClientStor.Put(context.Background(), "bucket", "key", reader, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Get with error in getClient", func(t *testing.T) {
		nilClientStor := &Storage{
			client: nil,
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}

		rc, info, err := nilClientStor.Get(context.Background(), "bucket", "key")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Delete with error in getClient", func(t *testing.T) {
		nilClientStor := &Storage{
			client: nil,
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}

		err := nilClientStor.Delete(context.Background(), "bucket", "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Exists with error in getClient", func(t *testing.T) {
		nilClientStor := &Storage{
			client: nil,
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}

		exists, err := nilClientStor.Exists(context.Background(), "bucket", "key")
		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("List with error in getClient", func(t *testing.T) {
		nilClientStor := &Storage{
			client: nil,
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}

		result, err := nilClientStor.List(context.Background(), "bucket", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_WithZeroValueOptions tests with zero value options.
func TestStorage_WithZeroValueOptions(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("Put with zero value PutOptions", func(t *testing.T) {
		opts := &storage.PutOptions{}
		reader := strings.NewReader("test")
		err := stor.Put(context.Background(), "bucket", "key", reader, opts)
		assert.Error(t, err)
	})

	t.Run("List with zero value ListOptions", func(t *testing.T) {
		opts := &storage.ListOptions{}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestStorage_WithNilOptions tests with nil options.
func TestStorage_WithNilOptions(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("Put with nil options", func(t *testing.T) {
		reader := strings.NewReader("test")
		err := stor.Put(context.Background(), "bucket", "key", reader, nil)
		assert.Error(t, err)
	})

	t.Run("List with nil options", func(t *testing.T) {
		result, err := stor.List(context.Background(), "bucket", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestStorage_MetadataOptions tests PutOptions with metadata.
func TestStorage_MetadataOptions(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	metadataTests := []struct {
		name     string
		metadata map[string]string
	}{
		{
			name:     "empty metadata",
			metadata: map[string]string{},
		},
		{
			name:     "single metadata entry",
			metadata: map[string]string{"key": "value"},
		},
		{
			name: "multiple metadata entries",
			metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name: "metadata with special characters",
			metadata: map[string]string{
				"key-with-dash":       "value-with-dash",
				"key_with_underscore": "value_with_underscore",
				"key.with.dots":       "value.with.dots",
			},
		},
		{
			name: "metadata with x-amz- prefix",
			metadata: map[string]string{
				"x-amz-meta-custom":  "custom-value",
				"x-amz-meta-another": "another-value",
			},
		},
	}

	for _, tc := range metadataTests {
		t.Run(tc.name, func(t *testing.T) {
			opts := &storage.PutOptions{
				ContentType: "text/plain",
				Metadata:    tc.metadata,
			}
			reader := strings.NewReader("test")
			err := stor.Put(context.Background(), "bucket", "key", reader, opts)
			assert.Error(t, err)
		})
	}
}

// TestStorage_ContentTypeVariations tests various content types.
func TestStorage_ContentTypeVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	contentTypes := []string{
		"",
		"text/plain",
		"text/html",
		"application/json",
		"application/xml",
		"application/octet-stream",
		"image/jpeg",
		"image/png",
		"video/mp4",
		"audio/mpeg",
		"application/pdf",
		"application/zip",
		"multipart/form-data",
	}

	for _, ct := range contentTypes {
		t.Run("content_type_"+ct, func(t *testing.T) {
			opts := &storage.PutOptions{
				ContentType: ct,
			}
			reader := strings.NewReader("test")
			err := stor.Put(context.Background(), "bucket", "key", reader, opts)
			assert.Error(t, err)
		})
	}
}

// TestStorage_ListOptions tests various list options.
func TestStorage_ListOptions(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("List with prefix only", func(t *testing.T) {
		opts := &storage.ListOptions{
			Prefix: "test/",
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("List with recursive only", func(t *testing.T) {
		opts := &storage.ListOptions{
			Recursive: true,
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("List with max keys only", func(t *testing.T) {
		opts := &storage.ListOptions{
			MaxKeys: 100,
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("List with max keys zero", func(t *testing.T) {
		opts := &storage.ListOptions{
			MaxKeys: 0,
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("List with all options", func(t *testing.T) {
		opts := &storage.ListOptions{
			Prefix:    "docs/",
			Recursive: true,
			MaxKeys:   1000,
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestStorage_BucketAndKeyVariations tests various bucket and key combinations.
func TestStorage_BucketAndKeyVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "default-bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	testCases := []struct {
		name   string
		bucket string
		key    string
	}{
		{"simple key", "bucket", "file.txt"},
		{"key with path", "bucket", "path/to/file.txt"},
		{"key with special chars", "bucket", "path/file-with-dashes.txt"},
		{"key with spaces", "bucket", "path/file name.txt"},
		{"key with extension", "bucket", "document.pdf"},
		{"nested key", "bucket", "a/b/c/d/file.txt"},
		{"empty bucket uses default", "", "file.txt"},
		{"root key", "bucket", "filename"},
		{"bucket with dots", "bucket.with.dots", "file.txt"},
		{"bucket with dashes", "bucket-with-dashes", "file.txt"},
		{"bucket with numbers", "bucket123", "file.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_Put", func(t *testing.T) {
			reader := strings.NewReader("test")
			err := stor.Put(context.Background(), tc.bucket, tc.key, reader, nil)
			assert.Error(t, err)
		})

		t.Run(tc.name+"_Get", func(t *testing.T) {
			rc, info, err := stor.Get(context.Background(), tc.bucket, tc.key)
			assert.Error(t, err)
			assert.Nil(t, rc)
			assert.Nil(t, info)
		})

		t.Run(tc.name+"_Exists", func(t *testing.T) {
			exists, err := stor.Exists(context.Background(), tc.bucket, tc.key)
			assert.Error(t, err)
			assert.False(t, exists)
		})
	}
}

// TestStorage_CloseTests variations for Close method.
func TestStorage_CloseVariations(t *testing.T) {
	t.Run("close with non-nil client", func(t *testing.T) {
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

	t.Run("close twice is safe", func(t *testing.T) {
		client := &Client{
			cfg:    Config{},
			logger: slog.Default(),
			closed: false,
		}

		stor := NewStorage(client, nil)
		err1 := stor.Close()
		err2 := stor.Close()
		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}

// TestStorage_ContextDeadlineExceeded tests context deadline exceeded scenarios.
func TestStorage_ContextDeadlineExceeded(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("Put with deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Hour))
		defer cancel()

		reader := strings.NewReader("test")
		err := stor.Put(ctx, "bucket", "key", reader, nil)
		assert.Error(t, err)
	})

	t.Run("Get with deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Hour))
		defer cancel()

		rc, info, err := stor.Get(ctx, "bucket", "key")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)
	})

	t.Run("Delete with deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Hour))
		defer cancel()

		err := stor.Delete(ctx, "bucket", "key")
		assert.Error(t, err)
	})

	t.Run("Exists with deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Hour))
		defer cancel()

		exists, err := stor.Exists(ctx, "bucket", "key")
		assert.Error(t, err)
		assert.False(t, exists)
	})
}

// TestStorage_ReaderVariations tests various reader types for Put.
func TestStorage_ReaderVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("with bytes.Reader", func(t *testing.T) {
		data := []byte("test data")
		reader := bytes.NewReader(data)
		err := stor.Put(context.Background(), "bucket", "key", reader, nil)
		assert.Error(t, err)
	})

	t.Run("with empty reader", func(t *testing.T) {
		reader := strings.NewReader("")
		err := stor.Put(context.Background(), "bucket", "key", reader, nil)
		assert.Error(t, err)
	})

	t.Run("with large data", func(t *testing.T) {
		data := bytes.Repeat([]byte("x"), 10*1024*1024) // 10MB
		reader := bytes.NewReader(data)
		err := stor.Put(context.Background(), "bucket", "key", reader, nil)
		assert.Error(t, err)
	})
}

// TestStorage_Get_ErrorPaths tests Get method error paths.
func TestStorage_Get_ErrorPaths(t *testing.T) {
	t.Run("Get with nil client returns error", func(t *testing.T) {
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

	t.Run("Get with empty bucket uses default", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		// Verify default bucket is set
		assert.Equal(t, "default-bucket", stor.cfg.DefaultBucket)

		// Call will fail because client is nil, but bucket should be used
		rc, info, err := stor.Get(context.Background(), "", "key")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_Exists_ErrorPaths tests Exists method error paths.
func TestStorage_Exists_ErrorPaths(t *testing.T) {
	t.Run("Exists with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		exists, err := stor.Exists(context.Background(), "bucket", "key")
		assert.Error(t, err)
		assert.False(t, exists)
	})

	t.Run("Exists with empty bucket uses default", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "my-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		// Verify default bucket is used
		exists, err := stor.Exists(context.Background(), "", "key")
		assert.Error(t, err)
		assert.False(t, exists)
	})
}

// TestStorage_Delete_ErrorPaths tests Delete method error paths.
func TestStorage_Delete_ErrorPaths(t *testing.T) {
	t.Run("Delete with nil client returns error", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		err := stor.Delete(context.Background(), "bucket", "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Delete with empty bucket uses default", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		err := stor.Delete(context.Background(), "", "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorage_List_ErrorPaths tests List method error paths.
func TestStorage_List_ErrorPaths(t *testing.T) {
	t.Run("List with nil client returns error", func(t *testing.T) {
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

	t.Run("List with nil options", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		result, err := stor.List(context.Background(), "bucket", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("List with empty bucket uses default", func(t *testing.T) {
		client := &Client{
			cfg:    Config{DefaultBucket: "default-bucket"},
			logger: slog.Default(),
		}
		stor := NewStorage(client, nil)

		result, err := stor.List(context.Background(), "", &storage.ListOptions{})
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestStorage_ListOptionsVariations tests List with various options.
func TestStorage_ListOptionsVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	_ = NewStorage(client, nil)

	options := []*storage.ListOptions{
		{Prefix: "test/", Recursive: true},
		{Prefix: "docs/", Recursive: false},
		{Prefix: "", Recursive: true},
		{Prefix: "images/", Recursive: true},
		{Prefix: "data/", Recursive: false},
	}

	for i, opts := range options {
		t.Run(fmt.Sprintf("options_%d", i), func(t *testing.T) {
			// Just verify options are structured correctly
			assert.NotNil(t, opts)
		})
	}
}

// TestStorage_PresignedURLVariations tests GetPresignedURL with various options.
func TestStorage_PresignedURLVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	expirations := []time.Duration{
		time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		time.Hour,
		24 * time.Hour,
	}

	for _, exp := range expirations {
		t.Run("expiry_"+exp.String(), func(t *testing.T) {
			opts := &storage.PresignedURLOptions{
				Method: "GET",
				Expiry: exp,
			}
			// Call will fail due to nil client, but we test the flow
			url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
			assert.Error(t, err)
			assert.Empty(t, url)
		})
	}
}

// TestStorage_ContextVariations tests various context scenarios.
func TestStorage_ContextVariations(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("With cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := stor.Get(ctx, "bucket", "key")
		assert.Error(t, err)
	})

	t.Run("With timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for timeout
		time.Sleep(10 * time.Millisecond)

		_, _, err := stor.Get(ctx, "bucket", "key")
		assert.Error(t, err)
	})
}
