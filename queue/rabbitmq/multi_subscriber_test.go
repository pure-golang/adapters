package rabbitmq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMultiQueueSubscriber_Defaults(t *testing.T) {
	t.Parallel()
	dialer := NewDefaultDialer("amqp://guest:guest@localhost:5672/")
	sub := NewMultiQueueSubscriber(dialer, MultiQueueOptions{MaxRetries: 3})
	assert.Equal(t, 1, sub.cfg.PrefetchCount)
	assert.Equal(t, 5*time.Second, sub.cfg.ReconnectDelay)
}
