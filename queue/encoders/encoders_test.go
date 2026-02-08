package encoders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSON_Encode_ValidObject tests encoding with a valid object.
func TestJSON_Encode_ValidObject(t *testing.T) {
	encoder := JSON{}

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	input := TestStruct{
		Name:  "test",
		Value: 42,
	}

	result, err := encoder.Encode(input)

	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"test","value":42}`, string(result))
}

// TestJSON_Encode_Nil tests encoding with nil.
func TestJSON_Encode_Nil(t *testing.T) {
	encoder := JSON{}

	result, err := encoder.Encode(nil)

	require.NoError(t, err)
	assert.Equal(t, []byte("null"), result)
}

// TestJSON_Encode_ComplexObject tests encoding with a complex nested object.
func TestJSON_Encode_ComplexObject(t *testing.T) {
	encoder := JSON{}

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

	input := Person{
		Name: "John Doe",
		Age:  30,
		Address: Address{
			City:    "Edinburgh",
			Country: "Scotland",
		},
		Tags: []string{"developer", "golang"},
	}

	result, err := encoder.Encode(input)

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

// TestJSON_Encode_WithInvalidType tests error handling when encoding an unmarshalable type.
func TestJSON_Encode_WithInvalidType(t *testing.T) {
	encoder := JSON{}

	// Using a channel which is not supported by json.Marshal
	invalidInput := make(chan int)

	result, err := encoder.Encode(invalidInput)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "marshal chan int:")
}

// TestJSON_ContentType tests the ContentType method.
func TestJSON_ContentType(t *testing.T) {
	encoder := JSON{}

	assert.Equal(t, "application/json", encoder.ContentType())
}

// TestText_Encode_ByteSlice tests encoding with a byte slice.
func TestText_Encode_ByteSlice(t *testing.T) {
	encoder := Text{}

	input := []byte("hello world")

	result, err := encoder.Encode(input)

	require.NoError(t, err)
	assert.Equal(t, input, result)
	assert.Equal(t, []byte("hello world"), result)
}

// TestText_Encode_ByteSlice tests encoding with an empty byte slice.
func TestText_Encode_EmptyByteSlice(t *testing.T) {
	encoder := Text{}

	input := []byte{}

	result, err := encoder.Encode(input)

	require.NoError(t, err)
	assert.Equal(t, input, result)
	assert.Equal(t, []byte{}, result)
}

// TestText_Encode_String tests encoding with a string.
func TestText_Encode_String(t *testing.T) {
	encoder := Text{}

	input := "hello world"

	result, err := encoder.Encode(input)

	require.NoError(t, err)
	assert.Equal(t, []byte(input), result)
	assert.Equal(t, []byte("hello world"), result)
}

// TestText_Encode_EmptyString tests encoding with an empty string.
func TestText_Encode_EmptyString(t *testing.T) {
	encoder := Text{}

	input := ""

	result, err := encoder.Encode(input)

	require.NoError(t, err)
	assert.Equal(t, []byte{}, result)
}

// TestText_Encode_UnsupportedType tests error handling for unsupported types.
func TestText_Encode_UnsupportedType(t *testing.T) {
	encoder := Text{}

	type UnsupportedStruct struct {
		Field string
	}

	input := UnsupportedStruct{Field: "test"}

	result, err := encoder.Encode(input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknown type encoders.UnsupportedStruct to encode with encoders.Text")
}

// TestText_Encode_Int tests error handling for integer type.
func TestText_Encode_Int(t *testing.T) {
	encoder := Text{}

	input := 42

	result, err := encoder.Encode(input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknown type int to encode with")
}

// TestText_ContentType tests the ContentType method.
func TestText_ContentType(t *testing.T) {
	encoder := Text{}

	assert.Equal(t, "text/plain", encoder.ContentType())
}
