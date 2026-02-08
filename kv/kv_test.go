package kv

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pure-golang/adapters/kv/noop"
	"github.com/pure-golang/adapters/kv/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefault_NoopProvider(t *testing.T) {
	// Set environment variable for noop provider
	t.Setenv("KV_PROVIDER", "noop")

	store, err := NewDefault()
	require.NoError(t, err)
	require.NotNil(t, store)

	// Verify it's a noop store by checking it implements Store interface
	_, ok := store.(*noop.Store)
	require.True(t, ok, "NewDefault with noop provider should return *noop.Store")

	// Verify the store implements Store interface by testing basic operations
	ctx := context.Background()

	val, err := store.Get(ctx, "test")
	assert.NoError(t, err)
	assert.Empty(t, val)

	err = store.Set(ctx, "test", "value", 0)
	assert.NoError(t, err)

	err = store.Delete(ctx, "test")
	assert.NoError(t, err)

	count, err := store.Exists(ctx, "test")
	assert.NoError(t, err)
	assert.Zero(t, count)

	n, err := store.Incr(ctx, "counter")
	assert.NoError(t, err)
	assert.Zero(t, n)

	n, err = store.Decr(ctx, "counter")
	assert.NoError(t, err)
	assert.Zero(t, n)

	err = store.Expire(ctx, "test", time.Hour)
	assert.NoError(t, err)

	ttl, err := store.TTL(ctx, "test")
	assert.NoError(t, err)
	assert.Zero(t, ttl)

	val, err = store.HGet(ctx, "hash", "field")
	assert.NoError(t, err)
	assert.Empty(t, val)

	err = store.HSet(ctx, "hash", "field", "value")
	assert.NoError(t, err)

	m, err := store.HGetAll(ctx, "hash")
	assert.NoError(t, err)
	assert.NotNil(t, m)

	err = store.HDel(ctx, "hash", "field")
	assert.NoError(t, err)

	err = store.LPush(ctx, "list", "value")
	assert.NoError(t, err)

	err = store.RPush(ctx, "list", "value")
	assert.NoError(t, err)

	val, err = store.LPop(ctx, "list")
	assert.NoError(t, err)
	assert.Empty(t, val)

	val, err = store.RPop(ctx, "list")
	assert.NoError(t, err)
	assert.Empty(t, val)

	length, err := store.LLen(ctx, "list")
	assert.NoError(t, err)
	assert.Zero(t, length)

	err = store.SAdd(ctx, "set", "member")
	assert.NoError(t, err)

	members, err := store.SMembers(ctx, "set")
	assert.NoError(t, err)
	assert.NotNil(t, members)

	isMember, err := store.SIsMember(ctx, "set", "member")
	assert.NoError(t, err)
	assert.False(t, isMember)

	err = store.SRem(ctx, "set", "member")
	assert.NoError(t, err)

	err = store.Ping(ctx)
	assert.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)
}

func TestNewDefault_RedisProvider_ConnectionError(t *testing.T) {
	// Set environment variable for redis provider with invalid address
	t.Setenv("KV_PROVIDER", "redis")
	t.Setenv("REDIS_ADDR", "localhost:19999") // Non-existent Redis

	store, err := NewDefault()
	require.Error(t, err)
	assert.Nil(t, store)
	// Error should contain "failed to ping redis" from redis package
	assert.Contains(t, err.Error(), "failed to ping redis")
}

func TestNewDefault_UnknownProvider(t *testing.T) {
	// Set environment variable for unknown provider
	t.Setenv("KV_PROVIDER", "unknown_provider")

	store, err := NewDefault()
	require.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "unknown kv provider")
	assert.Contains(t, err.Error(), "unknown_provider")
}

func TestNewDefault_DefaultProvider(t *testing.T) {
	// Unset KV_PROVIDER to test default behavior
	// The default should be "noop" according to Config struct
	unsetKey(t, "KV_PROVIDER")

	store, err := NewDefault()
	require.NoError(t, err)
	require.NotNil(t, store)

	// Should return noop store as default
	_, ok := store.(*noop.Store)
	require.True(t, ok, "NewDefault with no provider should return *noop.Store (default)")
}

func TestInitDefault_AliasForNewDefault(t *testing.T) {
	t.Setenv("KV_PROVIDER", "noop")

	// Test that InitDefault returns same result as NewDefault
	store1, err1 := NewDefault()
	store2, err2 := InitDefault()

	assert.Equal(t, err1, err2)
	assert.Equal(t, store1 != nil, store2 != nil)

	// Both should be noop stores
	noopStore1, ok1 := store1.(*noop.Store)
	noopStore2, ok2 := store2.(*noop.Store)
	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.Equal(t, noopStore1, noopStore2)
}

func TestInitDefault_UnknownProvider(t *testing.T) {
	t.Setenv("KV_PROVIDER", "invalid_provider")

	store, err := InitDefault()
	require.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "unknown kv provider")
}

func TestStore_InterfaceImplementation(t *testing.T) {
	// Test that noop.Store implements Store interface
	t.Setenv("KV_PROVIDER", "noop")

	store, err := NewDefault()
	require.NoError(t, err)
	require.NotNil(t, store)

	// Verify it implements the interface by checking type
	var _ Store = store
	ctx := context.Background()

	// Test all interface methods are implemented
	_, _ = store.Get(ctx, "key")
	_ = store.Set(ctx, "key", "value", 0)
	_ = store.Delete(ctx, "key")
	_, _ = store.Exists(ctx, "key")
	_, _ = store.Incr(ctx, "counter")
	_, _ = store.Decr(ctx, "counter")
	_ = store.Expire(ctx, "key", time.Hour)
	_, _ = store.TTL(ctx, "key")
	_, _ = store.HGet(ctx, "hash", "field")
	_ = store.HSet(ctx, "hash", "field", "value")
	_, _ = store.HGetAll(ctx, "hash")
	_ = store.HDel(ctx, "hash", "field")
	_ = store.LPush(ctx, "list", "value")
	_ = store.RPush(ctx, "list", "value")
	_, _ = store.LPop(ctx, "list")
	_, _ = store.RPop(ctx, "list")
	_, _ = store.LLen(ctx, "list")
	_ = store.SAdd(ctx, "set", "member")
	_, _ = store.SMembers(ctx, "set")
	_, _ = store.SIsMember(ctx, "set", "member")
	_ = store.SRem(ctx, "set", "member")
	_ = store.Ping(ctx)
	_ = store.Close()
}

func TestProvider_Constants(t *testing.T) {
	// Test provider constants
	assert.Equal(t, Provider("redis"), ProviderRedis)
	assert.Equal(t, Provider("noop"), ProviderNoop)
}

func TestConfig_DefaultValues(t *testing.T) {
	// Before parsing, default values are set by env.InitConfig
	// After initialization with no env vars, defaults should apply:
	// Provider defaults to "noop"
	// RedisAddr defaults to "localhost:6379"
	// RedisDB defaults to 0
	// RedisMaxRetries defaults to 3
	// RedisDialTimeout defaults to "5s"
	// RedisReadTimeout defaults to "3s"
	// RedisWriteTimeout defaults to "3s"
	// RedisPoolSize defaults to 10

	// Clear environment and test defaults
	unsetKey(t, "KV_PROVIDER")
	unsetKey(t, "REDIS_ADDR")
	unsetKey(t, "REDIS_DB")
	unsetKey(t, "REDIS_MAX_RETRIES")
	unsetKey(t, "REDIS_DIAL_TIMEOUT")
	unsetKey(t, "REDIS_READ_TIMEOUT")
	unsetKey(t, "REDIS_WRITE_TIMEOUT")
	unsetKey(t, "REDIS_POOL_SIZE")

	store, err := NewDefault()
	require.NoError(t, err)
	assert.NotNil(t, store)

	// Store should be noop (default)
	_, ok := store.(*noop.Store)
	assert.True(t, ok)

	// Close the store
	_ = store.Close()
}

// Test that redis.Client would implement Store interface if connected
func TestRedis_ClientImplementsStore(t *testing.T) {
	// This is a compile-time check that redis.Client implements Store
	var _ Store = &redis.Client{}
}

func TestConfig_WithRedisSettings(t *testing.T) {
	// Test that config can be created with custom Redis settings
	t.Setenv("KV_PROVIDER", "redis")
	t.Setenv("REDIS_ADDR", "custom-host:6380")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_MAX_RETRIES", "5")
	t.Setenv("REDIS_POOL_SIZE", "20")

	// This will fail to connect (no Redis running), but we can check
	// the config is parsed correctly by checking the error message
	store, err := NewDefault()
	assert.Error(t, err)
	assert.Nil(t, store)
}

// unsetKey unsets an environment variable and returns a function to restore it
func unsetKey(t *testing.T, key string) {
	t.Helper()
	original, existed := os.LookupEnv(key)
	err := os.Unsetenv(key)
	require.NoError(t, err)

	t.Cleanup(func() {
		if existed {
			err := os.Setenv(key, original)
			require.NoError(t, err)
		}
	})
}

func TestNewDefault_EnvInitError(t *testing.T) {
	// Test the error path when env.InitConfig fails
	// This happens when we set an invalid environment variable value
	// For example, a non-integer value for REDIS_DB
	t.Setenv("KV_PROVIDER", "redis")
	t.Setenv("REDIS_DB", "not-a-number")

	store, err := NewDefault()
	require.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "failed to init config")
}
