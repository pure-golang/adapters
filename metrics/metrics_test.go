package metrics

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("creates Metrics with valid config", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Host:                  "localhost",
			Port:                  9090,
			HttpServerReadTimeout: 30,
		}

		m := New(config)

		assert.NotNil(t, m)
		assert.Equal(t, config, m.config)
		assert.NotNil(t, m.server)
	})

	t.Run("creates HTTP server with correct address", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Host:                  "127.0.0.1",
			Port:                  8080,
			HttpServerReadTimeout: 15,
		}

		m := New(config)

		assert.NotNil(t, m.server)
		assert.Equal(t, "127.0.0.1:8080", m.server.Addr)
		assert.Equal(t, 15*time.Second, m.server.ReadTimeout)
	})
}

func TestNewHttpServer(t *testing.T) {
	t.Run("sets correct read timeout", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Host:                  "localhost",
			Port:                  9090,
			HttpServerReadTimeout: 60,
		}

		server := NewHttpServer(config)

		assert.Equal(t, 60*time.Second, server.ReadTimeout)
	})

	t.Run("uses default read timeout when not specified", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Host: "localhost",
			Port: 9090,
			// HttpServerReadTimeout not set, should use default 0
		}

		server := NewHttpServer(config)

		assert.Equal(t, 0*time.Second, server.ReadTimeout)
	})
}

func TestMetrics_Close_WithoutStart(t *testing.T) {
	t.Run("close without start is safe", func(t *testing.T) {
		config := Config{
			Host:                  "127.0.0.1",
			Port:                  9090,
			HttpServerReadTimeout: 30,
		}

		m := New(config)

		err := m.Close()
		assert.NoError(t, err)
	})
}

func TestNew_CreatesHTTPServer(t *testing.T) {
	config := Config{
		Host:                  "localhost",
		Port:                  9090,
		HttpServerReadTimeout: 30,
	}

	m := New(config)

	assert.NotNil(t, m)
	assert.NotNil(t, m.server)
	assert.IsType(t, &http.Server{}, m.server) //nolint:gosec
}

func TestMetrics_ImplementsCloser(t *testing.T) {
	config := Config{
		Host:                  "127.0.0.1",
		Port:                  9090,
		HttpServerReadTimeout: 30,
	}

	m := New(config)

	var _ io.Closer = m
}
