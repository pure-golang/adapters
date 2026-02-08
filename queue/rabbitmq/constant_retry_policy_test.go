package rabbitmq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConstantInterval(t *testing.T) {
	interval := time.Second * 5
	policy := NewConstantInterval(interval)

	require.NotNil(t, policy)
	assert.Equal(t, interval, policy.interval)
}

func TestConstantInterval_TryNum(t *testing.T) {
	t.Run("returns constant duration", func(t *testing.T) {
		interval := time.Second * 10
		policy := NewConstantInterval(interval)

		// Test multiple try numbers - all should return the same duration
		for i := 0; i < 10; i++ {
			duration, stop := policy.TryNum(i)
			assert.Equal(t, interval, duration, "TryNum(%d) should return constant interval", i)
			assert.False(t, stop, "TryNum(%d) should never stop", i)
		}
	})

	t.Run("never stops", func(t *testing.T) {
		policy := NewConstantInterval(time.Millisecond)

		// Even with very high try numbers, it should never stop
		for _, tryNum := range []int{0, 1, 10, 100, 1000} {
			_, stop := policy.TryNum(tryNum)
			assert.False(t, stop, "ConstantInterval should never stop, but did at tryNum=%d", tryNum)
		}
	})

	t.Run("handles zero try number", func(t *testing.T) {
		interval := time.Second
		policy := NewConstantInterval(interval)

		duration, stop := policy.TryNum(0)
		assert.Equal(t, interval, duration)
		assert.False(t, stop)
	})
}
