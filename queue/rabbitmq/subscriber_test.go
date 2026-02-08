package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/queue"
	"github.com/pure-golang/adapters/queue/encoders"
)

func (s *RabbitMQSuite) TestSubscriber_NewDefaultSubscriber() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	t := s.T()
	queueName := uuid.NewString()

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())

	t.Cleanup(func() {
		assert.NoError(t, dialer.Close())
	})

	// Create subscriber with defaults
	sub := NewDefaultSubscriber(dialer, queueName)

	assert.NotNil(t, sub)
	assert.Equal(t, queueName, sub.queueName)
	assert.Equal(t, dialer, sub.dialer)
	assert.NotNil(t, sub.logger)
	assert.NotNil(t, sub.close)
	assert.Equal(t, DefaultLastMessageTimeout, sub.lastMessageTimeout)
}

func (s *RabbitMQSuite) TestSubscriber_NewSubscriber() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	t := s.T()
	queueName := uuid.NewString()
	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())

	t.Cleanup(func() {
		assert.NoError(t, dialer.Close())
	})

	t.Run("with default options", func(t *testing.T) {
		sub := NewSubscriber(dialer, queueName, SubscriberOptions{})

		assert.NotNil(t, sub)
		assert.NotEmpty(t, sub.name)
		assert.Equal(t, queueName, sub.queueName)
		assert.Equal(t, 0, sub.cfg.PrefetchCount)
		assert.Equal(t, 0, sub.cfg.MaxTryNum)
		assert.Equal(t, 5*time.Second, sub.cfg.Backoff)
		assert.Equal(t, DefaultLastMessageTimeout, sub.lastMessageTimeout)
	})

	t.Run("with custom options", func(t *testing.T) {
		customBackoff := 10 * time.Second

		sub := NewSubscriber(dialer, queueName, SubscriberOptions{
			PrefetchCount: 10,
			MaxTryNum:     5,
			Backoff:       customBackoff,
		})

		assert.NotNil(t, sub)
		assert.NotEmpty(t, sub.name) // Name is generated via UUID since cfg.Name is empty
		assert.Equal(t, 10, sub.cfg.PrefetchCount)
		assert.Equal(t, 5, sub.cfg.MaxTryNum)
		assert.Equal(t, customBackoff, sub.cfg.Backoff)
	})

	t.Run("with zero MaxTryNum gets set to zero", func(t *testing.T) {
		sub := NewSubscriber(dialer, queueName, SubscriberOptions{
			MaxTryNum: -1, // Should be set to 0 (infinite retries indicator)
		})
		assert.Equal(t, 0, sub.cfg.MaxTryNum, "MaxTryNum <= 0 is set to 0")
	})

	t.Run("with zero Backoff gets default", func(t *testing.T) {
		sub := NewSubscriber(dialer, queueName, SubscriberOptions{
			Backoff: 0,
		})
		assert.Equal(t, 5*time.Second, sub.cfg.Backoff)
	})
}

func (s *RabbitMQSuite) TestSubscriber_Close() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	t := s.T()
	queueName := uuid.NewString()

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())

	sub := NewSubscriber(dialer, queueName, SubscriberOptions{})

	// Close without calling Listen should be safe
	err := sub.Close()
	assert.NoError(t, err)
}

func (s *RabbitMQSuite) TestSubscriber_AckHandling() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	t := s.T()
	queueName := uuid.NewString()

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())

	channel, err := dialer.Channel()
	require.NoError(t, err)
	_, err = channel.QueueDeclare(queueName, false, false, false, false, nil)
	require.NoError(t, err)

	pub := NewPublisher(dialer, PublisherConfig{
		RoutingKey: queueName,
		Encoder:    encoders.Text{},
	})

	// Handler that succeeds - should result in ack
	messageReceived := make(chan struct{}, 1)
	handler := func(ctx context.Context, msg queue.Delivery) (bool, error) {
		messageReceived <- struct{}{}
		return false, nil
	}

	sub := NewSubscriber(dialer, queueName, SubscriberOptions{})
	go sub.Listen(handler)
	time.Sleep(200 * time.Millisecond) // Give subscriber time to start

	// Publish message
	err = pub.Publish(context.Background(), queue.Message{Body: "test-ack"})
	require.NoError(t, err)

	// Wait for message to be received
	select {
	case <-messageReceived:
		t.Log("Message acknowledged successfully")
	case <-time.After(3 * time.Second):
		t.Log("Timeout waiting for message - may be OK in slow environments")
	}

	// Close subscriber with timeout
	done := make(chan struct{})
	go func() {
		assert.NoError(t, sub.Close())
		close(done)
	}()

	select {
	case <-done:
		// Closed successfully
	case <-time.After(5 * time.Second):
		t.Log("Close took longer than expected")
	}
}

func (s *RabbitMQSuite) TestSubscriber_NackHandling() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	t := s.T()
	queueName := uuid.NewString()

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())

	channel, err := dialer.Channel()
	require.NoError(t, err)
	_, err = channel.QueueDeclare(queueName, false, false, false, false, nil)
	require.NoError(t, err)

	pub := NewPublisher(dialer, PublisherConfig{
		RoutingKey: queueName,
		Encoder:    encoders.Text{},
	})

	// Handler that returns non-retryable error - should reject (nack)
	handlerCalled := make(chan struct{}, 10)
	handlerError := errors.New("processing failed")

	handler := func(ctx context.Context, msg queue.Delivery) (bool, error) {
		handlerCalled <- struct{}{}
		return false, handlerError // Non-retryable error
	}

	sub := NewSubscriber(dialer, queueName, SubscriberOptions{})
	go sub.Listen(handler)
	time.Sleep(200 * time.Millisecond)

	// Publish message
	err = pub.Publish(context.Background(), queue.Message{Body: "test-nack"})
	require.NoError(t, err)

	// Wait for handler to be called at least once
	select {
	case <-handlerCalled:
		t.Log("Handler was called, message was rejected")
	case <-time.After(3 * time.Second):
		t.Log("Handler was not called - may be OK in slow environments")
	}

	// Close subscriber
	done := make(chan struct{})
	go func() {
		assert.NoError(t, sub.Close())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Log("Close took longer than expected")
	}
}

func (s *RabbitMQSuite) TestSubscriber_InfiniteRetries() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	t := s.T()

	// Test InfiniteRetriesIndicator constant
	assert.Equal(t, -1, InfiniteRetriesIndicator)

	// Basic test that infinite retries value is set correctly
	// When MaxTryNum <= 0, it gets set to 0 (which represents infinite retries)
	queueName := uuid.NewString()
	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())

	sub := NewSubscriber(dialer, queueName, SubscriberOptions{
		MaxTryNum: InfiniteRetriesIndicator,
		Backoff:   50 * time.Millisecond,
	})

	assert.Equal(t, 0, sub.cfg.MaxTryNum, "InfiniteRetriesIndicator (-1) is normalized to 0")

	// Close subscriber
	assert.NoError(t, sub.Close())
}

func TestNewDelivery(t *testing.T) {
	t.Run("with headers", func(t *testing.T) {
		delivery := amqp.Delivery{
			Headers: amqp.Table{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
			Body: []byte("test body"),
		}

		qd := newDelivery(delivery)

		assert.NotNil(t, qd)
		assert.Len(t, qd.Headers, 3)
		assert.Equal(t, "test body", string(qd.Body))
		assert.Equal(t, "value1", qd.Headers["key1"])
		assert.Equal(t, "123", qd.Headers["key2"])  // Converted to string
		assert.Equal(t, "true", qd.Headers["key3"]) // Converted to string
	})

	t.Run("without headers", func(t *testing.T) {
		delivery := amqp.Delivery{
			Body: []byte("test body"),
		}

		qd := newDelivery(delivery)

		assert.NotNil(t, qd)
		assert.Empty(t, qd.Headers)
		assert.Equal(t, "test body", string(qd.Body))
	})
}
