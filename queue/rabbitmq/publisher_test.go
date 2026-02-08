package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/pure-golang/adapters/queue"
	"github.com/pure-golang/adapters/queue/encoders"
)

func (s *RabbitMQSuite) TestPublisher_Publish() {
	t := s.T()
	ctx := context.Background()
	queueName := uuid.NewString()
	dialer := NewDialer(s.RabbitURI, nil)
	assert.NoError(t, dialer.Connect())
	channel, err := dialer.Channel()
	assert.NoError(t, err)
	_, err = channel.QueueDeclare(queueName, false, false, false, false, nil)
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, dialer.Close())
	})

	encoder := encoders.Text{}
	pub := NewPublisher(dialer, PublisherConfig{
		RoutingKey: queueName,
		Encoder:    encoder,
	})
	expMsg := queue.Message{
		Body: "message",
	}
	err = pub.Publish(ctx, expMsg)
	assert.NoError(t, err)

	deliveries, err := channel.Consume(queueName, "consumer", false, false, false, false, nil)
	assert.NoError(t, err)
	select {
	case msg := <-deliveries:
		assert.Equal(t, encoder.ContentType(), msg.ContentType)
		assert.Equal(t, Persistent, DeliveryMode(msg.DeliveryMode))
		assert.EqualValues(t, expMsg.Body, msg.Body)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "Couldn't wait handler call in Subscriber")
	}

	t.Run("when dialer re-connected", func(t *testing.T) {
		dialer := NewDialer(s.RabbitURI, nil)
		assert.NoError(t, dialer.Connect())
		time.Sleep(100 * time.Millisecond) // Give RabbitMQ time to stabilize
		assert.NoError(t, dialer.Close())
		time.Sleep(100 * time.Millisecond) // Give RabbitMQ time to cleanup
		assert.NoError(t, dialer.Connect())
		time.Sleep(100 * time.Millisecond) // Give RabbitMQ time to establish connection
		err := pub.Publish(ctx, expMsg)
		assert.NoError(t, err)
	})

	t.Run("when dialer is closed", func(t *testing.T) {
		dialer := NewDialer(s.RabbitURI, nil)
		pub := NewPublisher(dialer, PublisherConfig{
			RoutingKey: queueName,
			Encoder:    encoders.Text{},
		})
		err := pub.Publish(ctx, expMsg)
		assert.ErrorIs(t, err, ErrConnectionClosed)
	})
}
