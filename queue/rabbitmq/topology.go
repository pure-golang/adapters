package rabbitmq

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Definitions отражает формат JSON management API RabbitMQ.
// Может быть сериализован в JSON для CI/CD (rabbitmqctl import_definitions / load_definitions)
// или применён напрямую через AMQP для интеграционных тестов.
type Definitions struct {
	Vhosts      []VhostDef      `json:"vhosts"`
	Users       []UserDef       `json:"users,omitempty"`
	Permissions []PermissionDef `json:"permissions,omitempty"`
	Exchanges   []ExchangeDef   `json:"exchanges"`
	Queues      []QueueDef      `json:"queues"`
	Bindings    []BindingDef    `json:"bindings"`
}

type VhostDef struct {
	Name string `json:"name"`
}

type UserDef struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Tags     string `json:"tags"`
}

type PermissionDef struct {
	User      string `json:"user"`
	Vhost     string `json:"vhost"`
	Configure string `json:"configure"`
	Write     string `json:"write"`
	Read      string `json:"read"`
}

type ExchangeDef struct {
	Name       string         `json:"name"`
	Vhost      string         `json:"vhost"`
	Type       string         `json:"type"`
	Durable    bool           `json:"durable"`
	AutoDelete bool           `json:"auto_delete"`
	Internal   bool           `json:"internal"`
	Arguments  map[string]any `json:"arguments"`
}

type QueueDef struct {
	Name       string         `json:"name"`
	Vhost      string         `json:"vhost"`
	Durable    bool           `json:"durable"`
	AutoDelete bool           `json:"auto_delete"`
	Arguments  map[string]any `json:"arguments"`
}

type BindingDef struct {
	Source          string         `json:"source"`
	Vhost           string         `json:"vhost"`
	Destination     string         `json:"destination"`
	DestinationType string         `json:"destination_type"`
	RoutingKey      string         `json:"routing_key"`
	Arguments       map[string]any `json:"arguments"`
}

// JSON сериализует Definitions в форматированный JSON для использования с
// rabbitmqctl import_definitions или опцией load_definitions management-плагина.
func (d *Definitions) JSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// Apply объявляет все обменники, очереди и привязки через AMQP.
// Удобно для настройки топологии в интеграционных тестах без Management API.
// Порядок применения: Exchanges → Queues → Bindings.
func (d *Definitions) Apply(ch *amqp.Channel) error {
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
