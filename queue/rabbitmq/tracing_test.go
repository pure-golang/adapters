package rabbitmq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestTableCarrier_Get(t *testing.T) {
	t.Run("existing key", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{
			"key1": "value1",
			"key2": 123,
		})

		result := carrier.Get("key1")
		assert.Equal(t, "value1", result)
	})

	t.Run("non-existing key", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{
			"key1": "value1",
		})

		result := carrier.Get("nonexistent")
		assert.Equal(t, "<nil>", result) // fmt.Sprintf with nil returns "<nil>"
	})

	t.Run("with integer value", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{
			"count": 42,
		})

		result := carrier.Get("count")
		assert.Equal(t, "42", result)
	})

	t.Run("with boolean value", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{
			"active": true,
		})

		result := carrier.Get("active")
		assert.Equal(t, "true", result)
	})
}

func TestTableCarrier_Set(t *testing.T) {
	t.Run("set new key", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{})

		carrier.Set("new-key", "new-value")

		assert.Equal(t, "new-value", carrier["new-key"])
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{
			"key": "old-value",
		})

		carrier.Set("key", "new-value")

		assert.Equal(t, "new-value", carrier["key"])
	})

	t.Run("set multiple keys", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{})

		carrier.Set("key1", "value1")
		carrier.Set("key2", "value2")

		assert.Equal(t, "value1", carrier["key1"])
		assert.Equal(t, "value2", carrier["key2"])
	})
}

func TestTableCarrier_Keys(t *testing.T) {
	t.Run("empty carrier", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{})

		keys := carrier.Keys()

		assert.NotNil(t, keys)
		assert.Empty(t, keys)
	})

	t.Run("single key", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{
			"key1": "value1",
		})

		keys := carrier.Keys()

		assert.Len(t, keys, 1)
		assert.Contains(t, keys, "key1")
	})

	t.Run("multiple keys", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		})

		keys := carrier.Keys()

		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "key1")
		assert.Contains(t, keys, "key2")
		assert.Contains(t, keys, "key3")
	})

	t.Run("keys are unique", func(t *testing.T) {
		carrier := tableCarrier(amqp.Table{
			"a": "1",
			"b": "2",
		})

		keys := carrier.Keys()

		// Convert to map to check uniqueness
		keyMap := make(map[string]bool)
		for _, k := range keys {
			keyMap[k] = true
		}
		assert.Len(t, keyMap, len(keys), "All keys should be unique")
	})
}

func TestTableCarrier_CombinedOperations(t *testing.T) {
	carrier := tableCarrier(amqp.Table{
		"existing": "value",
	})

	// Get existing key
	assert.Equal(t, "value", carrier.Get("existing"))

	// Get non-existing key
	assert.Equal(t, "<nil>", carrier.Get("nonexistent"))

	// Set new key
	carrier.Set("new-key", "new-value")
	assert.Equal(t, "new-value", carrier.Get("new-key"))

	// Get all keys
	keys := carrier.Keys()
	assert.Len(t, keys, 2)
}
