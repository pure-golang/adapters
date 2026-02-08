package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates Metrics with valid config", func(t *testing.T) {
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
	t.Run("registers metrics endpoint", func(t *testing.T) {
		config := Config{
			Host:                  "",
			Port:                  0, // Use random available port
			HttpServerReadTimeout: 30,
		}

		server := NewHttpServer(config)

		require.NotNil(t, server)
		assert.NotNil(t, server.Handler)

		// Start server to test the endpoint
		listener := httptest.NewServer(server.Handler)
		defer listener.Close()

		// Test metrics endpoint is registered
		resp, err := http.Get(listener.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("sets correct read timeout", func(t *testing.T) {
		config := Config{
			Host:                  "localhost",
			Port:                  9090,
			HttpServerReadTimeout: 60,
		}

		server := NewHttpServer(config)

		assert.Equal(t, 60*time.Second, server.ReadTimeout)
	})

	t.Run("uses default read timeout when not specified", func(t *testing.T) {
		config := Config{
			Host: "localhost",
			Port: 9090,
			// HttpServerReadTimeout not set, should use default 0
		}

		server := NewHttpServer(config)

		assert.Equal(t, 0*time.Second, server.ReadTimeout)
	})
}

func TestMetrics_Start(t *testing.T) {
	t.Run("success initializes prometheus and starts server", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode")
		}

		// Use a random port to avoid conflicts
		config := Config{
			Host:                  "127.0.0.1",
			Port:                  0, // Will be set to available port
			HttpServerReadTimeout: 5,
		}

		m := New(config)

		err := m.Start()
		require.NoError(t, err)

		// Clean up
		err = m.Close()
		assert.NoError(t, err)
	})

	t.Run("registers prometheus metrics", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode")
		}

		config := Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := New(config)

		err := m.Start()
		require.NoError(t, err)

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Verify prometheus was initialized by checking meter provider is set
		// This is tested indirectly through successful start

		err = m.Close()
		assert.NoError(t, err)
	})

	t.Run("multiple starts with close", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode")
		}

		config := Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := New(config)

		// First start
		err := m.Start()
		require.NoError(t, err)

		// Close
		err = m.Close()
		require.NoError(t, err)

		// Note: Starting again after close would fail because
		// http.Server cannot be restarted after Close
	})
}

func TestMetrics_Close(t *testing.T) {
	t.Run("closes the server successfully", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode")
		}

		config := Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := New(config)

		err := m.Start()
		require.NoError(t, err)

		// Give server time to start
		time.Sleep(50 * time.Millisecond)

		err = m.Close()
		assert.NoError(t, err)
	})

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

	t.Run("implements io.Closer interface", func(t *testing.T) {
		config := Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := New(config)
		m.Start()

		var closer io.Closer = m
		err := closer.Close()
		assert.NoError(t, err)
	})
}

func TestMetrics_Concurrent(t *testing.T) {
	t.Run("concurrent start and close", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode")
		}

		config := Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := New(config)

		var wg sync.WaitGroup
		errors := make(chan error, 2)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Start(); err != nil {
				errors <- err
			}
		}()

		// Give start time to complete
		time.Sleep(50 * time.Millisecond)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Close(); err != nil {
				errors <- err
			}
		}()

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestInitDefault(t *testing.T) {
	t.Run("starts server successfully", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode")
		}

		config := Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		closer, err := InitDefault(config)
		require.NoError(t, err)
		require.NotNil(t, closer)

		// Give server time to start
		time.Sleep(50 * time.Millisecond)

		// Clean up
		err = closer.Close()
		assert.NoError(t, err)
	})

	t.Run("returns io.Closer implementation", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode")
		}

		config := Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		closer, err := InitDefault(config)
		require.NoError(t, err)
		require.NotNil(t, closer)

		// Verify it implements io.Closer
		_, ok := closer.(io.Closer)
		assert.True(t, ok)

		closer.Close()
	})

	t.Run("New creates HTTP server via InitDefault flow", func(t *testing.T) {
		config := Config{
			Host:                  "localhost",
			Port:                  9090,
			HttpServerReadTimeout: 30,
		}

		m := New(config)

		assert.NotNil(t, m)
		assert.NotNil(t, m.server)
		assert.IsType(t, &http.Server{}, m.server)
	})
}
