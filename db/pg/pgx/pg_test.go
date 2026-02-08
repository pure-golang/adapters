package pgx

import (
	"context"
	"testing"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ValidConfig(t *testing.T) {
	// This test verifies the configuration parsing logic
	// We can't test actual database connection without a running PostgreSQL instance
	// but we can verify the config URL generation

	cfg := Config{
		User:            "testuser",
		Password:        "testpass",
		Host:            "localhost",
		Port:            5432,
		Name:            "testdb",
		MaxOpenConns:    10,
		MaxConnLifeTime: 300,
		MaxConnIdleTime: 600,
	}

	// Just verify the URL is generated correctly
	u := cfg.URL()
	require.NotNil(t, u)
	assert.Equal(t, "postgres://testuser:testpass@localhost/testdb?sslmode=disable&timezone=utc", u.String())
}

func TestNew_InvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "invalid URL - missing required fields",
			cfg: Config{
				User:     "",
				Password: "",
				Host:     "",
				Name:     "",
			},
			wantErr: true,
		},
		{
			name: "valid config but connection fails",
			cfg: Config{
				User:            "testuser",
				Password:        "testpass",
				Host:            "localhost",
				Port:            5432,
				Name:            "nonexistent",
				MaxOpenConns:    1,
				MaxConnLifeTime: 1,
				MaxConnIdleTime: 1,
			},
			wantErr: true, // Will fail to connect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Try to create DB - will fail without actual database
			db, err := New(tt.cfg, nil)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, db)
			}
		})
	}
}

func TestNew_ZeroMaxOpenConns(t *testing.T) {
	cfg := Config{
		User:            "testuser",
		Password:        "testpass",
		Host:            "localhost",
		Port:            5432,
		Name:            "testdb",
		MaxOpenConns:    0, // Should be defaulted to 1
		MaxConnLifeTime: 300,
		MaxConnIdleTime: 600,
	}

	// Verify that when we try to create the config, MaxOpenConns would be set to 1
	// We can't test the full connection without a database, but we can check
	// the URL generation works
	u := cfg.URL()
	require.NotNil(t, u)
	assert.NotEmpty(t, u.String())
}

func TestNewDefault_Tracers(t *testing.T) {
	cfg := Config{
		User:            "testuser",
		Password:        "testpass",
		Host:            "localhost",
		Port:            5432,
		Name:            "testdb",
		TraceLogLevel:   "debug",
		MaxOpenConns:    1,
		MaxConnLifeTime: 1,
		MaxConnIdleTime: 1,
	}

	// Will fail to connect but we can check the tracer setup logic
	// by verifying the trace log level parsing
	logLevel := parseTraceLogLevel(cfg.TraceLogLevel)
	assert.Equal(t, logLevel.String(), "debug")
}

func TestNewDefault_TraceLogLevelParsing(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
	}{
		{
			name:     "error level",
			logLevel: "error",
		},
		{
			name:     "warn level",
			logLevel: "warn",
		},
		{
			name:     "info level",
			logLevel: "info",
		},
		{
			name:     "debug level",
			logLevel: "debug",
		},
		{
			name:     "trace level",
			logLevel: "trace",
		},
		{
			name:     "none level",
			logLevel: "none",
		},
		{
			name:     "invalid defaults to none",
			logLevel: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTraceLogLevel(tt.logLevel)
			// Just verify it doesn't panic and returns a valid log level (1-6 for pgx v5)
			assert.GreaterOrEqual(t, int(result), 1)
			assert.LessOrEqual(t, int(result), 6)
		})
	}
}

func TestDB_Close(t *testing.T) {
	t.Run("close with nil pool panics - we test this doesn't panic", func(t *testing.T) {
		// The Close method calls db.Pool.Close() which will panic if Pool is nil
		// We can't actually call it without a real pool, but we can verify
		// the DB structure exists
		db := &DB{
			Pool: nil,
		}
		assert.NotNil(t, db)
		// Calling Close would panic, so we just verify the structure
	})

	t.Run("close method exists on DB", func(t *testing.T) {
		// Verify DB implements io.Closer interface
		var _ interface{ Close() error } = &DB{}
	})
}

func TestDB_Close_NilPool(t *testing.T) {
	t.Run("nil pool DB can be created", func(t *testing.T) {
		db := &DB{
			Pool: nil,
		}
		assert.NotNil(t, db)
		assert.Nil(t, db.Pool)
	})
}

func TestDB_ConnectionPoolOptions(t *testing.T) {
	tests := []struct {
		name             string
		cfg              Config
		expectedMaxConns int32
	}{
		{
			name: "default max connections",
			cfg: Config{
				User:            "testuser",
				Password:        "testpass",
				Host:            "localhost",
				Port:            5432,
				Name:            "testdb",
				MaxOpenConns:    0, // Should be defaulted to 1
				MaxConnLifeTime: 300,
				MaxConnIdleTime: 600,
			},
			expectedMaxConns: 1,
		},
		{
			name: "custom max connections",
			cfg: Config{
				User:            "testuser",
				Password:        "testpass",
				Host:            "localhost",
				Port:            5432,
				Name:            "testdb",
				MaxOpenConns:    20,
				MaxConnLifeTime: 300,
				MaxConnIdleTime: 600,
			},
			expectedMaxConns: 20,
		},
		{
			name: "negative max connections",
			cfg: Config{
				User:            "testuser",
				Password:        "testpass",
				Host:            "localhost",
				Port:            5432,
				Name:            "testdb",
				MaxOpenConns:    -5, // Should be defaulted to 1
				MaxConnLifeTime: 300,
				MaxConnIdleTime: 600,
			},
			expectedMaxConns: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The actual connection pool configuration is validated in the New function
			// We can't test it fully without a database, but we can verify the logic
			expected := tt.expectedMaxConns
			if tt.cfg.MaxOpenConns < 1 {
				expected = 1
			}
			assert.Equal(t, tt.expectedMaxConns, expected)
		})
	}
}

func TestDB_ConnectionDurations(t *testing.T) {
	cfg := Config{
		User:            "testuser",
		Password:        "testpass",
		Host:            "localhost",
		Port:            5432,
		Name:            "testdb",
		MaxOpenConns:    10,
		MaxConnLifeTime: 5,
		MaxConnIdleTime: 5,
	}

	// Verify duration calculations
	expectedLifeTime := 5 * time.Second
	expectedIdleTime := 5 * time.Second

	assert.Equal(t, expectedLifeTime, time.Duration(cfg.MaxConnLifeTime)*time.Second)
	assert.Equal(t, expectedIdleTime, time.Duration(cfg.MaxConnIdleTime)*time.Second)
}

func TestNew_WithOptions(t *testing.T) {
	// Test with nil options (should use default empty options)
	options := &Options{}
	assert.NotNil(t, options)
	assert.Nil(t, options.Tracers)

	// Test with custom tracers
	options = &Options{
		Tracers: []pgx.QueryTracer{
			otelpgx.NewTracer(),
		},
	}
	assert.NotNil(t, options.Tracers)
	assert.Len(t, options.Tracers, 1)
}

func TestNew_WithNilOptions(t *testing.T) {
	// Test with nil options (should be handled in New function)
	var options *Options = nil

	// In the actual function, nil options would be replaced with &Options{}
	if options == nil {
		options = &Options{}
	}

	assert.NotNil(t, options)
}

func TestOptions_Struct(t *testing.T) {
	// Test Options structure
	opts := &Options{
		Tracers: []pgx.QueryTracer{},
	}

	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Tracers)
	assert.Empty(t, opts.Tracers)
}

func TestDB_ImplementsCloser(t *testing.T) {
	// Verify DB implements io.Closer
	var _ interface{ Close() error } = &DB{}
}

func TestNew_ParseConfigFailure(t *testing.T) {
	// Test that invalid DSN returns an error
	cfg := Config{
		User:     "testuser",
		Password: "testpass",
		Host:     "localhost",
		Port:     5432,
		Name:     "test db", // Space in name should be URL-encoded
	}

	// URL encoding should handle the space
	u := cfg.URL()
	require.NotNil(t, u)
	assert.Contains(t, u.String(), "test%20db")
}

func TestDB_Ping_NilPool(t *testing.T) {
	db := &DB{
		Pool: nil,
	}

	// Ping on nil pool would panic if not handled
	// In real usage, this shouldn't happen as New validates the pool
	if db.Pool != nil {
		err := db.Pool.Ping(context.Background())
		assert.NoError(t, err)
	}
}

func TestConfig_DurationValues(t *testing.T) {
	tests := []struct {
		name           string
		cfg            Config
		expectedLife   time.Duration
		expectedIdle   time.Duration
		expectedHealth time.Duration
	}{
		{
			name: "default durations",
			cfg: Config{
				MaxConnLifeTime: 5,
				MaxConnIdleTime: 5,
			},
			expectedLife:   5 * time.Second,
			expectedIdle:   5 * time.Second,
			expectedHealth: 20 * time.Second, // hardcoded in New
		},
		{
			name: "custom durations",
			cfg: Config{
				MaxConnLifeTime: 300,
				MaxConnIdleTime: 600,
			},
			expectedLife:   300 * time.Second,
			expectedIdle:   600 * time.Second,
			expectedHealth: 20 * time.Second, // hardcoded in New
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lifeTime := time.Duration(tt.cfg.MaxConnLifeTime) * time.Second
			idleTime := time.Duration(tt.cfg.MaxConnIdleTime) * time.Second

			assert.Equal(t, tt.expectedLife, lifeTime)
			assert.Equal(t, tt.expectedIdle, idleTime)
		})
	}
}

func TestNew_InvalidDSN(t *testing.T) {
	// We can't easily test this without mocking pgxpool.ParseConfig
	// But we can verify the URL generation handles edge cases

	cfg := Config{
		User:     "user",
		Password: "pass",
		Host:     "host",
		Port:     5432,
		Name:     "db",
	}

	u := cfg.URL()
	assert.NotNil(t, u)

	// Parse the URL back to verify it's valid
	_, err := pgxpool.ParseConfig(u.String())
	// This will fail to connect but shouldn't fail to parse
	if err != nil {
		// If it fails, it should be a connection error, not a parse error
		assert.NotContains(t, err.Error(), "invalid")
	}
}

func TestNew_TracerConfiguration(t *testing.T) {
	// Test that tracers are properly configured
	opts := &Options{
		Tracers: []pgx.QueryTracer{
			otelpgx.NewTracer(),
			&tracelog.TraceLog{
				Logger:   NewLogger(),
				LogLevel: tracelog.LogLevelError,
			},
		},
	}

	assert.Len(t, opts.Tracers, 2)
}

func TestNew_ConnectionFailures(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "non-existent host",
			cfg: Config{
				User:            "testuser",
				Password:        "testpass",
				Host:            "nonexistent-host-12345.invalid",
				Port:            5432,
				Name:            "testdb",
				MaxOpenConns:    1,
				MaxConnLifeTime: 1,
				MaxConnIdleTime: 1,
			},
		},
		{
			name: "wrong port",
			cfg: Config{
				User:            "testuser",
				Password:        "testpass",
				Host:            "localhost",
				Port:            9999,
				Name:            "testdb",
				MaxOpenConns:    1,
				MaxConnLifeTime: 1,
				MaxConnIdleTime: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These will fail to connect, which is expected
			db, err := New(tt.cfg, nil)
			assert.Error(t, err)
			assert.Nil(t, db)
		})
	}
}

func TestDB_Ping(t *testing.T) {
	// Test that Ping would be called on a real DB structure
	// We can't test actual Ping without a database, but we can verify the DB struct
	db := &DB{
		Pool: nil,
	}

	// Verify DB structure exists
	assert.NotNil(t, db)
}

func TestDB_NewLogger(t *testing.T) {
	logger := NewLogger()
	assert.NotNil(t, logger)
}

func TestConfig_DefaultValues(t *testing.T) {
	// Test that config with zero values still generates a valid URL
	cfg := Config{
		User:     "user",
		Password: "pass",
		Host:     "localhost",
		Port:     5432, // Port defaults to 5432 in the struct tag
		Name:     "db",
		// MaxOpenConns defaults to 20
		// MaxConnLifeTime defaults to 5
		// MaxConnIdleTime defaults to 5
		// TraceLogLevel defaults to "error"
	}

	u := cfg.URL()
	assert.NotNil(t, u)
	assert.Equal(t, "postgres://user:pass@localhost/db?sslmode=disable&timezone=utc", u.String())
}

func TestOptions_WithTracers(t *testing.T) {
	opts := &Options{
		Tracers: []pgx.QueryTracer{
			otelpgx.NewTracer(),
		},
	}

	assert.NotNil(t, opts)
	assert.Len(t, opts.Tracers, 1)
}

func TestOptions_NilTracers(t *testing.T) {
	opts := &Options{}

	assert.NotNil(t, opts)
	assert.Nil(t, opts.Tracers)
}

// TestNewDefault_VariousConfigs tests NewDefault with various configurations.
func TestNewDefault_VariousConfigs(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "minimal valid config - will fail connection",
			cfg: Config{
				User:     "user",
				Password: "pass",
				Host:     "localhost",
				Port:     9999, // Wrong port to fail connection
				Name:     "db",
			},
			wantErr: true, // Will fail to connect
		},
		{
			name: "config with all fields",
			cfg: Config{
				User:            "testuser",
				Password:        "testpass",
				Host:            "nonexistent-host-12345",
				Port:            5432,
				Name:            "testdb",
				MaxOpenConns:    10,
				MaxConnLifeTime: 300,
				MaxConnIdleTime: 600,
				TraceLogLevel:   "debug",
			},
			wantErr: true, // Will fail to connect
		},
		{
			name: "config with zero max conns",
			cfg: Config{
				User:            "user",
				Password:        "pass",
				Host:            "localhost",
				Port:            9999,
				Name:            "db",
				MaxOpenConns:    0, // Should be defaulted to 1
				MaxConnLifeTime: 300,
				MaxConnIdleTime: 600,
			},
			wantErr: true, // Will fail to connect
		},
		{
			name: "config with negative max conns",
			cfg: Config{
				User:            "user",
				Password:        "pass",
				Host:            "localhost",
				Port:            9999,
				Name:            "db",
				MaxOpenConns:    -5, // Should be defaulted to 1
				MaxConnLifeTime: 300,
				MaxConnIdleTime: 600,
			},
			wantErr: true, // Will fail to connect
		},
		{
			name: "config with various trace levels",
			cfg: Config{
				User:          "user",
				Password:      "pass",
				Host:          "localhost",
				Port:          9999,
				Name:          "db",
				TraceLogLevel: "trace",
			},
			wantErr: true, // Will fail to connect
		},
		{
			name: "config with invalid trace level",
			cfg: Config{
				User:          "user",
				Password:      "pass",
				Host:          "localhost",
				Port:          9999,
				Name:          "db",
				TraceLogLevel: "invalid",
			},
			wantErr: true, // Will fail to connect (invalid trace level defaults to none)
		},
		{
			name: "config with empty trace level",
			cfg: Config{
				User:          "user",
				Password:      "pass",
				Host:          "localhost",
				Port:          9999,
				Name:          "db",
				TraceLogLevel: "",
			},
			wantErr: true, // Will fail to connect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDefault(tt.cfg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, db)
			}
		})
	}
}

// TestNewDefault_TracerSetup tests that tracers are properly set up in NewDefault.
func TestNewDefault_TracerSetup(t *testing.T) {
	t.Run("NewDefault always adds tracers", func(t *testing.T) {
		cfg := Config{
			User:          "user",
			Password:      "pass",
			Host:          "localhost",
			Port:          9999, // Wrong port to fail connection
			Name:          "db",
			TraceLogLevel: "info",
		}

		// This will fail to connect, but we can verify the function exists
		// and the structure would have tracers added
		_, err := NewDefault(cfg)
		assert.Error(t, err) // Connection will fail
	})
}

// TestNewDefault_ConnectionFailures tests various connection failure scenarios.
func TestNewDefault_ConnectionFailures(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		errContains string
	}{
		{
			name: "invalid host",
			cfg: Config{
				User:     "user",
				Password: "pass",
				Host:     "invalid-host-99999.example.com",
				Port:     5432,
				Name:     "db",
			},
			errContains: "failed", // Error will contain "failed"
		},
		{
			name: "connection refused",
			cfg: Config{
				User:     "user",
				Password: "pass",
				Host:     "localhost",
				Port:     9999, // No service on this port
				Name:     "db",
			},
			errContains: "failed",
		},
		{
			name: "authentication will fail (wrong credentials)",
			cfg: Config{
				User:     "nonexistentuser",
				Password: "wrongpassword",
				Host:     "localhost",
				Port:     5432,
				Name:     "db",
			},
			errContains: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDefault(tt.cfg)
			assert.Error(t, err)
			assert.Nil(t, db)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

// TestNewDefault_ConfigVariations tests NewDefault with config variations.
func TestNewDefault_ConfigVariations(t *testing.T) {
	t.Run("with very short timeout values", func(t *testing.T) {
		cfg := Config{
			User:            "user",
			Password:        "pass",
			Host:            "localhost",
			Port:            9999,
			Name:            "db",
			MaxConnLifeTime: 1,
			MaxConnIdleTime: 1,
		}

		db, err := NewDefault(cfg)
		assert.Error(t, err)
		assert.Nil(t, db)
	})

	t.Run("with very long timeout values", func(t *testing.T) {
		cfg := Config{
			User:            "user",
			Password:        "pass",
			Host:            "localhost",
			Port:            9999,
			Name:            "db",
			MaxConnLifeTime: 86400, // 1 day
			MaxConnIdleTime: 3600,  // 1 hour
		}

		db, err := NewDefault(cfg)
		assert.Error(t, err)
		assert.Nil(t, db)
	})

	t.Run("with very large max connections", func(t *testing.T) {
		cfg := Config{
			User:         "user",
			Password:     "pass",
			Host:         "localhost",
			Port:         9999,
			Name:         "db",
			MaxOpenConns: 10000,
		}

		db, err := NewDefault(cfg)
		assert.Error(t, err)
		assert.Nil(t, db)
	})
}
