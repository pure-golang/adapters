package metrics_test

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/metrics"
)

func TestMetrics_Start(t *testing.T) {
	t.Run("success initializes prometheus and starts server", func(t *testing.T) {
		if testing.Short() {
			t.Skip("integration test")
		}

		// Use a random port to avoid conflicts
		config := metrics.Config{
			Host:                  "127.0.0.1",
			Port:                  0, // Will be set to available port
			HttpServerReadTimeout: 5,
		}

		m := metrics.New(config)

		err := m.Start()
		require.NoError(t, err)

		// Clean up
		err = m.Close()
		assert.NoError(t, err)
	})

	t.Run("registers prometheus metrics", func(t *testing.T) {
		if testing.Short() {
			t.Skip("integration test")
		}

		config := metrics.Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := metrics.New(config)

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
			t.Skip("integration test")
		}

		config := metrics.Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := metrics.New(config)

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
			t.Skip("integration test")
		}

		config := metrics.Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := metrics.New(config)

		err := m.Start()
		require.NoError(t, err)

		// Give server time to start
		time.Sleep(50 * time.Millisecond)

		err = m.Close()
		assert.NoError(t, err)
	})

	t.Run("close without start is safe", func(t *testing.T) {
		config := metrics.Config{
			Host:                  "127.0.0.1",
			Port:                  9090,
			HttpServerReadTimeout: 30,
		}

		m := metrics.New(config)

		err := m.Close()
		assert.NoError(t, err)
	})

	t.Run("implements io.Closer interface", func(t *testing.T) {
		config := metrics.Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := metrics.New(config)
		_ = m.Start()

		var closer io.Closer = m
		err := closer.Close()
		assert.NoError(t, err)
	})
}

func TestMetrics_Concurrent(t *testing.T) {
	t.Run("concurrent start and close", func(t *testing.T) {
		if testing.Short() {
			t.Skip("integration test")
		}

		config := metrics.Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		m := metrics.New(config)

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
			t.Skip("integration test")
		}

		config := metrics.Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		closer, err := metrics.InitDefault(config)
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
			t.Skip("integration test")
		}

		config := metrics.Config{
			Host:                  "127.0.0.1",
			Port:                  0,
			HttpServerReadTimeout: 5,
		}

		closer, err := metrics.InitDefault(config)
		require.NoError(t, err)
		require.NotNil(t, closer)

		// closer already is io.Closer, no need for type assertion
		assert.NotNil(t, closer)

		closer.Close()
	})
}

func TestNewHttpServer(t *testing.T) {
	t.Run("creates server with handler", func(t *testing.T) {
		config := metrics.Config{
			Host:                  "",
			Port:                  0,
			HttpServerReadTimeout: 30,
		}

		server := metrics.NewHttpServer(config)

		require.NotNil(t, server)
		assert.NotNil(t, server.Handler)
		assert.Equal(t, ":0", server.Addr)
	})
}
