//go:build integration
// +build integration

package fcm

import (
	"context"
	"os"
	"testing"
)

// TestNewPusher_Integration is an integration test that requires real Firebase credentials.
// It's only run with the 'integration' build tag.
//
// To run: go test -tags=integration ./fcm/...
func TestNewPusher_Integration(t *testing.T) {
	ctx := context.Background()

	// Skip if no credentials file is set
	credentialsFile := os.Getenv("FCM_CREDENTIALS_FILE")
	if credentialsFile == "" {
		t.Skip("Skipping integration test: FCM_CREDENTIALS_FILE not set")
	}

	// Test with real Firebase credentials
	cfg := Config{
		CredentialsFile: credentialsFile,
	}

	pusher, err := NewPusher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPusher() failed: %v", err)
	}
	defer pusher.Close()

	if pusher == nil {
		t.Fatal("NewPusher() returned nil pusher")
	}
}

// TestNewPusher_WithJSON_Integration is an integration test with JSON credentials.
func TestNewPusher_WithJSON_Integration(t *testing.T) {
	ctx := context.Background()

	// Skip if no credentials JSON is set
	credentialsJSON := os.Getenv("FCM_CREDENTIALS_JSON")
	if credentialsJSON == "" {
		t.Skip("Skipping integration test: FCM_CREDENTIALS_JSON not set")
	}

	// Test with real Firebase credentials as JSON
	cfg := Config{
		CredentialsJSON: []byte(credentialsJSON),
	}

	pusher, err := NewPusher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPusher() failed: %v", err)
	}
	defer pusher.Close()

	if pusher == nil {
		t.Fatal("NewPusher() returned nil pusher")
	}
}
