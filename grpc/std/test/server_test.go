package std_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pure-golang/adapters/grpc/std"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

func TestServer_Start_ListenOnAvailablePort(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	s := std.New(std.Config{Port: freePort(t)}, func(srv *grpc.Server) {})

	err := s.Start()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, s.Close())
	}()
}

func TestServer_Start_AcceptsConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	port := freePort(t)
	s := std.New(std.Config{Port: port}, func(srv *grpc.Server) {})

	err := s.Start()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, s.Close())
	}()

	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}

func TestServer_Close_StopsServer(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	s := std.New(std.Config{Port: freePort(t)}, func(srv *grpc.Server) {})

	err := s.Start()
	require.NoError(t, err)

	err = s.Close()
	assert.NoError(t, err)
}

func TestServer_Close_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	s := std.New(std.Config{Port: freePort(t)}, func(srv *grpc.Server) {})

	err := s.Start()
	require.NoError(t, err)

	// Close должен завершиться в пределах ShutdownTimeout
	err = s.Close()
	assert.NoError(t, err)
}
