package rabbitmq

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/queue"
)

func TestNewMultiQueueSubscriber_Defaults(t *testing.T) {
	t.Parallel()
	dialer := NewDefaultDialer("amqp://guest:guest@localhost:5672/")
	sub := NewMultiQueueSubscriber(dialer, MultiQueueOptions{MaxRetries: 3})
	assert.Equal(t, 1, sub.cfg.PrefetchCount)
	assert.Equal(t, 5*time.Second, sub.cfg.ReconnectDelay)
}

func (s *RabbitMQSuite) TestMultiQueueSubscriber_Listen_TwoQueues() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	prefix := uuid.NewString()[:8]
	q1 := prefix + ".queue1"
	q2 := prefix + ".queue2"

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	_, err = ch.QueueDeclare(q1, false, false, false, false, nil)
	require.NoError(t, err)
	_, err = ch.QueueDeclare(q2, false, false, false, false, nil)
	require.NoError(t, err)

	// Publish one message to each queue
	require.NoError(t, ch.Publish("", q1, false, false, amqp.Publishing{Body: []byte("from-q1")}))
	require.NoError(t, ch.Publish("", q2, false, false, amqp.Publishing{Body: []byte("from-q2")}))

	received := make(chan string, 2)
	makeHandler := func(label string) queue.Handler {
		return func(_ context.Context, d queue.Delivery) (bool, error) {
			received <- label + ":" + string(d.Body)
			return false, nil
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := NewMultiQueueSubscriber(dialer, MultiQueueOptions{MaxRetries: 3})
	go func() {
		_ = sub.Listen(ctx,
			QueueHandler{QueueName: q1, Handler: makeHandler("q1")},
			QueueHandler{QueueName: q2, Handler: makeHandler("q2")},
		)
	}()

	got := make(map[string]bool)
	timeout := time.After(5 * time.Second)
	for len(got) < 2 {
		select {
		case msg := <-received:
			got[msg] = true
		case <-timeout:
			t.Fatalf("timeout: received only %v", got)
		}
	}

	assert.True(t, got["q1:from-q1"])
	assert.True(t, got["q2:from-q2"])
	cancel()
}

func (s *RabbitMQSuite) TestMultiQueueSubscriber_SingleGoroutineSequential() {
	if testing.Short() {
		s.T().Skip("integration test")
	}
	t := s.T()
	prefix := uuid.NewString()[:8]
	q1 := prefix + ".queue1"
	q2 := prefix + ".queue2"

	dialer := NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	_, err = ch.QueueDeclare(q1, false, false, false, false, nil)
	require.NoError(t, err)
	_, err = ch.QueueDeclare(q2, false, false, false, false, nil)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		require.NoError(t, ch.Publish("", q1, false, false, amqp.Publishing{Body: []byte("q1")}))
		require.NoError(t, ch.Publish("", q2, false, false, amqp.Publishing{Body: []byte("q2")}))
	}

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	received := make(chan struct{}, 20)

	handler := func(_ context.Context, _ queue.Delivery) (bool, error) {
		cur := concurrent.Add(1)
		if cur > maxConcurrent.Load() {
			maxConcurrent.Store(cur)
		}
		time.Sleep(10 * time.Millisecond) // simulate work
		concurrent.Add(-1)
		received <- struct{}{}
		return false, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := NewMultiQueueSubscriber(dialer, MultiQueueOptions{
		PrefetchCount: 1,
		MaxRetries:    3,
	})
	go func() {
		_ = sub.Listen(ctx,
			QueueHandler{QueueName: q1, Handler: handler},
			QueueHandler{QueueName: q2, Handler: handler},
		)
	}()

	timeout := time.After(10 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-received:
		case <-timeout:
			t.Fatalf("timeout: received only %d of 10 messages", i)
		}
	}

	// With global Qos=1, at most 1 message should be in flight at a time
	assert.Equal(t, int32(1), maxConcurrent.Load(), "expected at most 1 concurrent handler")
	cancel()
}

func (s *RabbitMQSuite) TestMultiQueueSubscriber_Retry() {
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

	require.NoError(t, ch.Publish("", qName, false, false, amqp.Publishing{
		Body:         []byte("retry-me"),
		DeliveryMode: amqp.Persistent,
	}))

	handler := func(_ context.Context, _ queue.Delivery) (bool, error) {
		return true, errors.New("transient")
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub := NewMultiQueueSubscriber(dialer, MultiQueueOptions{
		MaxRetries: 3,
	})
	go func() {
		_ = sub.Listen(ctx, QueueHandler{
			QueueName:      qName,
			RetryQueueName: qRetry,
			Handler:        handler,
		})
	}()

	retryDeliveries, err := ch.Consume(qRetry, "", false, false, false, false, nil)
	require.NoError(t, err)

	select {
	case d := <-retryDeliveries:
		assert.Equal(t, []byte("retry-me"), d.Body)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: message did not reach retry queue")
	}
	cancel()
}

func (s *RabbitMQSuite) TestMultiQueueSubscriber_GracefulShutdown() {
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
	sub := NewMultiQueueSubscriber(dialer, MultiQueueOptions{MaxRetries: 3})

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = sub.Listen(ctx, QueueHandler{
			QueueName: qName,
			Handler: func(_ context.Context, _ queue.Delivery) (bool, error) {
				return false, nil
			},
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
