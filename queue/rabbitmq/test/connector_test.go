package rabbitmq_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/queue/rabbitmq"
)

func (s *RabbitMQSuite) TestConnector_WaitForExchange() {
	if testing.Short() {
		s.T().Skip("integration test")
	}

	t := s.T()
	exchangeName := "test." + uuid.NewString()[:8]

	// Arrange: объявляем exchange и сразу проверяем, что WaitForExchange проходит
	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { dialer.Close() })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { ch.Close() })

	require.NoError(t, ch.ExchangeDeclare(exchangeName, "topic", false, true, false, false, amqp.Table{}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	connector := rabbitmq.NewConnector(dialer)

	// Act
	err = connector.WaitForExchange(ctx, exchangeName)

	// Assert
	require.NoError(t, err)
}

func (s *RabbitMQSuite) TestConnector_WaitForQueue() {
	if testing.Short() {
		s.T().Skip("integration test")
	}

	t := s.T()
	queueName := "test." + uuid.NewString()[:8]

	// Arrange: объявляем очередь и сразу проверяем, что WaitForQueue проходит
	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { dialer.Close() })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { ch.Close() })

	_, err = ch.QueueDeclare(queueName, false, true, false, false, amqp.Table{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	connector := rabbitmq.NewConnector(dialer)

	// Act
	err = connector.WaitForQueue(ctx, queueName)

	// Assert
	require.NoError(t, err)
}
