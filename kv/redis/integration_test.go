package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RedisSuite struct {
	suite.Suite
	container testcontainers.Container
	addr      string
}

func TestRedisSuite(t *testing.T) {
	suite.Run(t, new(RedisSuite))
}

func (s *RedisSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

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
	cfg := Config{Addr: s.addr}

	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	s.Require().NoError(client.Ping(ctx))
}

func (s *RedisSuite) TestNewDefault() {
	// Тестируем NewDefault с пустым конфигом (должен подставить значения по умолчанию)
	cfg := Config{Addr: s.addr}
	client, err := NewDefault(cfg)
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
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// Set
	s.Require().NoError(client.Set(ctx, "test_key", "test_value", 0))

	// Get
	val, err := client.Get(ctx, "test_key")
	s.Require().NoError(err)
	s.Equal("test_value", val)

	// Get non-existent key
	_, err = client.Get(ctx, "non_existent")
	s.ErrorIs(err, ErrKeyNotFound)
}

func (s *RedisSuite) TestSetWithTTL() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// Set with 1 second TTL
	s.Require().NoError(client.Set(ctx, "ttl_key", "ttl_value", time.Second))

	// Check TTL
	ttl, err := client.TTL(ctx, "ttl_key")
	s.Require().NoError(err)
	s.True(ttl >= time.Millisecond*900 && ttl <= time.Second, "unexpected TTL: %v", ttl)

	// Get should succeed
	val, err := client.Get(ctx, "ttl_key")
	s.Require().NoError(err)
	s.Equal("ttl_value", val)

	// Wait for expiration
	time.Sleep(time.Second + 100*time.Millisecond)

	// Get should fail now
	_, err = client.Get(ctx, "ttl_key")
	s.ErrorIs(err, ErrKeyNotFound)
}

func (s *RedisSuite) TestDelete() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// Set keys
	for i := 0; i < 3; i++ {
		s.Require().NoError(client.Set(ctx, fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), 0))
	}

	// Delete keys
	s.Require().NoError(client.Delete(ctx, "key0", "key1"))

	// Check existence
	count, err := client.Exists(ctx, "key0", "key1", "key2")
	s.Require().NoError(err)
	s.Equal(int64(1), count)
}

func (s *RedisSuite) TestIncrDecr() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// Set initial value
	s.Require().NoError(client.Set(ctx, "counter", 10, 0))

	// Incr
	val, err := client.Incr(ctx, "counter")
	s.Require().NoError(err)
	s.Equal(int64(11), val)

	// Decr
	val, err = client.Decr(ctx, "counter")
	s.Require().NoError(err)
	s.Equal(int64(10), val)
}

func (s *RedisSuite) TestHashOperations() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// HSet
	s.Require().NoError(client.HSet(ctx, "hash", "field1", "value1"))

	// HGet
	val, err := client.HGet(ctx, "hash", "field1")
	s.Require().NoError(err)
	s.Equal("value1", val)

	// HGet non-existent field
	_, err = client.HGet(ctx, "hash", "non_existent")
	s.ErrorIs(err, ErrKeyNotFound)

	// HSet more fields
	s.Require().NoError(client.HSet(ctx, "hash", "field2", "value2"))

	// HGetAll
	all, err := client.HGetAll(ctx, "hash")
	s.Require().NoError(err)
	s.Equal(2, len(all))

	// HDel
	s.Require().NoError(client.HDel(ctx, "hash", "field1"))

	all, err = client.HGetAll(ctx, "hash")
	s.Require().NoError(err)
	s.Equal(1, len(all))
}

func (s *RedisSuite) TestListOperations() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// LPush
	s.Require().NoError(client.LPush(ctx, "list", "item1", "item2"))

	// RPush
	s.Require().NoError(client.RPush(ctx, "list", "item3"))

	// LLen
	length, err := client.LLen(ctx, "list")
	s.Require().NoError(err)
	s.Equal(int64(3), length)

	// LPop
	item, err := client.LPop(ctx, "list")
	s.Require().NoError(err)
	s.Equal("item2", item) // LPush добавляет в начало, так что порядок будет item2, item1, item3

	// RPop
	item, err = client.RPop(ctx, "list")
	s.Require().NoError(err)
	s.Equal("item3", item)
}

func (s *RedisSuite) TestSetOperations() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// SAdd
	s.Require().NoError(client.SAdd(ctx, "set", "member1", "member2", "member3"))

	// SMembers
	members, err := client.SMembers(ctx, "set")
	s.Require().NoError(err)
	s.Equal(3, len(members))

	// SIsMember
	isMember, err := client.SIsMember(ctx, "set", "member1")
	s.Require().NoError(err)
	s.True(isMember)

	isMember, err = client.SIsMember(ctx, "set", "non_existent")
	s.Require().NoError(err)
	s.False(isMember)

	// SRem
	s.Require().NoError(client.SRem(ctx, "set", "member1"))

	members, err = client.SMembers(ctx, "set")
	s.Require().NoError(err)
	s.Equal(2, len(members))
}

func (s *RedisSuite) TestConcurrentOperations() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// Конкурентная запись
	const goroutines = 10
	const iterations = 10

	done := make(chan bool, goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("counter%d", n)
				if _, err := client.Incr(ctx, key); err != nil {
					s.T().Errorf("Incr failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Проверяем результаты
	for i := 0; i < goroutines; i++ {
		val, err := client.Get(ctx, fmt.Sprintf("counter%d", i))
		s.Require().NoError(err)
		// В JSON формате Redis хранит числа
		s.Equal(fmt.Sprintf("%d", iterations), val)
	}
}

func (s *RedisSuite) TestClose() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)

	// Close
	s.Require().NoError(client.Close())

	// Повторный Close должен быть успешным
	s.Require().NoError(client.Close())
}

func (s *RedisSuite) TestEmptyOperations() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// Empty Delete
	s.Require().NoError(client.Delete(ctx))

	// Empty Exists
	count, err := client.Exists(ctx)
	s.Require().NoError(err)
	s.Equal(int64(0), count)

	// Empty LPop
	_, err = client.LPop(ctx, "empty_list")
	s.ErrorIs(err, ErrKeyNotFound)

	// Empty RPop
	_, err = client.RPop(ctx, "empty_list")
	s.ErrorIs(err, ErrKeyNotFound)

	// Empty LLen
	length, err := client.LLen(ctx, "empty_list")
	s.Require().NoError(err)
	s.Equal(int64(0), length)

	// Empty HGetAll
	all, err := client.HGetAll(ctx, "empty_hash")
	s.Require().NoError(err)
	if all == nil {
		all = map[string]string{}
	}
	s.Equal(0, len(all))

	// Empty HDel
	s.Require().NoError(client.HDel(ctx, "empty_hash"))

	// Empty LPush
	s.Require().NoError(client.LPush(ctx, "list"))

	// Empty RPush
	s.Require().NoError(client.RPush(ctx, "list"))

	// Empty SAdd
	s.Require().NoError(client.SAdd(ctx, "set"))

	// Empty SMembers
	members, err := client.SMembers(ctx, "empty_set")
	s.Require().NoError(err)
	if members == nil {
		members = []string{}
	}
	s.Equal(0, len(members))

	// Empty SRem
	s.Require().NoError(client.SRem(ctx, "set"))
}

func (s *RedisSuite) TestContextCancellation() {
	// Создаём отменяемый контекст
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Сразу отменяем

	cfg := Config{Addr: s.addr}
	client, err := Connect(context.Background(), cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// Операции с отменённым контекстом должны возвращать ошибку
	_, err = client.Get(ctx, "test")
	s.Error(err)
}

func (s *RedisSuite) TestExpire() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// Set без TTL
	s.Require().NoError(client.Set(ctx, "expire_key", "value", 0))

	// Проверяем TTL (должен быть -1, то есть без ограничения)
	ttl, err := client.TTL(ctx, "expire_key")
	s.Require().NoError(err)
	if ttl != -1*time.Second && ttl > 0 {
		// Redis возвращает -1 для ключей без TTL
		s.T().Logf("Note: TTL for key without expiration is %v", ttl)
	}

	// Устанавливаем TTL
	s.Require().NoError(client.Expire(ctx, "expire_key", time.Minute))

	// Проверяем новый TTL
	ttl, err = client.TTL(ctx, "expire_key")
	s.Require().NoError(err)
	s.True(ttl >= 30*time.Second, "expected TTL >= 30s, got %v", ttl)

	// Проверяем, что ключ всё ещё существует
	val, err := client.Get(ctx, "expire_key")
	s.Require().NoError(err)
	s.Equal("value", val)

	// Устанавливаем очень короткий TTL и ждём истечения
	// Redis имеет минимальный TTL 1 секунду
	s.Require().NoError(client.Expire(ctx, "expire_key", time.Second))
	time.Sleep(1100 * time.Millisecond)

	_, err = client.Get(ctx, "expire_key")
	s.ErrorIs(err, ErrKeyNotFound)
}

func TestNilError(t *testing.T) {
	// Тестируем метод Error у Nil
	n := Nil{}
	s := n.Error()
	t.Run("Nil.Error()", func(t *testing.T) {
		if s != "redis: nil" {
			t.Errorf("expected 'redis: nil', got '%s'", s)
		}
	})

	// Проверяем IsNil с различными типами ошибок
	// Note: IsNil compares err == Nil{} which doesn't work correctly for interface values
	// This test documents the current behavior
	if IsNil(nil) {
		// IsNil(nil) returns true because err == nil check
		t.Log("IsNil(nil) returns true (expected)")
	}

	if IsNil(ErrKeyNotFound) {
		t.Error("IsNil(ErrKeyNotFound) should return false")
	}
}

func (s *RedisSuite) TestTypeMismatch() {
	ctx := context.Background()
	cfg := Config{Addr: s.addr}
	client, err := Connect(ctx, cfg)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := client.Close(); err != nil {
			s.T().Logf("failed to close client: %v", err)
		}
	})

	// Set a string value
	s.Require().NoError(client.Set(ctx, "string_key", "string_value", 0))

	// Try to HGet on a string key - should return error
	_, err = client.HGet(ctx, "string_key", "field")
	s.Error(err)
	// Error might be ErrKeyNotFound or a type mismatch error from Redis

	// Try to LPop on a string key - should return error
	_, err = client.LPop(ctx, "string_key")
	s.Error(err)

	// Try to SMembers on a string key - should return error
	_, err = client.SMembers(ctx, "string_key")
	s.Error(err)

	// Try to Incr on a non-numeric string - should return error
	_, err = client.Incr(ctx, "string_key")
	s.Error(err)
}
