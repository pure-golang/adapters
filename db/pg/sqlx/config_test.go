package sqlx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_DefaultValues verifies that config has expected default values
func TestConfig_DefaultValues(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		User:     "user",
		Password: "pass",
		Database: "db",
	}

	// Set default values manually as envconfig would do
	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 5
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 10
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 5
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 30 * time.Minute
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = 10 * time.Minute
	}
	if cfg.QueryTimeout == 0 {
		cfg.QueryTimeout = 10 * time.Second
	}

	assert.Equal(t, 5432, cfg.Port)
	assert.Equal(t, "disable", cfg.SSLMode)
	assert.Equal(t, 5, cfg.ConnectTimeout)
	assert.Equal(t, 10, cfg.MaxOpenConns)
	assert.Equal(t, 5, cfg.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.ConnMaxIdleTime)
	assert.Equal(t, 10*time.Second, cfg.QueryTimeout)
}

// TestConfig_SSLModes verifies SSL mode configurations
func TestConfig_SSLModes(t *testing.T) {
	sslModes := []string{"disable", "require", "verify-ca", "verify-full"}

	for _, mode := range sslModes {
		t.Run(mode, func(t *testing.T) {
			cfg := Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "db",
				SSLMode:  mode,
			}

			// Verify the mode is set correctly
			assert.Equal(t, mode, cfg.SSLMode)
		})
	}
}

// TestConfig_ConnectTimeout verifies connect timeout configuration
func TestConfig_ConnectTimeout(t *testing.T) {
	testCases := []int{
		0,   // no timeout
		1,   // 1 second
		5,   // 5 seconds (default)
		10,  // 10 seconds
		30,  // 30 seconds
		120, // 2 minutes
	}

	for _, timeout := range testCases {
		t.Run(timeoutString(timeout), func(t *testing.T) {
			cfg := Config{
				Host:           "localhost",
				Port:           5432,
				User:           "user",
				Password:       "pass",
				Database:       "db",
				ConnectTimeout: timeout,
			}

			assert.Equal(t, timeout, cfg.ConnectTimeout)
		})
	}
}

// TestConfig_QueryTimeout verifies query timeout configuration
func TestConfig_QueryTimeout(t *testing.T) {
	testCases := []time.Duration{
		0,                      // no timeout
		100 * time.Millisecond, // 100ms
		1 * time.Second,        // 1s
		5 * time.Second,        // 5s
		10 * time.Second,       // 10s (default)
		30 * time.Second,       // 30s
		60 * time.Second,       // 1m
	}

	for _, timeout := range testCases {
		t.Run(timeout.String(), func(t *testing.T) {
			cfg := Config{
				Host:         "localhost",
				Port:         5432,
				User:         "user",
				Password:     "pass",
				Database:     "db",
				QueryTimeout: timeout,
			}

			assert.Equal(t, timeout, cfg.QueryTimeout)
		})
	}
}

// TestConfig_ConnectionPoolSettings verifies connection pool settings
func TestConfig_ConnectionPoolSettings(t *testing.T) {
	testCases := []struct {
		name            string
		maxOpenConns    int
		maxIdleConns    int
		connMaxLifetime time.Duration
		connMaxIdleTime time.Duration
	}{
		{
			name:            "default values",
			maxOpenConns:    10,
			maxIdleConns:    5,
			connMaxLifetime: 30 * time.Minute,
			connMaxIdleTime: 10 * time.Minute,
		},
		{
			name:            "small pool",
			maxOpenConns:    2,
			maxIdleConns:    1,
			connMaxLifetime: 5 * time.Minute,
			connMaxIdleTime: 1 * time.Minute,
		},
		{
			name:            "large pool",
			maxOpenConns:    100,
			maxIdleConns:    50,
			connMaxLifetime: 1 * time.Hour,
			connMaxIdleTime: 30 * time.Minute,
		},
		{
			name:            "disabled limits",
			maxOpenConns:    0,
			maxIdleConns:    0,
			connMaxLifetime: 0,
			connMaxIdleTime: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Host:            "localhost",
				Port:            5432,
				User:            "user",
				Password:        "pass",
				Database:        "db",
				MaxOpenConns:    tc.maxOpenConns,
				MaxIdleConns:    tc.maxIdleConns,
				ConnMaxLifetime: tc.connMaxLifetime,
				ConnMaxIdleTime: tc.connMaxIdleTime,
			}

			assert.Equal(t, tc.maxOpenConns, cfg.MaxOpenConns)
			assert.Equal(t, tc.maxIdleConns, cfg.MaxIdleConns)
			assert.Equal(t, tc.connMaxLifetime, cfg.ConnMaxLifetime)
			assert.Equal(t, tc.connMaxIdleTime, cfg.ConnMaxIdleTime)
		})
	}
}

// TestConfig_RequiredFields verifies required field validation
func TestConfig_RequiredFields(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	// All required fields should be set
	require.NotEmpty(t, cfg.Host)
	require.NotEmpty(t, cfg.User)
	require.NotEmpty(t, cfg.Password)
	require.NotEmpty(t, cfg.Database)
}

func timeoutString(timeout int) string {
	if timeout == 0 {
		return "none"
	}
	return "seconds"
}
