package rabbitmq

import (
	"time"
)

type ConstantInterval struct {
	interval time.Duration
}

func NewConstantInterval(interval time.Duration) *ConstantInterval {
	return &ConstantInterval{interval: interval}
}

func (c *ConstantInterval) TryNum(int) (duration time.Duration, stop bool) {
	return c.interval, false
}
