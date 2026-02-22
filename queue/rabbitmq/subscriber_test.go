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

// --- unit tests (no docker) ---

func TestDeathCount_NoHeader(t *testing.T) {
	t.Parallel()
	d := &amqp.Delivery{}
	assert.Equal(t, 0, deathCount(d))
}

func TestDeathCount_WithHeader(t *testing.T) {
	t.Parallel()
	d := &amqp.Delivery{
		Headers: amqp.Table{
			"x-death": []any{
				amqp.Table{"count": int64(2), "queue": "media.preview"},
				amqp.Table{"count": int64(1), "queue": "media.preview.retry"},
			},
		},
	}
	assert.Equal(t, 3, deathCount(d))
}

func TestDeathCount_WrongType(t *testing.T) {
	t.Parallel()
	d := &amqp.Delivery{
		Headers: amqp.Table{"x-death": "not a slice"},
	}
	assert.Equal(t, 0, deathCount(d))
}

func TestNewSubscriber_Defaults(t *testing.T) {
	t.Parallel()
	dialer := NewDefaultDialer("amqp://guest:guest@localhost:5672/")
	sub := NewSubscriber(dialer, "test.queue", SubscriberOptions{MaxRetries: 3})

	assert.NotEmpty(t, sub.name)
	assert.Equal(t, "test.queue", sub.queueName)
	assert.Equal(t, 1, sub.cfg.PrefetchCount)
	assert.Equal(t, 3, sub.cfg.MaxRetries)
	assert.Equal(t, "test.queue.retry", sub.cfg.RetryQueueName)
}

func TestNewSubscriber_CustomRetryQueue(t *testing.T) {
	t.Parallel()
	dialer := NewDefaultDialer("amqp://guest:guest@localhost:5672/")
	sub := NewSubscriber(dialer, "test.queue", SubscriberOptions{
		MaxRetries:     5,
		RetryQueueName: "custom.retry",
	})
	assert.Equal(t, "custom.retry", sub.cfg.RetryQueueName)
}

func TestNewDelivery_Headers(t *testing.T) {
	t.Parallel()
	d := &amqp.Delivery{
		Headers: amqp.Table{
			"key1": "value1",
			"key2": 123,
			"key3": true,
		},
		Body: []byte("body"),
	}
	got := newDelivery(d)
	assert.Equal(t, "value1", got.Headers["key1"])
	assert.Equal(t, "123", got.Headers["key2"])
	assert.Equal(t, "true", got.Headers["key3"])
	assert.Equal(t, []byte("body"), got.Body)
}

// --- integration tests ---

func (s *RabbitMQSuite) TestSubscriber_Listen_Ack() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	qName := uuid.NewString()

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	_, err = ch.QueueDeclare(qName, false, false, false, false, nil)
	require.NoError(t, err)

	pub := NewPublisher(dialer, PublisherConfig{RoutingKey: qName, Encoder: encoders.Text{}})

	received := make(chan struct{}, 1)
	handler := func(_ context.Context, _ queue.Delivery) (bool, error) {
		received <- struct{}{}
		return false, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := NewSubscriber(dialer, qName, SubscriberOptions{MaxRetries: 3})
	go sub.Listen(ctx, handler)
	time.Sleep(100 * time.Millisecond)

	require.NoError(t, pub.Publish(context.Background(), queue.Message{Body: "hello"}))

	select {
	case <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}
	cancel()
}

func (s *RabbitMQSuite) TestSubscriber_Listen_DLQ() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	prefix := uuid.NewString()[:8]
	qName := prefix + ".main"
	qRetry := prefix + ".retry"
	dlxName := prefix + ".dlx"
	qDLQ := prefix + ".dlq"

	defs := &Definitions{
		Exchanges: []ExchangeDef{
			{Name: dlxName, Vhost: "/", Type: "direct", Durable: false, Arguments: map[string]any{}},
		},
		Queues: []QueueDef{
			{Name: qName, Vhost: "/", Durable: false, Arguments: map[string]any{
				"x-dead-letter-exchange":    dlxName,
				"x-dead-letter-routing-key": qDLQ,
			}},
			{Name: qRetry, Vhost: "/", Durable: false, Arguments: map[string]any{}},
			{Name: qDLQ, Vhost: "/", Durable: false, Arguments: map[string]any{}},
		},
		Bindings: []BindingDef{
			{Source: dlxName, Vhost: "/", Destination: qDLQ, DestinationType: "queue", RoutingKey: qDLQ, Arguments: map[string]any{}},
		},
	}

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	require.NoError(t, defs.applyDefinitions(ch))

	// Publish a message with x-death count already at MaxRetries so subscriber nacks it to DLQ immediately.
	maxRetries := 3
	err = ch.Publish("", qName, false, false, amqp.Publishing{
		Body:         []byte("dead"),
		DeliveryMode: amqp.Persistent,
		Headers: amqp.Table{
			"x-death": []any{
				amqp.Table{"count": int64(maxRetries), "queue": qName},
			},
		},
	})
	require.NoError(t, err)

	handler := func(_ context.Context, _ queue.Delivery) (bool, error) {
		return true, errors.New("always fails")
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := NewSubscriber(dialer, qName, SubscriberOptions{
		MaxRetries:     maxRetries,
		RetryQueueName: qRetry,
	})
	go sub.Listen(ctx, handler)

	dlqDeliveries, err := ch.Consume(qDLQ, "", false, false, false, false, nil)
	require.NoError(t, err)

	select {
	case <-dlqDeliveries:
		t.Log("message reached DLQ as expected")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: message did not reach DLQ")
	}
	cancel()
}

func (s *RabbitMQSuite) TestSubscriber_Listen_RetryQueue() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	prefix := uuid.NewString()[:8]
	qName := prefix + ".main"
	qRetry := prefix + ".retry"

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	_, err = ch.QueueDeclare(qName, false, false, false, false, nil)
	require.NoError(t, err)
	_, err = ch.QueueDeclare(qRetry, false, false, false, false, nil)
	require.NoError(t, err)

	// x-death count is 0 < MaxRetries=3, so the message should go to retry queue
	require.NoError(t, ch.Publish("", qName, false, false, amqp.Publishing{
		Body:         []byte("retry-me"),
		DeliveryMode: amqp.Persistent,
	}))

	handler := func(_ context.Context, _ queue.Delivery) (bool, error) {
		return true, errors.New("transient error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := NewSubscriber(dialer, qName, SubscriberOptions{
		MaxRetries:     3,
		RetryQueueName: qRetry,
	})
	go sub.Listen(ctx, handler)

	retryDeliveries, err := ch.Consume(qRetry, "", false, false, false, false, nil)
	require.NoError(t, err)

	select {
	case d := <-retryDeliveries:
		assert.Equal(t, []byte("retry-me"), d.Body)
		t.Log("message reached retry queue as expected")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: message did not reach retry queue")
	}
	cancel()
}

func (s *RabbitMQSuite) TestSubscriber_Listen_GracefulShutdown() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	qName := uuid.NewString()

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	_, err = ch.QueueDeclare(qName, false, false, false, false, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	sub := NewSubscriber(dialer, qName, SubscriberOptions{MaxRetries: 3})

	done := make(chan struct{})
	go func() {
		defer close(done)
		sub.Listen(ctx, func(_ context.Context, _ queue.Delivery) (bool, error) {
			return false, nil
		})
	}()
	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Listen did not return after context cancellation")
	}
}

func (s *RabbitMQSuite) TestSubscriber_MessageTimeout() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	qName := uuid.NewString()

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	_, err = ch.QueueDeclare(qName, false, false, false, false, nil)
	require.NoError(t, err)

	pub := NewPublisher(dialer, PublisherConfig{RoutingKey: qName, Encoder: encoders.Text{}})
	require.NoError(t, pub.Publish(context.Background(), queue.Message{Body: "timeout-test"}))

	ctxReceived := make(chan context.Context, 1)
	handler := func(ctx context.Context, _ queue.Delivery) (bool, error) {
		ctxReceived <- ctx
		return false, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := NewSubscriber(dialer, qName, SubscriberOptions{
		MaxRetries:     3,
		MessageTimeout: 10 * time.Second,
	})
	go sub.Listen(ctx, handler)

	select {
	case hCtx := <-ctxReceived:
		deadline, ok := hCtx.Deadline()
		assert.True(t, ok, "handler context should have a deadline")
		assert.False(t, deadline.IsZero())
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}
