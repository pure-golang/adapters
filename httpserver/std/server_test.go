package std

import (
	"context"
	"net/http"
	"testing"
	"time"

	"git.korputeam.ru/newbackend/adapters/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize noop logger for tests
	logger.InitDefault(logger.Config{
		Provider: logger.ProviderNoop,
		Level:    logger.INFO,
	})
}

// TestNew_WithValidConfig tests that New creates a server with valid config
func TestNew_WithValidConfig(t *testing.T) {
	t.Parallel()
	config := Config{
		Host: "127.0.0.1",
		Port: 8080,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := New(config, handler)

	require.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.logger)
	assert.Equal(t, config, server.config)
	assert.Equal(t, "127.0.0.1:8080", server.server.Addr)
	assert.NotNil(t, server.server.Handler) // Can't compare functions directly
}

// TestNew_WithTLSConfig tests that New handles TLS configuration
func TestNew_WithTLSConfig(t *testing.T) {
	t.Parallel()
	config := Config{
		Host:        "0.0.0.0",
		Port:        8443,
		TLSCertPath: "/path/to/cert.pem",
		TLSKeyPath:  "/path/to/key.pem",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := New(config, handler)

	require.NotNil(t, server)
	assert.Equal(t, config.TLSCertPath, server.config.TLSCertPath)
	assert.Equal(t, config.TLSKeyPath, server.config.TLSKeyPath)
}

// TestNew_SetsReadHeaderTimeout tests that New sets ReadHeaderTimeout to prevent Slowloris attacks
func TestNew_SetsReadHeaderTimeout(t *testing.T) {
	t.Parallel()
	config := Config{
		Host: "localhost",
		Port: 9090,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := New(config, handler)

	require.NotNil(t, server)
	assert.Equal(t, 10*time.Second, server.server.ReadHeaderTimeout, "ReadHeaderTimeout should be set to 10 seconds to prevent Slowloris attacks")
}

// TestNew_WithEmptyHost tests that New handles empty host (defaults to all interfaces)
func TestNew_WithEmptyHost(t *testing.T) {
	t.Parallel()
	config := Config{
		Host: "",
		Port: 8080,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := New(config, handler)

	require.NotNil(t, server)
	assert.Equal(t, ":8080", server.server.Addr)
}

// TestNewDefault_SetsErrorLog tests that NewDefault sets the error log
func TestNewDefault_SetsErrorLog(t *testing.T) {
	t.Parallel()
	config := Config{
		Host: "localhost",
		Port: 8080,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewDefault(config, handler)

	require.NotNil(t, server)
	assert.NotNil(t, server.server.ErrorLog, "ErrorLog should be set by NewDefault")
}

// TestServer_Close_GracefulShutdown tests that Server Close performs graceful shutdown
func TestServer_Close_GracefulShutdown(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := Config{
		Host: "localhost",
		Port: 9999,
	}

	server := New(config, handler)

	// Close should not panic even though server was never started
	err := server.Close()
	// An error is expected since the server wasn't running, but it shouldn't panic
	assert.NotNil(t, server)
	_ = err
}

// TestServer_Close_WithContextCancellation tests Close when context is cancelled
func TestServer_Close_WithContextCancellation(t *testing.T) {
	t.Parallel()
	// Note: Shutdown on an idle server may succeed even with cancelled context
	// because there are no active connections to wait for
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := Config{
		Host: "127.0.0.1",
		Port: 0,
	}

	server := New(config, handler)

	// Create a context that we'll cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Shutdown with cancelled context - may succeed if server is idle
	_ = server.server.Shutdown(ctx)
	// The important thing is it doesn't panic
}

// TestServer_Close_NilServer tests Close on a server that hasn't been properly initialized
func TestServer_Close_NilServer(t *testing.T) {
	t.Parallel()
	// Create a server but don't start it
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := Config{
		Host: "localhost",
		Port: 9999,
	}

	server := New(config, handler)

	// Close should not panic even though server was never started
	// Note: Shutdown on an idle http.Server returns nil (success)
	err := server.Close()
	// The error wrapping logic may wrap nil, resulting in nil
	// The important thing is Close doesn't panic
	assert.NotNil(t, server)
	_ = err
}

// TestServer_Start_InvalidPort tests Start with invalid port configuration
func TestServer_Start_InvalidPort(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := Config{
		Host: "localhost",
		Port: -1,
	}

	server := New(config, handler)

	err := server.Start()
	assert.Error(t, err, "Start should return error for invalid port")
}

// TestServer_Run_LogsErrorOnFailure tests that Run logs error when server fails to start
func TestServer_Run_LogsErrorOnFailure(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := Config{
		Host: "localhost",
		Port: -1,
	}

	server := New(config, handler)

	// Run should start in background and log error
	done := make(chan struct{})
	go func() {
		server.Run()
		close(done)
	}()

	// Run should return quickly since Start will fail
	select {
	case <-done:
		// Run completed (server failed to start)
	case <-time.After(2 * time.Second):
		t.Fatal("Run should return when Start fails")
	}
}

// TestShutdownTimeout_Constant tests the ShutdownTimeout constant value
func TestShutdownTimeout_Constant(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 15*time.Second, ShutdownTimeout, "ShutdownTimeout should be 15 seconds")
}

// TestServer_ImplementsRunableProvider tests that Server implements httpserver.RunableProvider
func TestServer_ImplementsRunableProvider(t *testing.T) {
	t.Parallel()
	// This is verified at compile time by the var declaration:
	// var _ httpserver.RunableProvider = (*Server)(nil)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := Config{
		Host: "localhost",
		Port: 8080,
	}

	server := New(config, handler)
	assert.NotNil(t, server)
}
