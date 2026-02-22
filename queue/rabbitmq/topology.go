package rabbitmq

import (
	"encoding/json"
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
