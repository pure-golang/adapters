package queue

import (
	"testing"
	"time"

	"github.com/pure-golang/adapters/queue/encoders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessage_EncodeValue_NilBody tests EncodeValue with nil body.
func TestMessage_EncodeValue_NilBody(t *testing.T) {
	msg := &Message{
		Topic:   "test-topic",
		Headers: map[string]string{"key": "value"},
		Body:    nil,
		TTL:     time.Minute,
	}

	encoder := encoders.JSON{}

	result, err := msg.EncodeValue(encoder)

	require.NoError(t, err)
	assert.Nil(t, result, "EncodeValue with nil body should return nil bytes")
}

// TestMessage_EncodeValue_ValidBody_JSON tests EncodeValue with valid body using JSON encoder.
func TestMessage_EncodeValue_ValidBody_JSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	msg := &Message{
		Topic:   "test-topic",
		Headers: map[string]string{"key": "value"},
		Body: TestStruct{
			Name:  "test",
			Value: 42,
		},
		TTL: time.Minute,
	}

	encoder := encoders.JSON{}

	result, err := msg.EncodeValue(encoder)

	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"test","value":42}`, string(result))
	assert.Equal(t, "application/json", encoder.ContentType())
}

// TestMessage_EncodeValue_ValidBody_Text tests EncodeValue with valid body using Text encoder.
func TestMessage_EncodeValue_ValidBody_Text(t *testing.T) {
	msg := &Message{
		Topic:   "test-topic",
		Headers: map[string]string{"key": "value"},
		Body:    "hello world",
		TTL:     time.Minute,
	}

	encoder := encoders.Text{}

	result, err := msg.EncodeValue(encoder)

	require.NoError(t, err)
	assert.Equal(t, []byte("hello world"), result)
	assert.Equal(t, "text/plain", encoder.ContentType())
}

// TestMessage_EncodeValue_ValidBody_ByteArray tests EncodeValue with byte array body using Text encoder.
func TestMessage_EncodeValue_ValidBody_ByteArray(t *testing.T) {
	msg := &Message{
		Topic:   "test-topic",
		Headers: map[string]string{"key": "value"},
		Body:    []byte("binary data"),
		TTL:     time.Minute,
	}

	encoder := encoders.Text{}

	result, err := msg.EncodeValue(encoder)

	require.NoError(t, err)
	assert.Equal(t, []byte("binary data"), result)
}

// TestMessage_EncodeValue_EncoderError tests EncodeValue when encoder returns an error.
func TestMessage_EncodeValue_EncoderError(t *testing.T) {
	// Use Text encoder with an unsupported type to trigger an error
	msg := &Message{
		Topic:   "test-topic",
		Headers: map[string]string{"key": "value"},
		Body:    123, // int is not supported by Text encoder
		TTL:     time.Minute,
	}

	encoder := encoders.Text{}

	result, err := msg.EncodeValue(encoder)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknown type")
}

// TestMessage_EncodeValue_ComplexObject tests EncodeValue with a complex nested object.
func TestMessage_EncodeValue_ComplexObject(t *testing.T) {
	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Person struct {
		Name    string   `json:"name"`
		Age     int      `json:"age"`
		Address Address  `json:"address"`
		Tags    []string `json:"tags"`
	}

	msg := &Message{
		Topic: "person-topic",
		Body: Person{
			Name: "John Doe",
			Age:  30,
			Address: Address{
				City:    "Edinburgh",
				Country: "Scotland",
			},
			Tags: []string{"developer", "golang"},
		},
		TTL: time.Hour,
	}

	encoder := encoders.JSON{}

	result, err := msg.EncodeValue(encoder)

	require.NoError(t, err)
	assert.JSONEq(t, `{
		"name": "John Doe",
		"age": 30,
		"address": {
			"city": "Edinburgh",
			"country": "Scotland"
		},
		"tags": ["developer", "golang"]
	}`, string(result))
}

// TestMessage_EncodeValue_JSONEncoderError tests EncodeValue with JSON encoder error.
func TestMessage_EncodeValue_JSONEncoderError(t *testing.T) {
	// Use a channel which is not supported by json.Marshal
	msg := &Message{
		Topic: "test-topic",
		Body:  make(chan int),
		TTL:   time.Minute,
	}

	encoder := encoders.JSON{}

	result, err := msg.EncodeValue(encoder)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "marshal")
}

// TestMessage_Fields tests Message struct fields.
func TestMessage_Fields(t *testing.T) {
	headers := map[string]string{"content-type": "application/json", "x-custom": "value"}
	body := "test body"
	ttl := 30 * time.Second

	msg := &Message{
		Topic:   "test-topic",
		Headers: headers,
		Body:    body,
		TTL:     ttl,
	}

	assert.Equal(t, "test-topic", msg.Topic)
	assert.Equal(t, headers, msg.Headers)
	assert.Equal(t, body, msg.Body)
	assert.Equal(t, ttl, msg.TTL)
}

// TestMessage_EncodeValue_EmptyMessage tests EncodeValue on message with empty values.
func TestMessage_EncodeValue_EmptyMessage(t *testing.T) {
	msg := &Message{}

	encoder := encoders.JSON{}

	result, err := msg.EncodeValue(encoder)

	require.NoError(t, err)
	assert.Nil(t, result, "EncodeValue with nil body (empty message) should return nil")
}

// TestMessage_EncodeValue_StringWithJSON tests EncodeValue with string body and JSON encoder.
func TestMessage_EncodeValue_StringWithJSON(t *testing.T) {
	msg := &Message{
		Topic: "test-topic",
		Body:  "plain string",
		TTL:   time.Minute,
	}

	encoder := encoders.JSON{}

	result, err := msg.EncodeValue(encoder)

	require.NoError(t, err)
	assert.JSONEq(t, `"plain string"`, string(result))
}

// TestMessage_EncodeValue_WithHeaders tests that headers are preserved during encoding.
func TestMessage_EncodeValue_WithHeaders(t *testing.T) {
	headers := map[string]string{
		"content-type": "application/json",
		"x-trace-id":   "12345",
	}

	msg := &Message{
		Topic:   "test-topic",
		Headers: headers,
		Body:    "data",
		TTL:     time.Minute,
	}

	encoder := encoders.JSON{}

	// EncodeValue should not modify headers
	_ = msg.Headers
	result, err := msg.EncodeValue(encoder)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Headers should remain unchanged
	assert.Equal(t, "application/json", msg.Headers["content-type"])
	assert.Equal(t, "12345", msg.Headers["x-trace-id"])
}

// TestDelivery_Fields tests Delivery struct fields.
func TestDelivery_Fields(t *testing.T) {
	headers := map[string]string{"content-type": "application/json"}
	body := []byte(`{"test": "data"}`)

	delivery := Delivery{
		Headers: headers,
		Body:    body,
	}

	assert.Equal(t, headers, delivery.Headers)
	assert.Equal(t, body, delivery.Body)
}

// TestDelivery_EmptyFields tests Delivery with empty fields.
func TestDelivery_EmptyFields(t *testing.T) {
	delivery := Delivery{
		Headers: map[string]string{},
		Body:    []byte{},
	}

	assert.NotNil(t, delivery.Headers)
	assert.Empty(t, delivery.Headers)
	assert.Empty(t, delivery.Body)
}

// TestDelivery_NilFields tests Delivery with nil fields.
func TestDelivery_NilFields(t *testing.T) {
	delivery := Delivery{
		Headers: nil,
		Body:    nil,
	}

	assert.Nil(t, delivery.Headers)
	assert.Nil(t, delivery.Body)
}
