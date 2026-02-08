package minio

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
)

var _ Closer = (*Client)(nil)

// Closer is the interface for closing resources.
type Closer interface {
	Close() error
}

// Client wraps minio.Client for S3-compatible storage operations.
// Supports MinIO, Yandex Cloud Storage, AWS S3, and other S3-compatible providers.
type Client struct {
	client *minio.Client
	cfg    Config
	logger *slog.Logger
	mu     sync.RWMutex
	closed bool
}

// ClientOptions contains options for client creation.
type ClientOptions struct {
	Logger *slog.Logger
}

// NewClient creates a new S3-compatible storage client.
func NewClient(cfg Config, options *ClientOptions) (*Client, error) {
	if options == nil {
		options = &ClientOptions{}
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}

	logger := options.Logger.WithGroup("s3")

	// Initialize minio client with static credentials
	creds := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")

	endpoint := cfg.GetEndpoint()

	// Determine secure setting: InsecureSkipVerify takes precedence
	secure := cfg.Secure
	if cfg.InsecureSkipVerify {
		secure = false
	}

	opts := &minio.Options{
		Creds:  creds,
		Region: cfg.Region,
		Secure: secure,
	}

	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create S3 client")
	}

	// Verify connection by listing buckets
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err = client.ListBuckets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to S3 storage")
	}

	logger.Info("S3 client initialized", "endpoint", endpoint, "region", cfg.Region)

	return &Client{
		client: client,
		cfg:    cfg,
		logger: logger,
	}, nil
}

// NewDefaultClient creates a client with default options.
func NewDefaultClient(cfg Config) (*Client, error) {
	return NewClient(cfg, nil)
}

// GetMinioClient returns the underlying minio.Client.
func (c *Client) GetMinioClient() *minio.Client {
	return c.client
}

// Close closes the S3 client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.logger.Info("S3 client closed")
	return nil
}

// IsClosed returns true if the client is closed.
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}
