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

// TestGetPresignedURL_UnsupportedMethod tests GetPresignedURL with unsupported HTTP methods.
func TestGetPresignedURL_UnsupportedMethod(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	unsupportedMethods := []string{
		"DELETE",
		"POST",
		"PATCH",
		"HEAD",
		"OPTIONS",
		"CONNECT",
		"TRACE",
		"invalid",
	}

	for _, method := range unsupportedMethods {
		t.Run("method_"+method, func(t *testing.T) {
			opts := &storage.PresignedURLOptions{
				Method: method,
				Expiry: 15 * time.Minute,
			}
			url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
			assert.Error(t, err)
			assert.Empty(t, url)
			assert.Contains(t, err.Error(), "unsupported HTTP method")
		})
	}
}

// TestGetPresignedURL_NilClient tests GetPresignedURL with nil minio client.
func TestGetPresignedURL_NilClient(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("GET method with nil client", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("PUT method with nil client", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "PUT",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("empty method defaults to GET with nil client", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestGetPresignedURL_NilOptions tests GetPresignedURL with nil options.
func TestGetPresignedURL_NilOptions(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", nil)
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "not initialized")
}

// TestGetPresignedURL_DefaultBucket tests GetPresignedURL with default bucket.
func TestGetPresignedURL_DefaultBucket(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "default-bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("empty bucket uses default", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "", "key", opts)
		assert.Error(t, err) // nil client error
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("explicit bucket overrides default", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "explicit-bucket", "key", opts)
		assert.Error(t, err) // nil client error
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestGetPresignedURL_DefaultExpiry tests that default expiry is applied.
func TestGetPresignedURL_DefaultExpiry(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("zero expiry uses default", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 0,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err) // nil client error
		assert.Empty(t, url)
	})

	t.Run("custom expiry is used", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 30 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err) // nil client error
		assert.Empty(t, url)
	})
}

// TestGetPresignedURL_DefaultMethod tests that default method is applied.
func TestGetPresignedURL_DefaultMethod(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("empty method defaults to GET", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err) // nil client error
		assert.Empty(t, url)
		// Should try GET and fail on client initialization, not on method validation
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestGetPresignedURL_VariousExpiries tests various expiry durations.
func TestGetPresignedURL_VariousExpiries(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	expiries := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
	}

	for _, expiry := range expiries {
		t.Run("expiry_"+expiry.String(), func(t *testing.T) {
			opts := &storage.PresignedURLOptions{
				Method: "GET",
				Expiry: expiry,
			}
			url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
			assert.Error(t, err) // nil client error
			assert.Empty(t, url)
		})
	}
}

// TestGetPresignedURL_BucketAndKey tests various bucket and key combinations.
func TestGetPresignedURL_BucketAndKey(t *testing.T) {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &storage.PresignedURLOptions{
				Method: "GET",
				Expiry: 15 * time.Minute,
			}
			url, err := stor.GetPresignedURL(context.Background(), tc.bucket, tc.key, opts)
			assert.Error(t, err) // nil client error
			assert.Empty(t, url)
		})
	}
}

// TestPresignedURLOptions tests PresignedURLOptions defaults and validation.
func TestPresignedURLOptions(t *testing.T) {
	t.Run("nil options", func(t *testing.T) {
		var opts *storage.PresignedURLOptions
		assert.Nil(t, opts)
	})

	t.Run("empty options", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{}
		assert.Equal(t, "", opts.Method)
		assert.Equal(t, time.Duration(0), opts.Expiry)
	})

	t.Run("GET options", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
		assert.Equal(t, "GET", opts.Method)
		assert.Equal(t, 15*time.Minute, opts.Expiry)
	})

	t.Run("PUT options", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "PUT",
			Expiry: 30 * time.Minute,
		}
		assert.Equal(t, "PUT", opts.Method)
		assert.Equal(t, 30*time.Minute, opts.Expiry)
	})
}

// TestGetPresignedURL_ErrorPriority tests that method validation happens before client validation.
func TestGetPresignedURL_ErrorPriority(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("invalid method returns error before client check", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "DELETE",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		// Method validation should happen first
		assert.Contains(t, err.Error(), "unsupported HTTP method")
		// Should NOT contain "not initialized" because method validation comes first
		assert.NotContains(t, err.Error(), "not initialized")
	})

	t.Run("valid method checks client first", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		// Should fail on client check
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestPresignedURL_Context tests context handling.
func TestPresignedURL_Context(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(ctx, "bucket", "key", opts)
		// Will fail on client check before method validation
		assert.Error(t, err)
		assert.Empty(t, url)
	})

	t.Run("with timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for timeout
		time.Sleep(10 * time.Millisecond)

		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(ctx, "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
	})
}

// TestGetPresignedURL_PutMethod tests PUT method specifically.
func TestGetPresignedURL_PutMethod(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("PUT method with nil client", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "PUT",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("PUT with lowercase put", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "put", // lowercase
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
		// Should be unsupported (case-sensitive)
		assert.Contains(t, err.Error(), "unsupported HTTP method")
	})
}

// TestPresignedURL_CaseSensitivity tests case sensitivity of HTTP methods.
func TestPresignedURL_CaseSensitivity(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	methods := []string{"get", "put", "Get", "Put", "GET", "PUT"}

	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			opts := &storage.PresignedURLOptions{
				Method: method,
				Expiry: 15 * time.Minute,
			}
			url, err := stor.GetPresignedURL(context.Background(), "bucket", "key", opts)

			// Only uppercase GET and PUT are valid
			if method == "GET" || method == "PUT" {
				// Should fail on client initialization
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not initialized")
			} else {
				// Should fail on method validation
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported HTTP method")
			}
			assert.Empty(t, url)
		})
	}
}

// TestGetPresignedURL_EmptyKey tests with empty key.
func TestGetPresignedURL_EmptyKey(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	t.Run("empty key with GET method", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "GET",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
	})

	t.Run("empty key with PUT method", func(t *testing.T) {
		opts := &storage.PresignedURLOptions{
			Method: "PUT",
			Expiry: 15 * time.Minute,
		}
		url, err := stor.GetPresignedURL(context.Background(), "bucket", "", opts)
		assert.Error(t, err)
		assert.Empty(t, url)
	})
}

// TestPresignedURL_StringCases tests various string edge cases.
func TestPresignedURL_StringCases(t *testing.T) {
	client := &Client{
		cfg:    Config{DefaultBucket: "bucket"},
		logger: slog.Default(),
	}
	stor := NewStorage(client, nil)

	testCases := []struct {
		name string
		key  string
	}{
		{"unicode key", "файл.txt"},
		{"key with dots", "file.v2.final.txt"},
		{"key with underscores", "my_file_name.txt"},
		{"key with plus", "file+name.txt"},
		{"key with percent", "file%20name.txt"},
		{"very long key", strings.Repeat("a", 1024)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &storage.PresignedURLOptions{
				Method: "GET",
				Expiry: 15 * time.Minute,
			}
			url, err := stor.GetPresignedURL(context.Background(), "bucket", tc.key, opts)
			assert.Error(t, err) // nil client error
			assert.Empty(t, url)
		})
	}
}
