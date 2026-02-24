package kv_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/kv"
	kvnoop "github.com/pure-golang/adapters/kv/noop"
	kvredis "github.com/pure-golang/adapters/kv/redis"
)

func TestNoop_Store(t *testing.T) {
	t.Parallel()
	store := kvnoop.New()
	require.NotNil(t, store)

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

func TestRedis_ConnectionError(t *testing.T) {
	t.Parallel()
	store, err := kvredis.NewDefault(kvredis.Config{Addr: "localhost:19999"})
	require.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "failed to ping redis")
}

func TestRedis_ClientImplementsStore(t *testing.T) {
	t.Parallel()
	var _ kv.Store = &kvredis.Client{}
}

func TestNoop_StoreInterfaceMethods(t *testing.T) {
	t.Parallel()
	store := kvnoop.New()
	require.NotNil(t, store)

	ctx := context.Background()

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
