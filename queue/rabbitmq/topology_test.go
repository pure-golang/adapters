package rabbitmq

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinitions_JSON(t *testing.T) {
	t.Parallel()

	defs := &Definitions{
		Vhosts: []VhostDef{{Name: "/"}},
		Exchanges: []ExchangeDef{
			{Name: "media", Vhost: "/", Type: "topic", Durable: true, Arguments: map[string]any{}},
			{Name: "media.dlx", Vhost: "/", Type: "topic", Durable: true, Arguments: map[string]any{}},
		},
		Queues: []QueueDef{
			{Name: "media.preview", Vhost: "/", Durable: true, Arguments: map[string]any{
				"x-dead-letter-exchange":    "media.dlx",
				"x-dead-letter-routing-key": "media.preview.dlq",
			}},
			{Name: "media.preview.dlq", Vhost: "/", Durable: true, Arguments: map[string]any{}},
			{Name: "media.preview.retry", Vhost: "/", Durable: true, Arguments: map[string]any{
				"x-message-ttl":             60000,
				"x-dead-letter-exchange":    "media",
				"x-dead-letter-routing-key": "media.preview",
			}},
		},
		Bindings: []BindingDef{
			{Source: "media", Vhost: "/", Destination: "media.preview", DestinationType: "queue", RoutingKey: "media.preview", Arguments: map[string]any{}},
			{Source: "media.dlx", Vhost: "/", Destination: "media.preview.dlq", DestinationType: "queue", RoutingKey: "media.preview.dlq", Arguments: map[string]any{}},
		},
	}

	data, err := defs.JSON()
	require.NoError(t, err)
	assert.True(t, json.Valid(data))

	// Round-trip: unmarshal back and compare
	var got Definitions
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, defs.Vhosts, got.Vhosts)
	assert.Equal(t, defs.Exchanges, got.Exchanges)
	assert.Len(t, got.Queues, 3)
	assert.Len(t, got.Bindings, 2)
}

func TestDefinitions_JSON_OmitsEmptyUsers(t *testing.T) {
	t.Parallel()

	defs := &Definitions{
		Vhosts:    []VhostDef{{Name: "/"}},
		Exchanges: []ExchangeDef{},
		Queues:    []QueueDef{},
		Bindings:  []BindingDef{},
	}

	data, err := defs.JSON()
	require.NoError(t, err)

	// users and permissions should be omitted when empty
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.NotContains(t, raw, "users")
	assert.NotContains(t, raw, "permissions")
}
