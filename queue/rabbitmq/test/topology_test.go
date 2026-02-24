package rabbitmq_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/queue/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

// applyDefinitions объявляет все обменники, очереди и привязки через AMQP.
// Используется только в интеграционных тестах вместо Management API.
// Порядок применения: Exchanges → Queues → Bindings.
func applyDefinitions(ch *amqp.Channel, defs *rabbitmq.Definitions) error {
	for _, ex := range defs.Exchanges {
		args := toTable(ex.Arguments)
		if err := ch.ExchangeDeclare(
			ex.Name, ex.Type, ex.Durable, ex.AutoDelete, ex.Internal, false, args,
		); err != nil {
			return fmt.Errorf("declare exchange %q: %w", ex.Name, err)
		}
	}

	for _, q := range defs.Queues {
		args := toTable(q.Arguments)
		if _, err := ch.QueueDeclare(
			q.Name, q.Durable, q.AutoDelete, false, false, args,
		); err != nil {
			return fmt.Errorf("declare queue %q: %w", q.Name, err)
		}
	}

	for _, b := range defs.Bindings {
		args := toTable(b.Arguments)
		if err := ch.QueueBind(
			b.Destination, b.RoutingKey, b.Source, false, args,
		); err != nil {
			return fmt.Errorf("bind queue %q to exchange %q with key %q: %w",
				b.Destination, b.Source, b.RoutingKey, err)
		}
	}

	return nil
}

func toTable(m map[string]any) amqp.Table {
	if len(m) == 0 {
		return amqp.Table{}
	}
	t := make(amqp.Table, len(m))
	for k, v := range m {
		t[k] = v
	}
	return t
}

func (s *RabbitMQSuite) TestDefinitions_Apply() {
	if testing.Short() {
		s.T().Skip("integration test")
	}

	t := s.T()
	prefix := uuid.NewString()[:8]

	exchange := prefix + ".media"
	dlx := prefix + ".media.dlx"
	qMain := prefix + ".media.preview"
	qDLQ := prefix + ".media.preview.dlq"
	qRetry := prefix + ".media.preview.retry"

	defs := &rabbitmq.Definitions{
		Exchanges: []rabbitmq.ExchangeDef{
			{Name: exchange, Vhost: "/", Type: "topic", Durable: false, Arguments: map[string]any{}},
			{Name: dlx, Vhost: "/", Type: "topic", Durable: false, Arguments: map[string]any{}},
		},
		Queues: []rabbitmq.QueueDef{
			{Name: qMain, Vhost: "/", Durable: false, Arguments: map[string]any{
				"x-dead-letter-exchange":    dlx,
				"x-dead-letter-routing-key": qDLQ,
			}},
			{Name: qDLQ, Vhost: "/", Durable: false, Arguments: map[string]any{}},
			{Name: qRetry, Vhost: "/", Durable: false, Arguments: map[string]any{
				"x-message-ttl":             int32(60000),
				"x-dead-letter-exchange":    exchange,
				"x-dead-letter-routing-key": qMain,
			}},
		},
		Bindings: []rabbitmq.BindingDef{
			{Source: exchange, Vhost: "/", Destination: qMain, DestinationType: "queue", RoutingKey: qMain, Arguments: map[string]any{}},
			{Source: dlx, Vhost: "/", Destination: qDLQ, DestinationType: "queue", RoutingKey: qDLQ, Arguments: map[string]any{}},
		},
	}

	dialer := rabbitmq.NewDialer(s.RabbitURI, nil)
	require.NoError(t, dialer.Connect())
	t.Cleanup(func() { assert.NoError(t, dialer.Close()) })

	ch, err := dialer.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, ch.Close()) })

	require.NoError(t, applyDefinitions(ch, defs))

	// Verify queues are accessible by passively declaring them
	for _, name := range []string{qMain, qDLQ, qRetry} {
		_, err := ch.QueueDeclarePassive(name, false, false, false, false, nil)
		assert.NoError(t, err, "queue %q should exist", name)
	}
}
