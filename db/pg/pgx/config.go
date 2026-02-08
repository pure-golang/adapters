package pgx

import (
	"fmt"
	"net/url"
)

type Config struct {
	User            string `envconfig:"POSTGRES_USER" required:"true"`
	Password        string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	Host            string `envconfig:"POSTGRES_HOST" required:"true"`
	Port            int    `envconfig:"POSTGRES_PORT" default:"5432"`
	Name            string `envconfig:"POSTGRES_DB_NAME" required:"true"`
	CertPath        string `envconfig:"POSTGRES_SSL_CERT_PATH"`
	MaxOpenConns    int32  `envconfig:"POSTGRES_MAX_OPEN_CONNECTIONS" default:"20"`
	MaxConnLifeTime int32  `envconfig:"POSTGRES_MAX_CONNECTIONS_LIFETIME" default:"5"`
	MaxConnIdleTime int32  `envconfig:"POSTGRES_MAX_CONNECTIONS_IDLE_TIME" default:"5"`
	// TraceLogLevel  values: trace, debug, info, warn, error, none.
	// Set "error" or omit empty for production, "debug" for dev.
	TraceLogLevel string `envconfig:"POSTGRES_TRACE_LOG_LEVEL" default:"error"`
}

// URL returns database config in URL presentation
func (c *Config) URL() *url.URL {
	q := url.Values{"timezone": []string{"utc"}}
	if c.CertPath != "" {
		q.Set("sslmode", "verify-full")
		q.Set("sslrootcert", c.CertPath)
	} else {
		q.Set("sslmode", "disable")
	}

	// Build host:port string
	host := c.Host
	if c.Port != 5432 {
		host = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}

	return &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     host,
		Path:     c.Name,
		RawQuery: q.Encode(),
	}
}
