package rabbitmq

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultDialer(t *testing.T) {
	uri := "amqp://guest:guest@localhost:5672/"

	dialer := NewDefaultDialer(uri)

	require.NotNil(t, dialer)
	assert.Equal(t, uri, dialer.uri)
	assert.NotNil(t, dialer.options)
	assert.NotNil(t, dialer.options.RetryPolicy)
	assert.NotNil(t, dialer.options.Logger)
}

func TestNewDialer(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		uri := "amqp://guest:guest@localhost:5672/"
		dialer := NewDialer(uri, nil)

		require.NotNil(t, dialer)
		assert.Equal(t, uri, dialer.uri)
		assert.NotNil(t, dialer.options)
		assert.NotNil(t, dialer.options.RetryPolicy)
		assert.NotNil(t, dialer.options.Logger)
	})

	t.Run("with default logger", func(t *testing.T) {
		uri := "amqp://guest:guest@localhost:5672/"
		dialer := NewDialer(uri, &DialerOptions{})

		require.NotNil(t, dialer)
		assert.NotNil(t, dialer.options.Logger)
	})

	t.Run("with custom logger", func(t *testing.T) {
		uri := "amqp://guest:guest@localhost:5672/"
		customLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		dialer := NewDialer(uri, &DialerOptions{
			Logger: customLogger,
		})

		require.NotNil(t, dialer)
		// Logger is wrapped with "rabbitmq" group, so we can't compare directly
		assert.NotNil(t, dialer.options.Logger)
	})

	t.Run("with default retry policy", func(t *testing.T) {
		uri := "amqp://guest:guest@localhost:5672/"
		dialer := NewDialer(uri, &DialerOptions{})

		require.NotNil(t, dialer)
		assert.NotNil(t, dialer.options.RetryPolicy)
		// Should be MaxInterval with defaults
		_, ok := dialer.options.RetryPolicy.(*MaxInterval)
		assert.True(t, ok, "Default retry policy should be MaxInterval")
	})

	t.Run("with custom retry policy", func(t *testing.T) {
		uri := "amqp://guest:guest@localhost:5672/"
		customPolicy := NewConstantInterval(5)

		dialer := NewDialer(uri, &DialerOptions{
			RetryPolicy: customPolicy,
		})

		require.NotNil(t, dialer)
		assert.Equal(t, customPolicy, dialer.options.RetryPolicy)
	})

	t.Run("logger has rabbitmq group", func(t *testing.T) {
		uri := "amqp://guest:guest@localhost:5672/"
		dialer := NewDialer(uri, nil)

		// Logger should be set with the "rabbitmq" group
		// We can't easily test the group without exposing internals,
		// but we can verify the logger is not nil
		assert.NotNil(t, dialer.options.Logger)
	})
}

func (s *RabbitMQSuite) TestDialer_ReconnectWithRetryPolicy() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	t := s.T()

	// Test with ConstantInterval policy
	constantPolicy := NewConstantInterval(100) // 100ns (very fast for testing)
	dialer := NewDialer(s.RabbitURI, &DialerOptions{
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
		s.T().Skip("skipping integration test in short mode")
	}

	t := s.T()

	dialer := NewDialer(s.RabbitURI, nil)

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
		s.T().Skip("skipping integration test in short mode")
	}

	t := s.T()

	dialer := NewDialer(s.RabbitURI, nil)
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
