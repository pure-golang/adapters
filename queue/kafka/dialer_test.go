package kafka

import (
	"github.com/stretchr/testify/assert"
)

func (s *KafkaSuite) TestDialer_Connect() {
	cfg := Config{
		Brokers: s.brokers,
	}
	dialer := NewDialer(cfg, nil)
	assert.NotNil(s.T(), dialer)
	assert.NotNil(s.T(), dialer.GetDialer())
	assert.Equal(s.T(), s.brokers, dialer.GetBrokers())

	err := dialer.Close()
	assert.NoError(s.T(), err)
}

func (s *KafkaSuite) TestDialer_Close() {
	cfg := Config{
		Brokers: s.brokers,
	}
	dialer := NewDialer(cfg, nil)

	err := dialer.Close()
	assert.NoError(s.T(), err)

	// Повторный Close не должен вызывать ошибку
	err = dialer.Close()
	assert.NoError(s.T(), err)
}
