package rabbitmq

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RabbitMQSuite struct {
	suite.Suite
	RabbitURI string
	container testcontainers.Container
}

func TestRabbitMQSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	suite.Run(t, new(RabbitMQSuite))
}

func (s *RabbitMQSuite) SetupSuite() {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "rabbitmq:management-alpine",
			ExposedPorts: []string{"5672/tcp"},
			WaitingFor:   wait.ForListeningPort("5672/tcp"),
		},
		Started: true,
	})
	require.NoError(s.T(), err)

	host, err := container.Host(ctx)
	require.NoError(s.T(), err)

	port, err := container.MappedPort(ctx, "5672/tcp")
	require.NoError(s.T(), err)

	s.container = container
	s.RabbitURI = "amqp://guest:guest@" + host + ":" + port.Port()
}

func (s *RabbitMQSuite) TearDownSuite() {
	if s.container != nil {
		require.NoError(s.T(), s.container.Terminate(context.Background()))
	}
}
