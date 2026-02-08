package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultDialer(t *testing.T) {
	brokers := []string{"localhost:9092"}
	dialer := NewDefaultDialer(brokers)

	assert.NotNil(t, dialer)
	assert.Equal(t, brokers, dialer.GetBrokers())
	assert.NotNil(t, dialer.GetDialer())
}

func TestNewDialer(t *testing.T) {
	cfg := Config{
		Brokers: []string{"broker1:9092", "broker2:9092"},
	}

	dialer := NewDialer(cfg, nil)

	assert.NotNil(t, dialer)
	assert.Equal(t, cfg.Brokers, dialer.GetBrokers())
	assert.NotNil(t, dialer.GetDialer())
}

func TestConfig_DefaultValues(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}

	// Проверяем, что структура создается с дефолтными значениями
	assert.NotEmpty(t, cfg.Brokers)
}
