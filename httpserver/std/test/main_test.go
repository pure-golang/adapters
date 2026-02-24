package std_test

import (
	"git.korputeam.ru/newbackend/adapters/logger"
)

func init() {
	logger.InitDefault(logger.Config{
		Provider: logger.ProviderNoop,
		Level:    logger.INFO,
	})
}
