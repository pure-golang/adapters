package minio_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pure-golang/adapters/storage/minio"
)

// TestClient_ConnectionTimeout tests that connection timeout is respected.
func TestClient_ConnectionTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	// Use a non-routable IP to trigger timeout
	cfg := minio.Config{
		Endpoint:  "192.0.2.1:9000", // TEST-NET-1, should never route
		AccessKey: "test",
		SecretKey: "test",
		Timeout:   1, // 1 second timeout
	}

	start := time.Now()
	_, err := minio.NewClient(cfg)
	elapsed := time.Since(start)

	assert.Error(t, err)
	// Should timeout within a reasonable time (allow some margin)
	assert.Less(t, elapsed, 5*time.Second)
}
