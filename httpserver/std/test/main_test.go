package std_test

import (
	"github.com/pure-golang/adapters/logger"
)

func init() {
	logger.InitDefault(logger.Config{
		Provider: logger.ProviderNoop,
		Level:    logger.INFO,
	})
}
