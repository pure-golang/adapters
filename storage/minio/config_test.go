package minio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfig_GetEndpoint tests the GetEndpoint method.
func TestConfig_GetEndpoint(t *testing.T) {
	t.Run("with custom endpoint", func(t *testing.T) {
		cfg := Config{
			Endpoint: "localhost:9000",
		}
		assert.Equal(t, "localhost:9000", cfg.GetEndpoint())
	})

	t.Run("with default endpoint when empty", func(t *testing.T) {
		cfg := Config{
			Endpoint: "",
		}
		assert.Equal(t, DefaultYandexEndpoint, cfg.GetEndpoint())
	})

	t.Run("with Yandex endpoint explicitly set", func(t *testing.T) {
		cfg := Config{
			Endpoint: "storage.yandexcloud.net",
		}
		assert.Equal(t, "storage.yandexcloud.net", cfg.GetEndpoint())
	})

	t.Run("with AWS S3 endpoint", func(t *testing.T) {
		cfg := Config{
			Endpoint: "s3.amazonaws.com",
		}
		assert.Equal(t, "s3.amazonaws.com", cfg.GetEndpoint())
	})

	t.Run("zero config returns default", func(t *testing.T) {
		var cfg Config
		assert.Equal(t, DefaultYandexEndpoint, cfg.GetEndpoint())
	})
}

// TestConfig_DefaultYandexEndpoint tests the default endpoint constant.
func TestConfig_DefaultYandexEndpoint(t *testing.T) {
	assert.Equal(t, "storage.yandexcloud.net", DefaultYandexEndpoint)
}

// TestConfig_Fields tests all Config fields are properly settable.
func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		Endpoint:           "localhost:9000",
		AccessKey:          "test-access-key",
		SecretKey:          "test-secret-key",
		Region:             "eu-west-1",
		DefaultBucket:      "test-bucket",
		Secure:             false,
		Timeout:            60,
		InsecureSkipVerify: true,
	}

	assert.Equal(t, "localhost:9000", cfg.Endpoint)
	assert.Equal(t, "test-access-key", cfg.AccessKey)
	assert.Equal(t, "test-secret-key", cfg.SecretKey)
	assert.Equal(t, "eu-west-1", cfg.Region)
	assert.Equal(t, "test-bucket", cfg.DefaultBucket)
	assert.False(t, cfg.Secure)
	assert.Equal(t, 60, cfg.Timeout)
	assert.True(t, cfg.InsecureSkipVerify)
}

// TestConfig_DefaultValues tests default values for optional Config fields.
func TestConfig_DefaultValues(t *testing.T) {
	cfg := Config{
		AccessKey: "key",
		SecretKey: "secret",
	}

	// Endpoint defaults to Yandex via GetEndpoint
	assert.Equal(t, DefaultYandexEndpoint, cfg.GetEndpoint())
	// Other fields have zero defaults
	assert.Empty(t, cfg.Region)
	assert.Empty(t, cfg.DefaultBucket)
	assert.False(t, cfg.Secure)
	assert.Zero(t, cfg.Timeout)
	assert.False(t, cfg.InsecureSkipVerify)
}
