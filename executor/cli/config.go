package cli

import (
	"github.com/kelseyhightower/envconfig"
)

// Config содержит конфигурацию CLI executor
type Config struct {
	// Command - имя исполняемой команды (например, "ffmpeg", "gsutil", "aws")
	// Загружается из переменной окружения CLI_COMMAND
	Command string `envconfig:"CLI_COMMAND" required:"true"`
}

// InitConfig загружает конфигурацию из переменных окружения
func InitConfig(cfg *Config) error {
	return envconfig.Process("", cfg)
}
