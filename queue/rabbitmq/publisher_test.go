package rabbitmq

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pure-golang/adapters/queue/encoders"
	"github.com/pure-golang/adapters/queue/rabbitmq/mocks"
)

func TestNewPublisher2_DefaultEncoder(t *testing.T) {
	t.Parallel()
	dialer := mocks.NewChannelProvider(t)
	dialer.EXPECT().Logger().Return(slog.Default())

	pub := NewPublisher2(PublisherConfig{}, dialer)

	assert.Equal(t, encoders.JSON{}, pub.cfg.Encoder)
}

func TestNewPublisher2_DefaultDeliveryMode(t *testing.T) {
	t.Parallel()
	dialer := mocks.NewChannelProvider(t)
	dialer.EXPECT().Logger().Return(slog.Default())

	pub := NewPublisher2(PublisherConfig{}, dialer)

	assert.Equal(t, Persistent, pub.cfg.DeliveryMode)
}
