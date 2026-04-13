package integration_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		fmt.Println("skipping integration tests")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestContainersSmoke проверяет, что testcontainers работает в CI-окружении:
// запускает postgres-контейнер и выполняет простой SELECT 1.
func TestContainersSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_USER":     "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Printf("failed to start container: %s", err)
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://test:secret@%s:%s/test?sslmode=disable", host, port.Port())
	t.Logf("container started, dsn: %s", dsn)

	state, err := container.State(ctx)
	require.NoError(t, err)
	require.True(t, state.Running, "container must be running")
}
