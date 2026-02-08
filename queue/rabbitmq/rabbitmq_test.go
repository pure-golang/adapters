package rabbitmq

import (
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RabbitMQSuite struct {
	suite.Suite
	RabbitURI string
	pool      *dockertest.Pool
	rabbitMQ  *dockertest.Resource
}

func TestRabbitMQSuite(t *testing.T) {
	suite.Run(t, new(RabbitMQSuite))
}

func (s *RabbitMQSuite) SetupSuite() {
	t := s.T()
	if testing.Short() {
		t.Skip("integration test is skipped")
	}

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	rabbitMQ, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "rabbitmq",
		Tag:        "management-alpine",
	}, func(config *docker.HostConfig) {
		// Remove AutoRemove to prevent container from being deleted during tests
		config.AutoRemove = false
	})
	require.NoError(t, err)

	// Wait for RabbitMQ to be ready
	err = pool.Retry(func() error {
		s.RabbitURI = "amqp://guest:guest@" + rabbitMQ.GetHostPort("5672/tcp")
		_, err := amqp.Dial(s.RabbitURI)
		return err
	})
	require.NoError(t, err)

	s.pool = pool
	s.rabbitMQ = rabbitMQ
}

func (s *RabbitMQSuite) TearDownSuite() {
	t := s.T()
	require.NoError(t, s.pool.Purge(s.rabbitMQ))
}
