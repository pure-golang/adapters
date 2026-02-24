package rabbitmq_test

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/queue/rabbitmq"
)

func (s *RabbitMQSuite) TestDialer_Connect() {
	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	if t := s.T(); assert.NoError(t, dialer.Connect()) {
		assert.NoError(t, dialer.Close())
	}
}

func (s *RabbitMQSuite) TestDialer_Channel() {
	t := s.T()
	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	assert.NoError(t, dialer.Connect())
	t.Cleanup(func() {
		assert.NoError(t, dialer.Close())
	})

	channel, err := dialer.Channel()
	assert.NoError(t, err)
	assert.NoError(t, channel.Publish("", "", false, false, amqp.Publishing{}))

	t.Run("when dialer is closed", func(t *testing.T) {
		dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
		_, err := dialer.Channel()
		assert.ErrorIs(t, err, rabbitmq.ErrConnectionClosed)
	})
}

func (s *RabbitMQSuite) TestDialer_Close() {
	t := s.T()
	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	assert.NoError(t, dialer.Connect())
	assert.NoError(t, dialer.Close())

	t.Run("when dialer is closed", func(t *testing.T) {
		dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
		assert.NoError(t, dialer.Close())
	})
}

func (s *RabbitMQSuite) TestDialer_ReconnectWithRetryPolicy() {
	if testing.Short() {
		s.T().Skip("integration test")
	}

	t := s.T()

	// Test with ConstantInterval policy
	constantPolicy := rabbitmq.NewConstantInterval(100) // 100ns (very fast for testing)
	dialer := rabbitmq.NewDialer(s.RabbitURI, &rabbitmq.DialerOptions{
		RetryPolicy: constantPolicy,
	})

	err := dialer.Connect()
	require.NoError(t, err)

	// Verify connection works
	channel, err := dialer.Channel()
	require.NoError(t, err)
	assert.NoError(t, channel.Close())

	assert.NoError(t, dialer.Close())
}

func (s *RabbitMQSuite) TestDialer_ConnectMultipleTimes() {
	if testing.Short() {
		s.T().Skip("integration test")
	}

	t := s.T()

	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)

	// First connection
	err := dialer.Connect()
	require.NoError(t, err)

	// Second connection should still work (replaces the first)
	err = dialer.Connect()
	require.NoError(t, err)

	// Verify we can get a channel
	channel, err := dialer.Channel()
	require.NoError(t, err)
	assert.NoError(t, channel.Close())

	assert.NoError(t, dialer.Close())
}

func (s *RabbitMQSuite) TestDialer_CloseIdempotent() {
	if testing.Short() {
		s.T().Skip("integration test")
	}

	t := s.T()

	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	err := dialer.Connect()
	require.NoError(t, err)

	// First close
	err = dialer.Close()
	require.NoError(t, err)

	// Second close should be safe (no-op)
	err = dialer.Close()
	assert.NoError(t, err)

	// Third close should also be safe
	err = dialer.Close()
	assert.NoError(t, err)
}
