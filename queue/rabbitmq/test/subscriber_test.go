package rabbitmq_test

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
	"github.com/pure-golang/adapters/queue/rabbitmq"
)

func (s *RabbitMQSuite) TestSubscriber_Listen_Ack() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	qName := uuid.NewString()

	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ch.Close() })
	_, err = ch.QueueDeclare(qName, false, false, false, false, nil)
	require.NoError(t, err)

	pub := rabbitmq.NewPublisher(dialer, rabbitmq.PublisherConfig{RoutingKey: qName, Encoder: encoders.Text{}})

	received := make(chan struct{}, 1)
	handler := func(_ context.Context, _ queue.Delivery) (bool, error) {
		received <- struct{}{}
		return false, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := rabbitmq.NewSubscriber(dialer, qName, rabbitmq.SubscriberOptions{MaxRetries: 3})
	done := make(chan struct{})
	go func() {
		defer close(done)
		sub.Listen(ctx, handler)
	}()
	time.Sleep(100 * time.Millisecond)

	require.NoError(t, pub.Publish(context.Background(), queue.Message{Body: "hello"}))

	select {
	case <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}
	cancel()
	<-done
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

	defs := &rabbitmq.Definitions{
		Exchanges: []rabbitmq.ExchangeDef{
			{Name: dlxName, Vhost: "/", Type: "direct", Durable: false, Arguments: map[string]any{}},
		},
		Queues: []rabbitmq.QueueDef{
			{Name: qName, Vhost: "/", Durable: false, Arguments: map[string]any{
				"x-dead-letter-exchange":    dlxName,
				"x-dead-letter-routing-key": qDLQ,
			}},
			{Name: qRetry, Vhost: "/", Durable: false, Arguments: map[string]any{}},
			{Name: qDLQ, Vhost: "/", Durable: false, Arguments: map[string]any{}},
		},
		Bindings: []rabbitmq.BindingDef{
			{Source: dlxName, Vhost: "/", Destination: qDLQ, DestinationType: "queue", RoutingKey: qDLQ, Arguments: map[string]any{}},
		},
	}

	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ch.Close() })
	require.NoError(t, applyDefinitions(ch, defs))

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
	sub := rabbitmq.NewSubscriber(dialer, qName, rabbitmq.SubscriberOptions{
		MaxRetries:     maxRetries,
		RetryQueueName: qRetry,
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		sub.Listen(ctx, handler)
	}()

	dlqDeliveries, err := ch.Consume(qDLQ, "", false, false, false, false, nil)
	require.NoError(t, err)

	select {
	case <-dlqDeliveries:
		t.Log("message reached DLQ as expected")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: message did not reach DLQ")
	}
	cancel()
	<-done
}

func (s *RabbitMQSuite) TestSubscriber_Listen_RetryQueue() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	prefix := uuid.NewString()[:8]
	qName := prefix + ".main"
	qRetry := prefix + ".retry"

	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ch.Close() })
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
	sub := rabbitmq.NewSubscriber(dialer, qName, rabbitmq.SubscriberOptions{
		MaxRetries:     3,
		RetryQueueName: qRetry,
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		sub.Listen(ctx, handler)
	}()

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
	<-done
}

func (s *RabbitMQSuite) TestSubscriber_Listen_GracefulShutdown() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	qName := uuid.NewString()

	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ch.Close() })
	_, err = ch.QueueDeclare(qName, false, false, false, false, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	sub := rabbitmq.NewSubscriber(dialer, qName, rabbitmq.SubscriberOptions{MaxRetries: 3})

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

	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ch.Close() })
	_, err = ch.QueueDeclare(qName, false, false, false, false, nil)
	require.NoError(t, err)

	pub := rabbitmq.NewPublisher(dialer, rabbitmq.PublisherConfig{RoutingKey: qName, Encoder: encoders.Text{}})
	require.NoError(t, pub.Publish(context.Background(), queue.Message{Body: "timeout-test"}))

	ctxReceived := make(chan context.Context, 1)
	handler := func(ctx context.Context, _ queue.Delivery) (bool, error) {
		ctxReceived <- ctx
		return false, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := rabbitmq.NewSubscriber(dialer, qName, rabbitmq.SubscriberOptions{
		MaxRetries:     3,
		MessageTimeout: 10 * time.Second,
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		sub.Listen(ctx, handler)
	}()

	select {
	case hCtx := <-ctxReceived:
		deadline, ok := hCtx.Deadline()
		assert.True(t, ok, "handler context should have a deadline")
		assert.False(t, deadline.IsZero())
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}
	cancel()
	<-done
}
