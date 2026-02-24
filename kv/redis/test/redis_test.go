package redis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/pure-golang/adapters/kv/redis"
)

type RedisSuite struct {
	suite.Suite
	container testcontainers.Container
	addr      string
}

func TestRedisSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	suite.Run(t, new(RedisSuite))
}

func (s *RedisSuite) SetupSuite() {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	s.Require().NoError(err, "failed to start container")

	s.container = container

	host, err := container.Host(ctx)
	s.Require().NoError(err, "failed to get container host")

	port, err := container.MappedPort(ctx, "6379")
	s.Require().NoError(err, "failed to get container port")

	s.addr = fmt.Sprintf("%s:%s", host, port.Port())
}

func (s *RedisSuite) TearDownSuite() {
	if s.container != nil {
		ctx := context.Background()
		if err := s.container.Terminate(ctx); err != nil {
			s.T().Logf("failed to terminate container: %v", err)
		}
	}
}

func (s *RedisSuite) TestConnect() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}

	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.Ping(ctx))
}

func (s *RedisSuite) TestNewDefault() {
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.NewDefault(cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.Ping(context.Background()))
}

func (s *RedisSuite) TestSetAndGet() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.Set(ctx, "test_key", "test_value", 0))

	val, err := client.Get(ctx, "test_key")
	s.Require().NoError(err)
	s.Equal("test_value", val)

	_, err = client.Get(ctx, "non_existent")
	s.ErrorIs(err, redis.ErrKeyNotFound)
}

func (s *RedisSuite) TestSetWithTTL() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.Set(ctx, "ttl_key", "ttl_value", time.Second))

	ttl, err := client.TTL(ctx, "ttl_key")
	s.Require().NoError(err)
	s.True(ttl >= time.Millisecond*900 && ttl <= time.Second, "unexpected TTL: %v", ttl)

	val, err := client.Get(ctx, "ttl_key")
	s.Require().NoError(err)
	s.Equal("ttl_value", val)

	time.Sleep(time.Second + 100*time.Millisecond)

	_, err = client.Get(ctx, "ttl_key")
	s.ErrorIs(err, redis.ErrKeyNotFound)
}

func (s *RedisSuite) TestDelete() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	for i := range 3 {
		s.Require().NoError(client.Set(ctx, fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), 0))
	}

	s.Require().NoError(client.Delete(ctx, "key0", "key1"))

	count, err := client.Exists(ctx, "key0", "key1", "key2")
	s.Require().NoError(err)
	s.Equal(int64(1), count)
}

func (s *RedisSuite) TestIncrDecr() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.Set(ctx, "counter", 10, 0))

	val, err := client.Incr(ctx, "counter")
	s.Require().NoError(err)
	s.Equal(int64(11), val)

	val, err = client.Decr(ctx, "counter")
	s.Require().NoError(err)
	s.Equal(int64(10), val)
}

func (s *RedisSuite) TestHashOperations() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.HSet(ctx, "hash", "field1", "value1"))

	val, err := client.HGet(ctx, "hash", "field1")
	s.Require().NoError(err)
	s.Equal("value1", val)

	_, err = client.HGet(ctx, "hash", "non_existent")
	s.ErrorIs(err, redis.ErrKeyNotFound)

	s.Require().NoError(client.HSet(ctx, "hash", "field2", "value2"))

	all, err := client.HGetAll(ctx, "hash")
	s.Require().NoError(err)
	s.Equal(2, len(all))

	s.Require().NoError(client.HDel(ctx, "hash", "field1"))

	all, err = client.HGetAll(ctx, "hash")
	s.Require().NoError(err)
	s.Equal(1, len(all))
}

func (s *RedisSuite) TestListOperations() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.LPush(ctx, "list", "item1", "item2"))
	s.Require().NoError(client.RPush(ctx, "list", "item3"))

	length, err := client.LLen(ctx, "list")
	s.Require().NoError(err)
	s.Equal(int64(3), length)

	item, err := client.LPop(ctx, "list")
	s.Require().NoError(err)
	s.Equal("item2", item)

	item, err = client.RPop(ctx, "list")
	s.Require().NoError(err)
	s.Equal("item3", item)
}

func (s *RedisSuite) TestSetOperations() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.SAdd(ctx, "set", "member1", "member2", "member3"))

	members, err := client.SMembers(ctx, "set")
	s.Require().NoError(err)
	s.Equal(3, len(members))

	isMember, err := client.SIsMember(ctx, "set", "member1")
	s.Require().NoError(err)
	s.True(isMember)

	isMember, err = client.SIsMember(ctx, "set", "non_existent")
	s.Require().NoError(err)
	s.False(isMember)

	s.Require().NoError(client.SRem(ctx, "set", "member1"))

	members, err = client.SMembers(ctx, "set")
	s.Require().NoError(err)
	s.Equal(2, len(members))
}

func (s *RedisSuite) TestConcurrentOperations() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	const goroutines = 10
	const iterations = 10

	done := make(chan bool, goroutines)
	for i := range goroutines {
		go func(n int) {
			for range iterations {
				key := fmt.Sprintf("counter%d", n)
				if _, err := client.Incr(ctx, key); err != nil {
					s.T().Errorf("Incr failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	for range goroutines {
		<-done
	}

	for i := range goroutines {
		val, err := client.Get(ctx, fmt.Sprintf("counter%d", i))
		s.Require().NoError(err)
		s.Equal(fmt.Sprintf("%d", iterations), val)
	}
}

func (s *RedisSuite) TestClose() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)

	s.Require().NoError(client.Close())
	s.Require().NoError(client.Close())
}

func (s *RedisSuite) TestEmptyOperations() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.Delete(ctx))

	count, err := client.Exists(ctx)
	s.Require().NoError(err)
	s.Equal(int64(0), count)

	_, err = client.LPop(ctx, "empty_list")
	s.ErrorIs(err, redis.ErrKeyNotFound)

	_, err = client.RPop(ctx, "empty_list")
	s.ErrorIs(err, redis.ErrKeyNotFound)

	length, err := client.LLen(ctx, "empty_list")
	s.Require().NoError(err)
	s.Equal(int64(0), length)

	all, err := client.HGetAll(ctx, "empty_hash")
	s.Require().NoError(err)
	if all == nil {
		all = map[string]string{}
	}
	s.Equal(0, len(all))

	s.Require().NoError(client.HDel(ctx, "empty_hash"))
	s.Require().NoError(client.LPush(ctx, "list"))
	s.Require().NoError(client.RPush(ctx, "list"))
	s.Require().NoError(client.SAdd(ctx, "set"))

	members, err := client.SMembers(ctx, "empty_set")
	s.Require().NoError(err)
	if members == nil {
		members = []string{}
	}
	s.Equal(0, len(members))

	s.Require().NoError(client.SRem(ctx, "set"))
}

func (s *RedisSuite) TestContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(context.Background(), cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	_, err = client.Get(ctx, "test")
	s.Error(err)
}

func (s *RedisSuite) TestExpire() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.Set(ctx, "expire_key", "value", 0))

	ttl, err := client.TTL(ctx, "expire_key")
	s.Require().NoError(err)
	if ttl != -1*time.Second && ttl > 0 {
		s.T().Logf("Note: TTL for key without expiration is %v", ttl)
	}

	s.Require().NoError(client.Expire(ctx, "expire_key", time.Minute))

	ttl, err = client.TTL(ctx, "expire_key")
	s.Require().NoError(err)
	s.True(ttl >= 30*time.Second, "expected TTL >= 30s, got %v", ttl)

	val, err := client.Get(ctx, "expire_key")
	s.Require().NoError(err)
	s.Equal("value", val)

	s.Require().NoError(client.Expire(ctx, "expire_key", time.Second))
	time.Sleep(1100 * time.Millisecond)

	_, err = client.Get(ctx, "expire_key")
	s.ErrorIs(err, redis.ErrKeyNotFound)
}

func (s *RedisSuite) TestTypeMismatch() {
	ctx := context.Background()
	cfg := redis.Config{Addr: s.addr}
	client, err := redis.Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.Set(ctx, "string_key", "string_value", 0))

	_, err = client.HGet(ctx, "string_key", "field")
	s.Error(err)

	_, err = client.LPop(ctx, "string_key")
	s.Error(err)

	_, err = client.SMembers(ctx, "string_key")
	s.Error(err)

	_, err = client.Incr(ctx, "string_key")
	s.Error(err)
}
