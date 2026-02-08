package rabbitmq

import (
	"time"
)

// Default values
const (
	DefaultRetryInterval             = time.Millisecond * 100
	DefaultConnIntervalMultiplicator = 2
	DefaultMaxInterval               = time.Hour * 2
)

// NewDefaultMaxInterval returns MaxInterval with default values.
func NewDefaultMaxInterval() *MaxInterval {
	return &MaxInterval{
		base:          time.Millisecond * 100,
		max:           time.Hour * 2,
		multiplicator: 2,
	}
}

// NewMaxInterval creates a new  MaxInterval.
func NewMaxInterval(
	baseInterval, maxInterval time.Duration,
	intervalMultiplicator int,
) *MaxInterval {
	if baseInterval == 0 {
		panic("interval should not be 0")
	}

	if intervalMultiplicator == 0 {
		panic("multiplicator should not be 0")
	}

	if maxInterval == 0 {
		panic("max interval should not be 0")
	}

	return &MaxInterval{
		base:          baseInterval,
		max:           maxInterval,
		multiplicator: intervalMultiplicator,
	}
}

// MaxInterval is RetryPolicy.
type MaxInterval struct {
	base          time.Duration
	max           time.Duration
	multiplicator int
}

// TryNum for use in for loop. tryNum int is number of iteration.
func (interval *MaxInterval) TryNum(tryNum int) (time.Duration, bool) {
	retryInterval := interval.base * time.Duration(tryNum*interval.multiplicator)
	if retryInterval > interval.max {
		return 0, true
	}

	return retryInterval, false
}
