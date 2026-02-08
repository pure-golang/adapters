package noop

import (
	"context"
	"testing"
	"time"
)

func TestStoreMethods(t *testing.T) {
	ctx := context.Background()
	s := NewStore()

	// Get
	val, err := s.Get(ctx, "test")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if val != "" {
		t.Fatalf("Get returned unexpected value: %s", val)
	}

	// Set
	if err := s.Set(ctx, "test", "value", 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	// Delete
	if err := s.Delete(ctx, "test"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// Exists
	count, err := s.Exists(ctx, "test")
	if err != nil {
		t.Fatalf("Exists returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("Exists returned unexpected count: %d", count)
	}

	// Incr
	n, err := s.Incr(ctx, "counter")
	if err != nil {
		t.Fatalf("Incr returned error: %v", err)
	}
	if n != 0 {
		t.Fatalf("Incr returned unexpected value: %d", n)
	}

	// Decr
	n, err = s.Decr(ctx, "counter")
	if err != nil {
		t.Fatalf("Decr returned error: %v", err)
	}
	if n != 0 {
		t.Fatalf("Decr returned unexpected value: %d", n)
	}

	// Expire
	if err := s.Expire(ctx, "test", time.Hour); err != nil {
		t.Fatalf("Expire returned error: %v", err)
	}

	// TTL
	ttl, err := s.TTL(ctx, "test")
	if err != nil {
		t.Fatalf("TTL returned error: %v", err)
	}
	if ttl != 0 {
		t.Fatalf("TTL returned unexpected value: %v", ttl)
	}

	// HGet
	val, err = s.HGet(ctx, "hash", "field")
	if err != nil {
		t.Fatalf("HGet returned error: %v", err)
	}
	if val != "" {
		t.Fatalf("HGet returned unexpected value: %s", val)
	}

	// HSet
	if err := s.HSet(ctx, "hash", "field", "value"); err != nil {
		t.Fatalf("HSet returned error: %v", err)
	}

	// HGetAll
	m, err := s.HGetAll(ctx, "hash")
	if err != nil {
		t.Fatalf("HGetAll returned error: %v", err)
	}
	if m == nil {
		m = map[string]string{}
	}
	if len(m) != 0 {
		t.Fatalf("HGetAll returned unexpected map: %v", m)
	}

	// HDel
	if err := s.HDel(ctx, "hash", "field"); err != nil {
		t.Fatalf("HDel returned error: %v", err)
	}

	// LPush
	if err := s.LPush(ctx, "list", "value"); err != nil {
		t.Fatalf("LPush returned error: %v", err)
	}

	// RPush
	if err := s.RPush(ctx, "list", "value"); err != nil {
		t.Fatalf("RPush returned error: %v", err)
	}

	// LPop
	val, err = s.LPop(ctx, "list")
	if err != nil {
		t.Fatalf("LPop returned error: %v", err)
	}
	if val != "" {
		t.Fatalf("LPop returned unexpected value: %s", val)
	}

	// RPop
	val, err = s.RPop(ctx, "list")
	if err != nil {
		t.Fatalf("RPop returned error: %v", err)
	}
	if val != "" {
		t.Fatalf("RPop returned unexpected value: %s", val)
	}

	// LLen
	length, err := s.LLen(ctx, "list")
	if err != nil {
		t.Fatalf("LLen returned error: %v", err)
	}
	if length != 0 {
		t.Fatalf("LLen returned unexpected length: %d", length)
	}

	// SAdd
	if err := s.SAdd(ctx, "set", "member"); err != nil {
		t.Fatalf("SAdd returned error: %v", err)
	}

	// SMembers
	members, err := s.SMembers(ctx, "set")
	if err != nil {
		t.Fatalf("SMembers returned error: %v", err)
	}
	if members == nil {
		members = []string{}
	}
	if len(members) != 0 {
		t.Fatalf("SMembers returned unexpected members: %v", members)
	}

	// SIsMember
	isMember, err := s.SIsMember(ctx, "set", "member")
	if err != nil {
		t.Fatalf("SIsMember returned error: %v", err)
	}
	if isMember {
		t.Fatalf("SIsMember returned true")
	}

	// SRem
	if err := s.SRem(ctx, "set", "member"); err != nil {
		t.Fatalf("SRem returned error: %v", err)
	}

	// Ping
	if err := s.Ping(ctx); err != nil {
		t.Fatalf("Ping returned error: %v", err)
	}

	// Close
	if err := s.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}
