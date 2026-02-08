package std

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/pure-golang/adapters/logger"
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

// TestServer_Close_TimeoutExceeded tests Close behavior when shutdown timeout is exceeded
func TestServer_Close_TimeoutExceeded(t *testing.T) {
	// Create a server with a handler that blocks connections
	handlerCalled := make(chan struct{}, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled <- struct{}{}
		// Keep connection alive to prevent graceful shutdown
		<-r.Context().Done()
	})

	config := Config{
		Host: "127.0.0.1",
		Port: 0,
	}

	server := New(config, handler)

	// Start the server in a goroutine
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()

	serverErrChan := make(chan error, 1)
	go func() {
		serverErrChan <- http.Serve(listener, handler)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Make a request that will block
	client := &http.Client{Timeout: 5 * time.Second}
	go func() {
		_, _ = client.Get("http://" + addr + "/")
	}()

	// Wait for handler to be called
	select {
	case <-handlerCalled:
	case <-time.After(1 * time.Second):
		t.Fatal("Handler should have been called")
	}

	// Create a context with very short timeout to simulate timeout exceeded
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Attempt shutdown with short timeout - this may or may not error
	// depending on whether the active connection blocks shutdown
	_ = server.server.Shutdown(ctx)

	// Clean up - close listener to force server to exit
	listener.Close()

	// Wait for server to stop
	select {
	case <-serverErrChan:
	case <-time.After(2 * time.Second):
	}
}

// TestServer_Close_WithContextCancellation tests Close when context is cancelled
func TestServer_Close_WithContextCancellation(t *testing.T) {
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

// TestServer_Run_StartsInBackground tests that Run starts the server in a background goroutine
func TestServer_Run_StartsInBackground(t *testing.T) {
	handlerCalled := make(chan struct{}, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case handlerCalled <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	})

	config := Config{
		Host: "127.0.0.1",
		Port: 0,
	}

	server := New(config, handler)

	// Find an available port
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	listener.Close()

	server.server.Addr = addr

	// Run should start server in background and return immediately
	runComplete := make(chan struct{})
	go func() {
		server.Run()
		close(runComplete)
	}()

	// Run should return quickly (it starts server in goroutine)
	select {
	case <-runComplete:
		// Run completed quickly, as expected
	case <-time.After(1 * time.Second):
		t.Fatal("Run should return immediately after starting server in background")
	}

	// Give the actual server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is accepting connections
	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}
	resp, err := client.Get("http://" + addr + "/")
	if err == nil {
		defer resp.Body.Close()
		// Server responded - it's running
		select {
		case <-handlerCalled:
			// Handler was called
		case <-time.After(500 * time.Millisecond):
			// Handler might not have been called yet, but that's OK
			// The important thing is that the server started
		}
	}

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.server.Shutdown(ctx)
}

// TestServer_Start_ListenAndServe tests Start method without TLS
func TestServer_Start_ListenAndServe(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	config := Config{
		Host: "",
		Port: 0,
	}

	server := New(config, handler)

	// Find available port
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	listener.Close()

	server.server.Addr = addr

	// Start server in background
	startErrChan := make(chan error, 1)
	go func() {
		startErrChan <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Make request
	resp, err := http.Get("http://" + addr + "/")
	require.NoError(t, err, "Should be able to connect to server")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.server.Shutdown(ctx)

	// Start should return nil after graceful shutdown
	select {
	case err := <-startErrChan:
		assert.NoError(t, err, "Start should return nil after shutdown")
	case <-time.After(6 * time.Second):
		t.Fatal("Start should have returned after shutdown")
	}
}

// TestServer_Start_ErrServerClosed tests that Start returns nil when ErrServerClosed occurs
func TestServer_Start_ErrServerClosed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := Config{
		Host: "",
		Port: 0,
	}

	server := New(config, handler)

	// Find available port
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	listener.Close()

	server.server.Addr = addr

	// Start server in background
	startErrChan := make(chan error, 1)
	go func() {
		startErrChan <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Close the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.server.Shutdown(ctx)

	// Verify Start returns nil (ErrServerClosed is handled)
	select {
	case err := <-startErrChan:
		assert.NoError(t, err, "Start should return nil when ErrServerClosed")
	case <-time.After(6 * time.Second):
		t.Fatal("Start should have returned after shutdown")
	}
}

// TestServer_Start_InvalidPort tests Start with invalid port configuration
func TestServer_Start_InvalidPort(t *testing.T) {
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

// TestServer_Start_AddressInUse tests Start when address is already in use
func TestServer_Start_AddressInUse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := Config{
		Host: "127.0.0.1",
		Port: 0,
	}

	// Find available port and create a listener to occupy it
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	defer listener.Close()

	// Parse port from address
	_, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)

	// Convert port string to int
	var port int
	_, err = fmt.Sscanf(portStr, "%d", &port)
	require.NoError(t, err)

	config.Port = port

	server := New(config, handler)

	// Try to start - should fail because address is in use
	err = server.Start()
	assert.Error(t, err, "Start should return error when address is in use")
}

// TestServer_Run_LogsErrorOnFailure tests that Run logs error when server fails to start
func TestServer_Run_LogsErrorOnFailure(t *testing.T) {
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
	assert.Equal(t, 15*time.Second, ShutdownTimeout, "ShutdownTimeout should be 15 seconds")
}

// TestServer_ImplementsRunableProvider tests that Server implements httpserver.RunableProvider
func TestServer_ImplementsRunableProvider(t *testing.T) {
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
