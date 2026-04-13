package rabbitmq

import (
	"log/slog"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"

	"github.com/pure-golang/adapters/queue/rabbitmq/mocks"
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

func TestNewSubscriber2_Defaults(t *testing.T) {
	t.Parallel()
	dialer := mocks.NewChannelProvider(t)
	dialer.EXPECT().Logger().Return(slog.Default())

	sub := NewSubscriber2(SubscriberConfig{QueueName: "test.queue", MaxRetries: 3}, dialer)

	assert.NotEmpty(t, sub.name)
	assert.Equal(t, "test.queue", sub.queueName)
	assert.Equal(t, 1, sub.cfg.PrefetchCount)
	assert.Equal(t, 3, sub.cfg.MaxRetries)
	assert.Equal(t, "test.queue.retry", sub.cfg.RetryQueueName)
}

func TestNewSubscriber2_CustomRetryQueue(t *testing.T) {
	t.Parallel()
	dialer := mocks.NewChannelProvider(t)
	dialer.EXPECT().Logger().Return(slog.Default())

	sub := NewSubscriber2(SubscriberConfig{
		QueueName:      "test.queue",
		MaxRetries:     5,
		RetryQueueName: "custom.retry",
	}, dialer)

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
