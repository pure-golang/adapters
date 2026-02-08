package rabbitmq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func (s *RabbitMQSuite) TestDialer_Connect() {
	dialer := NewDialer(s.RabbitURI, nil)
	if t := s.T(); assert.NoError(t, dialer.Connect()) {
		assert.NoError(t, dialer.Close())
	}
}

func (s *RabbitMQSuite) TestDialer_Channel() {
	t := s.T()
	dialer := NewDialer(s.RabbitURI, nil)
	assert.NoError(t, dialer.Connect())
	t.Cleanup(func() {
		assert.NoError(t, dialer.Close())
	})

	channel, err := dialer.Channel()
	assert.NoError(t, err)
	assert.NoError(t, channel.Publish("", "", false, false, amqp.Publishing{}))

	t.Run("when dialer is closed", func(t *testing.T) {
		dialer := NewDialer(s.RabbitURI, nil)
		_, err := dialer.Channel()
		assert.ErrorIs(t, err, ErrConnectionClosed)
	})
}

func (s *RabbitMQSuite) TestDialer_Close() {
	t := s.T()
	dialer := NewDialer(s.RabbitURI, nil)
	assert.NoError(t, dialer.Connect())
	assert.NoError(t, dialer.Close())

	t.Run("when dialer is closed", func(t *testing.T) {
		dialer := NewDialer(s.RabbitURI, nil)
		assert.NoError(t, dialer.Close())
	})
}
