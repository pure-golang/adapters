package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// applyDefinitions объявляет все обменники, очереди и привязки через AMQP.
// Используется только в интеграционных тестах вместо Management API.
// Порядок применения: Exchanges → Queues → Bindings.
func (d *Definitions) applyDefinitions(ch *amqp.Channel) error {
	for _, ex := range d.Exchanges {
		args := toTable(ex.Arguments)
		if err := ch.ExchangeDeclare(
			ex.Name, ex.Type, ex.Durable, ex.AutoDelete, ex.Internal, false, args,
		); err != nil {
			return fmt.Errorf("declare exchange %q: %w", ex.Name, err)
		}
	}

	for _, q := range d.Queues {
		args := toTable(q.Arguments)
		if _, err := ch.QueueDeclare(
			q.Name, q.Durable, q.AutoDelete, false, false, args,
		); err != nil {
			return fmt.Errorf("declare queue %q: %w", q.Name, err)
		}
	}

	for _, b := range d.Bindings {
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
