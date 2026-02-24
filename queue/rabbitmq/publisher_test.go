package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pure-golang/adapters/queue/encoders"
)

func TestNewPublisher_DefaultEncoder(t *testing.T) {
	t.Parallel()
	// Test that publisher uses JSON encoder by default
	dialer := &Dialer{} // We just need a non-nil dialer for this test
	pub := NewPublisher(dialer, PublisherConfig{})

	assert.NotNil(t, pub)
	assert.Equal(t, encoders.JSON{}, pub.cfg.Encoder)
}

func TestNewPublisher_DefaultDeliveryMode(t *testing.T) {
	t.Parallel()
	dialer := &Dialer{}
	pub := NewPublisher(dialer, PublisherConfig{})

	assert.NotNil(t, pub)
	assert.Equal(t, Persistent, pub.cfg.DeliveryMode)
}
