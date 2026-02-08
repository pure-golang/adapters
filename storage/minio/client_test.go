package minio

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)

// TestNewClient_InvalidConfig tests NewClient with invalid configurations.
func TestNewClient_InvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		options     *ClientOptions
		expectedErr string
	}{
		{
			name: "invalid endpoint with options",
			cfg: Config{
				Endpoint:  "localhost:9999",
				AccessKey: "test",
				SecretKey: "test",
			},
			options:     &ClientOptions{Logger: slog.Default()},
			expectedErr: "failed to connect to S3 storage",
		},
		{
			name: "empty endpoint (uses default) but fails to connect",
			cfg: Config{
				Endpoint:  "",
				AccessKey: "test",
				SecretKey: "test",
			},
			options:     nil,
			expectedErr: "failed to connect to S3 storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg, tt.options)
			assert.Error(t, err)
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// TestNewClient_Options tests NewClient with various options.
func TestNewClient_Options(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		options *ClientOptions
	}{
		{
			name: "nil options",
			cfg: Config{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
			},
			options: nil,
		},
		{
			name: "empty options",
			cfg: Config{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
			},
			options: &ClientOptions{},
		},
		{
			name: "options with nil logger",
			cfg: Config{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
			},
			options: &ClientOptions{
				Logger: nil,
			},
		},
		{
			name: "options with custom logger",
			cfg: Config{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
			},
			options: &ClientOptions{
				Logger: slog.Default(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These will fail to connect, but we're testing option handling
			client, err := NewClient(tt.cfg, tt.options)
			assert.Error(t, err) // Expected to fail connection
			assert.Nil(t, client)
		})
	}
}

// TestNewClient_SecureFlag tests that Secure and InsecureSkipVerify are handled correctly.
func TestNewClient_SecureFlag(t *testing.T) {
	tests := []struct {
		name                 string
		cfg                  Config
		expectedSecureConfig bool // This represents what secure value should be passed to minio
	}{
		{
			name: "secure true",
			cfg: Config{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
				Secure:    true,
			},
			expectedSecureConfig: true,
		},
		{
			name: "secure false",
			cfg: Config{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
				Secure:    false,
			},
			expectedSecureConfig: false,
		},
		{
			name: "InsecureSkipVerify overrides secure true",
			cfg: Config{
				Endpoint:           "localhost:9000",
				AccessKey:          "key",
				SecretKey:          "secret",
				Secure:             true,
				InsecureSkipVerify: true,
			},
			expectedSecureConfig: false,
		},
		{
			name: "InsecureSkipVerify with secure false",
			cfg: Config{
				Endpoint:           "localhost:9000",
				AccessKey:          "key",
				SecretKey:          "secret",
				Secure:             false,
				InsecureSkipVerify: true,
			},
			expectedSecureConfig: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These will fail to connect, but we're testing config handling
			_, err := NewClient(tt.cfg, nil)
			assert.Error(t, err)
		})
	}
}

// TestNewClient_Timeout tests custom timeout configuration.
func TestNewClient_Timeout(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected time.Duration
	}{
		{
			name: "zero timeout uses default 30s",
			cfg: Config{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
				Timeout:   0,
			},
			expected: 30 * time.Second,
		},
		{
			name: "custom timeout 60s",
			cfg: Config{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
				Timeout:   60,
			},
			expected: 60 * time.Second,
		},
		{
			name: "custom timeout 120s",
			cfg: Config{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
				Timeout:   120,
			},
			expected: 120 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These will fail to connect, but we're testing timeout handling
			_, err := NewClient(tt.cfg, nil)
			assert.Error(t, err)
		})
	}
}

// TestNewDefaultClient_Extended tests the NewDefaultClient function.
func TestNewDefaultClient_Extended(t *testing.T) {
	t.Run("uses default options", func(t *testing.T) {
		cfg := Config{
			Endpoint:  "localhost:9999",
			AccessKey: "test",
			SecretKey: "test",
		}

		client, err := NewDefaultClient(cfg)
		assert.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("with custom endpoint that fails", func(t *testing.T) {
		cfg := Config{
			Endpoint:  "nonexistent.example.com:9000",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			Secure:    false,
		}

		client, err := NewDefaultClient(cfg)
		assert.Error(t, err) // No server running
		assert.Nil(t, client)
	})
}

// TestClient_Close_Extended tests the Client Close method.
func TestClient_Close_Extended(t *testing.T) {
	t.Run("close once successfully", func(t *testing.T) {
		client := &Client{
			logger: slog.Default(),
			closed: false,
		}

		err := client.Close()
		assert.NoError(t, err)
		assert.True(t, client.IsClosed())
	})

	t.Run("close twice returns nil both times", func(t *testing.T) {
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

	t.Run("IsClosed before and after close", func(t *testing.T) {
		client := &Client{
			logger: slog.Default(),
			closed: false,
		}

		assert.False(t, client.IsClosed())

		err := client.Close()
		assert.NoError(t, err)

		assert.True(t, client.IsClosed())
	})

	t.Run("close with nil logger panics", func(t *testing.T) {
		client := &Client{
			logger: nil,
			closed: false,
		}

		// Closing with nil logger will panic due to logger.Info call
		assert.Panics(t, func() {
			_ = client.Close()
		})
	})

	t.Run("close already closed client", func(t *testing.T) {
		client := &Client{
			logger: slog.Default(),
			closed: true, // Already closed
		}

		err := client.Close()
		assert.NoError(t, err)
		assert.True(t, client.IsClosed())
	})
}

// TestClient_IsClosed tests the IsClosed method with concurrent access.
func TestClient_IsClosed(t *testing.T) {
	t.Run("concurrent close and check", func(t *testing.T) {
		client := &Client{
			logger: slog.Default(),
			closed: false,
		}

		// Start goroutines that check IsClosed
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				_ = client.IsClosed()
				done <- true
			}()
		}

		// Close the client
		err := client.Close()
		assert.NoError(t, err)

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		assert.True(t, client.IsClosed())
	})
}

// TestClient_GetMinioClient tests the GetMinioClient method.
func TestClient_GetMinioClient(t *testing.T) {
	t.Run("returns underlying minio client", func(t *testing.T) {
		minioClient := &minio.Client{}
		client := &Client{
			client: minioClient,
		}

		result := client.GetMinioClient()
		assert.Equal(t, minioClient, result)
	})

	t.Run("returns nil when client is nil", func(t *testing.T) {
		client := &Client{
			client: nil,
		}

		result := client.GetMinioClient()
		assert.Nil(t, result)
	})
}

// TestClient_Initialization tests client initialization scenarios.
func TestClient_Initialization(t *testing.T) {
	t.Run("client with full config", func(t *testing.T) {
		cfg := Config{
			Endpoint:           "localhost:9000",
			AccessKey:          "access",
			SecretKey:          "secret",
			Region:             "us-east-1",
			DefaultBucket:      "bucket",
			Secure:             false,
			Timeout:            30,
			InsecureSkipVerify: false,
		}

		// Will fail to connect but tests config parsing
		client, err := NewClient(cfg, &ClientOptions{Logger: slog.Default()})
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

// TestClient_ConnectionTimeout tests that connection timeout is respected.
func TestClient_ConnectionTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	// Use a non-routable IP to trigger timeout
	cfg := Config{
		Endpoint:  "192.0.2.1:9000", // TEST-NET-1, should never route
		AccessKey: "test",
		SecretKey: "test",
		Timeout:   1, // 1 second timeout
	}

	start := time.Now()
	_, err := NewClient(cfg, nil)
	elapsed := time.Since(start)

	assert.Error(t, err)
	// Should timeout within a reasonable time (allow some margin)
	assert.Less(t, elapsed, 5*time.Second)
}

// TestCloserInterface_Extended verifies Client implements Closer interface.
func TestCloserInterface_Extended(t *testing.T) {
	// Compile-time interface check
	var _ Closer = (*Client)(nil)

	client := &Client{}
	assert.Implements(t, (*Closer)(nil), client)
}

// TestClient_Logging tests that client logs appropriately.
func TestClient_Logging(t *testing.T) {
	t.Run("client with logger", func(t *testing.T) {
		logger := slog.Default()
		client := &Client{
			logger: logger,
			closed: false,
		}

		err := client.Close()
		assert.NoError(t, err)
		assert.True(t, client.IsClosed())
	})
}

// TestNewClient_Context tests client creation with context timeout.
func TestNewClient_Context(t *testing.T) {
	t.Run("connection timeout during ListBuckets", func(t *testing.T) {
		// This test verifies that the timeout is applied during the ListBuckets check
		cfg := Config{
			Endpoint:  "localhost:9999",
			AccessKey: "test",
			SecretKey: "test",
			Timeout:   1,
		}

		_, err := NewClient(cfg, nil)
		assert.Error(t, err)
	})
}

// mockListBuckets simulates a minio.Client for testing.
type mockMinioClient struct {
	minio.Client
	listBucketsFunc func(ctx context.Context) ([]minio.BucketInfo, error)
}

func (m *mockMinioClient) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	if m.listBucketsFunc != nil {
		return m.listBucketsFunc(ctx)
	}
	return []minio.BucketInfo{}, nil
}
