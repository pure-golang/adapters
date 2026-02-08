package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeadersCarrier(t *testing.T) {
	carrier := make(headersCarrier)

	// Тест Set и Get
	carrier.Set("key1", "value1")
	carrier.Set("key2", "value2")

	assert.Equal(t, "value1", carrier.Get("key1"))
	assert.Equal(t, "value2", carrier.Get("key2"))
	assert.Equal(t, "", carrier.Get("nonexistent"))

	// Тест Keys
	keys := carrier.Keys()
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
}

func TestHeadersCarrier_Empty(t *testing.T) {
	carrier := make(headersCarrier)

	// Тест на пустом carries
	assert.Equal(t, "", carrier.Get("key"))
	keys := carrier.Keys()
	assert.Len(t, keys, 0)
}
