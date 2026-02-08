package sqlx

import "time"

// Config содержит параметры подключения к PostgreSQL
type Config struct {
	Host            string        `envconfig:"POSTGRES_HOST" required:"true"`
	Port            int           `envconfig:"POSTGRES_PORT" default:"5432"`
	User            string        `envconfig:"POSTGRES_USER" required:"true"`
	Password        string        `envconfig:"POSTGRES_PASSWORD" required:"true"`
	Database        string        `envconfig:"POSTGRES_DB" required:"true"`
	SSLMode         string        `envconfig:"POSTGRES_SSLMODE" default:"disable"`
	ConnectTimeout  int           `envconfig:"POSTGRES_CONNECT_TIMEOUT" default:"5"`
	MaxOpenConns    int           `envconfig:"POSTGRES_MAX_OPEN_CONNS" default:"10"`
	MaxIdleConns    int           `envconfig:"POSTGRES_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"POSTGRES_CONN_MAX_LIFETIME" default:"30m"`
	ConnMaxIdleTime time.Duration `envconfig:"POSTGRES_CONN_MAX_IDLE_TIME" default:"10m"`
	QueryTimeout    time.Duration `envconfig:"POSTGRES_QUERY_TIMEOUT" default:"10s"`
}
