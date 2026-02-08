package env

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

const DefaultEnvFile = ".env"

func InitConfig(config any) error {
	// Try to load .env file, but don't fail if it doesn't exist
	// nolint:errcheck // .env file is optional, failure is acceptable
	_ = godotenv.Load(DefaultEnvFile)

	if err := envconfig.Process("", config); err != nil {
		return errors.Wrap(err, "failed to envconfig.Process")
	}

	return nil
}
