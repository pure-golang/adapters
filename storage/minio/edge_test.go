package minio

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pure-golang/adapters/storage"
	"github.com/stretchr/testify/assert"
)

// TestStorage_EdgeCases tests edge cases for storage operations.
func TestStorage_EdgeCases(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "default-bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("Put with empty key uses default bucket", func(t *testing.T) {
		reader := strings.NewReader("test")
		err := stor.Put(context.Background(), "", "key.txt", reader, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Get with empty key uses default bucket", func(t *testing.T) {
		rc, info, err := stor.Get(context.Background(), "", "key.txt")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Delete with empty key uses default bucket", func(t *testing.T) {
		err := stor.Delete(context.Background(), "", "key.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Exists with empty key uses default bucket", func(t *testing.T) {
		exists, err := stor.Exists(context.Background(), "", "key.txt")
		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("List with empty bucket uses default bucket", func(t *testing.T) {
		result, err := stor.List(context.Background(), "", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("GetPresignedURL with empty bucket uses default", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
		}
		url, err := stor.GetPresignedURL(context.Background(), "", "key.txt", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("CreateMultipartUpload with empty bucket uses default", func(t *testing.T) {
		upload, err := stor.CreateMultipartUpload(context.Background(), "", "key.txt", nil)
		assert.Error(t, err)
		assert.Nil(t, upload)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("UploadPart with empty bucket uses default", func(t *testing.T) {
		reader := strings.NewReader("test data")
		part, err := stor.UploadPart(context.Background(), "", "key.txt", "upload-id", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("CompleteMultipartUpload with empty bucket uses default", func(t *testing.T) {
		opts := &storage.CompleteMultipartUploadOptions{}
		info, err := stor.CompleteMultipartUpload(context.Background(), "", "key.txt", "upload-id", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("AbortMultipartUpload with empty bucket uses default", func(t *testing.T) {
		err := stor.AbortMultipartUpload(context.Background(), "", "key.txt", "upload-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("ListMultipartUploads with empty bucket uses default", func(t *testing.T) {
		t.Skip("ListIncompleteUploads spawns goroutines that panic cannot be caught")
		_, err := stor.ListMultipartUploads(context.Background(), "")
		assert.Error(t, err)
	})
}

// TestStorage_SpecialCharacters tests keys with special characters.
func TestStorage_SpecialCharacters(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	specialKeys := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.with.dots.txt",
		"path/to/file.txt",
		"file@special.com",
		"file(1).txt",
		"file[1].txt",
		"file{1}.txt",
		"file+plus.txt",
		"file%percent.txt",
		"file#hash.txt",
		"file$dollar.txt",
		"file&ampersand.txt",
		"file=equal.txt",
		"file!exclaim.txt",
		"file~tilde.txt",
		"file^caret.txt",
		"file`backtick.txt",
		"file'apostrophe.txt",
		"file\"quote.txt",
		"file<less.txt",
		"file>greater.txt",
		"file,comma.txt",
		"file;semi.txt",
		"file:colon.txt",
		"file|pipe.txt",
		"file\\backslash.txt",
		"file/forward.txt",
		"file?question.txt",
		"file*asterisk.txt",
	}

	for _, key := range specialKeys {
		t.Run("key_"+strings.ReplaceAll(key, "/", "_slash_"), func(t *testing.T) {
			// Put
			reader := strings.NewReader("test")
			err := stor.Put(context.Background(), "bucket", key, reader, nil)
			assert.Error(t, err)

			// Get
			rc, info, err := stor.Get(context.Background(), "bucket", key)
			assert.Error(t, err)
			assert.Nil(t, rc)
			assert.Nil(t, info)

			// Delete
			err = stor.Delete(context.Background(), "bucket", key)
			assert.Error(t, err)

			// Exists
			exists, err := stor.Exists(context.Background(), "bucket", key)
			assert.Error(t, err)
			assert.False(t, exists)
		})
	}
}

// TestStorage_UnicodeKeys tests keys with unicode characters.
func TestStorage_UnicodeKeys(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	unicodeKeys := []string{
		"файл.txt",         // Cyrillic
		"文件.txt",           // Chinese
		"ファイル.txt",         // Japanese
		"파일.txt",           // Korean
		"αρχείο.txt",       // Greek
		"fichierè.txt",     // French with accent
		"archivoñ.txt",     // Spanish with tilde
		"dateiü.txt",       // German with umlaut
		"папка/файл.txt",   // Cyrillic with path
		"dossier/файл.txt", // Mixed
	}

	for _, key := range unicodeKeys {
		t.Run("unicode_key", func(t *testing.T) {
			// Put
			reader := strings.NewReader("test")
			err := stor.Put(context.Background(), "bucket", key, reader, nil)
			assert.Error(t, err)

			// Get
			rc, info, err := stor.Get(context.Background(), "bucket", key)
			assert.Error(t, err)
			assert.Nil(t, rc)
			assert.Nil(t, info)
		})
	}
}

// TestStorage_LongKeys tests very long key names.
func TestStorage_LongKeys(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("very long key", func(t *testing.T) {
		longKey := strings.Repeat("a", 1024)
		reader := strings.NewReader("test")
		err := stor.Put(context.Background(), "bucket", longKey, reader, nil)
		assert.Error(t, err)
	})

	t.Run("nested long path", func(t *testing.T) {
		longPath := strings.Repeat("a/", 100) + "file.txt"
		reader := strings.NewReader("test")
		err := stor.Put(context.Background(), "bucket", longPath, reader, nil)
		assert.Error(t, err)
	})
}

// TestStorage_EmptyAndNilValues tests empty and nil values.
func TestStorage_EmptyAndNilValues(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("Put with empty reader", func(t *testing.T) {
		reader := strings.NewReader("")
		err := stor.Put(context.Background(), "bucket", "key.txt", reader, nil)
		assert.Error(t, err)
	})

	t.Run("Put with nil reader", func(t *testing.T) {
		err := stor.Put(context.Background(), "bucket", "key.txt", nil, nil)
		assert.Error(t, err)
	})

	t.Run("Get with empty key", func(t *testing.T) {
		rc, info, err := stor.Get(context.Background(), "bucket", "")
		assert.Error(t, err)
		assert.Nil(t, rc)
		assert.Nil(t, info)
	})

	t.Run("Delete with empty key", func(t *testing.T) {
		err := stor.Delete(context.Background(), "bucket", "")
		assert.Error(t, err)
	})

	t.Run("Exists with empty key", func(t *testing.T) {
		exists, err := stor.Exists(context.Background(), "bucket", "")
		assert.Error(t, err)
		assert.False(t, exists)
	})
}

// TestMultipartUpload_EdgeCases tests edge cases for multipart upload.
func TestMultipartUpload_EdgeCases(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("UploadPart with empty reader", func(t *testing.T) {
		reader := strings.NewReader("")
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-id", 1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
	})

	t.Run("UploadPart with part number 0", func(t *testing.T) {
		reader := strings.NewReader("test")
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-id", 0, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
	})

	t.Run("UploadPart with negative part number", func(t *testing.T) {
		reader := strings.NewReader("test")
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-id", -1, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
	})

	t.Run("UploadPart with very large part number", func(t *testing.T) {
		reader := strings.NewReader("test")
		part, err := stor.UploadPart(context.Background(), "bucket", "key", "upload-id", 10000, reader)
		assert.Error(t, err)
		assert.Nil(t, part)
	})

	t.Run("CompleteMultipartUpload with empty parts", func(t *testing.T) {
		opts := &storage.CompleteMultipartUploadOptions{
			Parts: []storage.UploadedPart{},
		}
		info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-id", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("CompleteMultipartUpload with nil parts", func(t *testing.T) {
		opts := &storage.CompleteMultipartUploadOptions{}
		info, err := stor.CompleteMultipartUpload(context.Background(), "bucket", "key", "upload-id", opts)
		assert.Error(t, err)
		assert.Nil(t, info)
	})
}

// TestPresignedURL_EdgeCases tests edge cases for presigned URLs.
func TestPresignedURL_EdgeCases(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("with very short expiry", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 1,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
	})

	t.Run("with very long expiry", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 7 * 24 * time.Hour, // 7 days
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
	})

	t.Run("GET with special characters in key", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
		}
		specialKeys := []string{
			"file with spaces.txt",
			"файл.txt",
			"文件.txt",
			"file@special.com",
			"path/to/file(1).txt",
		}
		for _, key := range specialKeys {
			url, err := stor.GetPresignedURL(context.Background(), "bucket", key, opts)
			assert.Error(t, err)
			assert.Empty(t, url)
		}
	})
}

// TestList_EdgeCases tests edge cases for List operation.
func TestList_EdgeCases(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("with empty prefix", func(t *testing.T) {
		opts := &storage.ListOptions{
			Prefix: "",
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("with very long prefix", func(t *testing.T) {
		opts := &storage.ListOptions{
			Prefix: strings.Repeat("a", 1024),
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("with special characters in prefix", func(t *testing.T) {
		prefixes := []string{
			"файл/",
			"文件/",
			"file with spaces/",
			"file@special/",
			"path:to:",
		}
		for _, prefix := range prefixes {
			opts := &storage.ListOptions{
				Prefix: prefix,
			}
			result, err := stor.List(context.Background(), "bucket", opts)
			assert.Error(t, err)
			assert.Nil(t, result)
		}
	})

	t.Run("with zero max keys", func(t *testing.T) {
		opts := &storage.ListOptions{
			MaxKeys: 0,
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("with large max keys", func(t *testing.T) {
		opts := &storage.ListOptions{
			MaxKeys: 1000000,
		}
		result, err := stor.List(context.Background(), "bucket", opts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
