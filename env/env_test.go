package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig is a sample config struct for testing
type TestConfig struct {
	Host     string `envconfig:"HOST" default:"localhost"`
	Port     int    `envconfig:"PORT" default:"8080"`
	Required string `envconfig:"REQUIRED"`
	Optional string `envconfig:"OPTIONAL"`
}

// TestConfigWithRequired is a config with explicitly required fields
type TestConfigWithRequired struct {
	Host string `envconfig:"HOST" required:"true"`
	Port int    `envconfig:"PORT" required:"true"`
}

// TestConfigWithDefaults tests a config with only default values
type TestConfigWithDefaults struct {
	Host string `envconfig:"HOST" default:"localhost"`
	Port int    `envconfig:"PORT" default:"8080"`
}

// TestConfigNoDefaults tests a config with required fields and no defaults
type TestConfigNoDefaults struct {
	Host    string `envconfig:"HOST"`
	Port    int    `envconfig:"PORT"`
	APIKey  string `envconfig:"API_KEY"`
	Enabled bool   `envconfig:"ENABLED" default:"false"`
}

func TestInitConfig_ValidStruct(t *testing.T) {
	// Set up environment variables
	t.Setenv("HOST", "example.com")
	t.Setenv("PORT", "9000")
	t.Setenv("REQUIRED", "value")

	var cfg TestConfig
	err := InitConfig(&cfg)

	require.NoError(t, err)
	assert.Equal(t, "example.com", cfg.Host)
	assert.Equal(t, 9000, cfg.Port)
	assert.Equal(t, "value", cfg.Required)
}

func TestInitConfig_MissingRequiredEnvVar(t *testing.T) {
	// Don't set required env vars
	// Both HOST and PORT are required but not set

	var cfg TestConfigWithRequired
	err := InitConfig(&cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to envconfig.Process")
}

func TestInitConfig_DefaultValues(t *testing.T) {
	// Don't set any env vars, rely on defaults
	var cfg TestConfigWithDefaults
	err := InitConfig(&cfg)

	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
}

func TestInitConfig_PartialEnvVarsWithDefaults(t *testing.T) {
	t.Setenv("HOST", "customhost")
	// PORT not set, should use default
	t.Setenv("REQUIRED", "value")

	var cfg TestConfig
	err := InitConfig(&cfg)

	require.NoError(t, err)
	assert.Equal(t, "customhost", cfg.Host)
	assert.Equal(t, 8080, cfg.Port) // default value
	assert.Equal(t, "value", cfg.Required)
}

func TestInitConfig_BoolDefault(t *testing.T) {
	var cfg TestConfigNoDefaults
	t.Setenv("HOST", "localhost")
	t.Setenv("PORT", "8080")
	t.Setenv("API_KEY", "secret")
	// ENABLED not set, should use default

	err := InitConfig(&cfg)

	require.NoError(t, err)
	assert.Equal(t, false, cfg.Enabled)
}

func TestInitConfig_BoolSet(t *testing.T) {
	var cfg TestConfigNoDefaults
	t.Setenv("HOST", "localhost")
	t.Setenv("PORT", "8080")
	t.Setenv("API_KEY", "secret")
	t.Setenv("ENABLED", "true")

	err := InitConfig(&cfg)

	require.NoError(t, err)
	assert.Equal(t, true, cfg.Enabled)
}

func TestInitConfig_WithDotEnvFile(t *testing.T) {
	// Create a temporary .env file
	tmpDir := t.TempDir()
	envFilePath := filepath.Join(tmpDir, ".env")
	content := "HOST=fromdotenv\nPORT=7000\nREQUIRED=fromfile"
	err := os.WriteFile(envFilePath, []byte(content), 0600)
	require.NoError(t, err)

	// Change working directory to tmpDir so InitConfig finds the .env file
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	var cfg TestConfig
	err = InitConfig(&cfg)

	require.NoError(t, err)
	assert.Equal(t, "fromdotenv", cfg.Host)
	assert.Equal(t, 7000, cfg.Port)
	assert.Equal(t, "fromfile", cfg.Required)
}

func TestInitConfig_WithoutDotEnvFile(t *testing.T) {
	// Create a temp directory without .env file
	tmpDir := t.TempDir()

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	// Set env vars directly instead
	t.Setenv("HOST", "direct")
	t.Setenv("PORT", "6000")
	t.Setenv("REQUIRED", "directvalue")

	var cfg TestConfig
	err = InitConfig(&cfg)

	// Should succeed even without .env file
	require.NoError(t, err)
	assert.Equal(t, "direct", cfg.Host)
	assert.Equal(t, 6000, cfg.Port)
	assert.Equal(t, "directvalue", cfg.Required)
}

func TestInitConfig_EnvVarsOverrideDotEnv(t *testing.T) {
	// Create a .env file
	tmpDir := t.TempDir()
	envFilePath := filepath.Join(tmpDir, ".env")
	content := "HOST=fromdotenv\nPORT=7000\nREQUIRED=fromfile"
	err := os.WriteFile(envFilePath, []byte(content), 0600)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	// Set env var that should override .env
	t.Setenv("HOST", "override")

	var cfg TestConfig
	err = InitConfig(&cfg)

	require.NoError(t, err)
	// Environment variables should take precedence over .env file
	assert.Equal(t, "override", cfg.Host)
	assert.Equal(t, 7000, cfg.Port)
	assert.Equal(t, "fromfile", cfg.Required)
}

func TestInitConfig_NilPointer(t *testing.T) {
	err := InitConfig(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to envconfig.Process")
}

func TestInitConfig_NonStructPointer(t *testing.T) {
	var notAStruct string
	err := InitConfig(&notAStruct)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to envconfig.Process")
}

func TestInitConfig_EmptyStringField(t *testing.T) {
	t.Setenv("HOST", "example.com")
	t.Setenv("PORT", "8080")
	t.Setenv("REQUIRED", "")

	var cfg TestConfig
	err := InitConfig(&cfg)

	require.NoError(t, err)
	assert.Equal(t, "", cfg.Required)
}

func TestInitConfig_InvalidPortFormat(t *testing.T) {
	t.Setenv("HOST", "example.com")
	t.Setenv("PORT", "invalid")
	t.Setenv("REQUIRED", "value")

	var cfg TestConfig
	err := InitConfig(&cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to envconfig.Process")
}

func TestInitConfig_DefaultEnvFileConstant(t *testing.T) {
	assert.Equal(t, ".env", DefaultEnvFile)
}
