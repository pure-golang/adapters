package pgx

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_URL_DefaultPort(t *testing.T) {
	cfg := Config{
		User:     "testuser",
		Password: "testpass",
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
	}

	u := cfg.URL()

	require.NotNil(t, u)
	assert.Equal(t, "postgres", u.Scheme)
	assert.Equal(t, "localhost", u.Host)
	assert.Equal(t, "testdb", u.Path)

	// Verify user info
	user := u.User
	require.NotNil(t, user)
	assert.Equal(t, "testuser", user.Username())
	password, ok := user.Password()
	assert.True(t, ok)
	assert.Equal(t, "testpass", password)

	// Verify query params
	q := u.Query()
	assert.Equal(t, "utc", q.Get("timezone"))
	assert.Equal(t, "disable", q.Get("sslmode"))
}

func TestConfig_URL_CustomPort(t *testing.T) {
	cfg := Config{
		User:     "testuser",
		Password: "testpass",
		Host:     "db.example.com",
		Port:     5433,
		Name:     "testdb",
	}

	u := cfg.URL()

	require.NotNil(t, u)
	assert.Equal(t, "db.example.com:5433", u.Host)
	assert.Equal(t, "postgres", u.Scheme)

	// Verify user info
	user := u.User
	require.NotNil(t, user)
	assert.Equal(t, "testuser", user.Username())

	// Verify query params
	q := u.Query()
	assert.Equal(t, "utc", q.Get("timezone"))
	assert.Equal(t, "disable", q.Get("sslmode"))
}

func TestConfig_URL_WithSSLCert(t *testing.T) {
	cfg := Config{
		User:     "testuser",
		Password: "testpass",
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		CertPath: "/path/to/cert.pem",
	}

	u := cfg.URL()

	require.NotNil(t, u)
	assert.Equal(t, "postgres", u.Scheme)

	// Verify query params with SSL
	q := u.Query()
	assert.Equal(t, "utc", q.Get("timezone"))
	assert.Equal(t, "verify-full", q.Get("sslmode"))
	assert.Equal(t, "/path/to/cert.pem", q.Get("sslrootcert"))
}

func TestConfig_URL_WithoutSSLCert(t *testing.T) {
	cfg := Config{
		User:     "testuser",
		Password: "testpass",
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		CertPath: "",
	}

	u := cfg.URL()

	require.NotNil(t, u)

	// Verify query params without SSL
	q := u.Query()
	assert.Equal(t, "utc", q.Get("timezone"))
	assert.Equal(t, "disable", q.Get("sslmode"))
	assert.Equal(t, "", q.Get("sslrootcert"))
}

func TestConfig_URL_FullURLBuilding(t *testing.T) {
	cfg := Config{
		User:     "dbuser",
		Password: "dbpassword",
		Host:     "db.example.com",
		Port:     6432,
		Name:     "production",
		CertPath: "/etc/ssl/certs/postgres.crt",
	}

	u := cfg.URL()

	require.NotNil(t, u)

	// Full URL string representation
	expected := "postgres://dbuser:dbpassword@db.example.com:6432/production?sslmode=verify-full&sslrootcert=%2Fetc%2Fssl%2Fcerts%2Fpostgres.crt&timezone=utc"
	assert.Equal(t, expected, u.String())

	// Verify all components
	assert.Equal(t, "postgres", u.Scheme)
	assert.Equal(t, "db.example.com:6432", u.Host)
	assert.Equal(t, "production", u.Path)

	user := u.User
	require.NotNil(t, user)
	assert.Equal(t, "dbuser", user.Username())
	password, ok := user.Password()
	assert.True(t, ok)
	assert.Equal(t, "dbpassword", password)

	// Verify all query params
	q := u.Query()
	assert.Equal(t, "utc", q.Get("timezone"))
	assert.Equal(t, "verify-full", q.Get("sslmode"))
	assert.Equal(t, "/etc/ssl/certs/postgres.crt", q.Get("sslrootcert"))
}

func TestConfig_URL_SpecialCharactersInPassword(t *testing.T) {
	cfg := Config{
		User:     "user@example.com",
		Password: "p@ss:w0rd/123",
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
	}

	u := cfg.URL()

	require.NotNil(t, u)

	// URL encoding should be handled by url.UserPassword
	user := u.User
	require.NotNil(t, user)
	assert.Equal(t, "user@example.com", user.Username())
	password, ok := user.Password()
	assert.True(t, ok)
	assert.Equal(t, "p@ss:w0rd/123", password)

	// When converted to string, special chars should be encoded
	parsed, err := url.Parse(u.String())
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", parsed.User.Username())
	parsedPassword, ok := parsed.User.Password()
	assert.True(t, ok)
	assert.Equal(t, "p@ss:w0rd/123", parsedPassword)
}

func TestConfig_URL_EmptyCertPath(t *testing.T) {
	cfg := Config{
		User:     "testuser",
		Password: "testpass",
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		CertPath: "",
	}

	u := cfg.URL()

	require.NotNil(t, u)

	q := u.Query()
	assert.Equal(t, "disable", q.Get("sslmode"))
	assert.Empty(t, q.Get("sslrootcert"))
}

func TestConfig_URL_NonDefaultPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected string
	}{
		{"port zero", 0, "localhost:0"},
		{"port one", 1, "localhost:1"},
		{"port 8080", 8080, "localhost:8080"},
		{"default port 5432", 5432, "localhost"},
		{"port 6432", 6432, "localhost:6432"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				User:     "testuser",
				Password: "testpass",
				Host:     "localhost",
				Port:     tt.port,
				Name:     "testdb",
			}

			u := cfg.URL()
			require.NotNil(t, u)

			if tt.port == 5432 {
				assert.Equal(t, "localhost", u.Host)
			} else {
				assert.Equal(t, tt.expected, u.Host)
			}
		})
	}
}
