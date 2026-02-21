package cli

import (
	"github.com/kelseyhightower/envconfig"
)

// SSHConfig содержит параметры для удалённого выполнения команд через SSH
type SSHConfig struct {
	Host     string `envconfig:"SSH_HOST"`
	Port     int    `envconfig:"SSH_PORT" default:"22"`
	User     string `envconfig:"SSH_USER"`
	KeyPath  string `envconfig:"SSH_KEY_PATH"`
	Password string `envconfig:"SSH_PASSWORD"`
}

// Config содержит конфигурацию CLI executor
type Config struct {
	// Command - имя исполняемой команды (например, "ffmpeg", "gsutil", "aws")
	Command string

	// SSH - параметры для удалённого выполнения команд
	SSH SSHConfig
}

// InitConfig загружает конфигурацию из переменных окружения
func InitConfig(cfg *Config) error {
	return envconfig.Process("", cfg)
}
