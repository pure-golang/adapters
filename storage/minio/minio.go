package minio

const (
	// DefaultYandexEndpoint is the default Yandex Cloud Storage endpoint.
	DefaultYandexEndpoint = "storage.yandexcloud.net"
)

// Config contains S3-compatible storage connection configuration.
// Works with MinIO, Yandex Cloud Storage, AWS S3, and other S3-compatible providers.
type Config struct {
	Endpoint           string `envconfig:"S3_ENDPOINT"`                             // S3 endpoint (e.g., "localhost:9000" for MinIO, "storage.yandexcloud.net" for Yandex)
	AccessKey          string `envconfig:"S3_ACCESS_KEY" required:"true"`           // Access key ID
	SecretKey          string `envconfig:"S3_SECRET_KEY" required:"true"`           // Secret access key
	Region             string `envconfig:"S3_REGION" default:"us-east-1"`           // Region name
	DefaultBucket      string `envconfig:"S3_BUCKET"`                               // Default bucket name
	Secure             bool   `envconfig:"S3_SECURE" default:"true"`                // Use HTTPS (default true for cloud providers)
	Timeout            int    `envconfig:"S3_TIMEOUT" default:"30"`                 // Connection timeout in seconds
	InsecureSkipVerify bool   `envconfig:"S3_INSECURE_SKIP_VERIFY" default:"false"` // Skip TLS verification (for self-signed certs)
}

// GetEndpoint returns the endpoint to use, defaulting to Yandex Cloud if not set.
// For local MinIO, you should explicitly set Endpoint to "localhost:9000".
func (c *Config) GetEndpoint() string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	return DefaultYandexEndpoint
}
