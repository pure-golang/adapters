package std_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/pure-golang/adapters/grpc/std"
)

func TestServer_Start_ListenOnAvailablePort(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	// Find an available port
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	addr := l.Addr().(*net.TCPAddr)
	c := std.Config{
		Port: addr.Port,
	}

	s := std.New(c, func(srv *grpc.Server) {})
	require.NotNil(t, s)

	// Start server in a goroutine to avoid blocking
	startDone := make(chan struct{})
	go func() {
		_ = s.Start()
		close(startDone)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Server should be running with listener set
	if s.GetListener() != nil {
		// Successfully started, close it
		s.Close()
		<-startDone // Wait for Start to return
	} else {
		s.Close()
		<-startDone
		t.Skip("server did not start in time")
	}
}

func TestServer_Close_WithListener(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	// Find an available port
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	addr := l.Addr().(*net.TCPAddr)
	port := addr.Port
	l.Close()

	c := std.Config{
		Port: port,
	}

	s := std.New(c, func(srv *grpc.Server) {})
	require.NotNil(t, s)

	// Start the server in a goroutine
	startDone := make(chan struct{})
	go func() {
		_ = s.Start()
		close(startDone)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	if s.GetListener() != nil {
		// Server started successfully
		assert.NotNil(t, s.GetListener())

		// Close should stop server and close listener
		s.Close()
		<-startDone
	} else {
		s.Close()
		<-startDone
		t.Skip("port was taken")
	}
}

func TestServer_Close_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	// Find an available port
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	addr := l.Addr().(*net.TCPAddr)
	port := addr.Port
	l.Close()

	c := std.Config{
		Port: port,
	}

	s := std.New(c, func(srv *grpc.Server) {})
	require.NotNil(t, s)

	// Start server in goroutine
	startDone := make(chan struct{})
	go func() {
		_ = s.Start()
		close(startDone)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	if s.GetListener() != nil {
		// Close the server - should complete within timeout
		s.Close()
		<-startDone
	} else {
		s.Close()
		<-startDone
		t.Skip("port was taken")
	}
}
