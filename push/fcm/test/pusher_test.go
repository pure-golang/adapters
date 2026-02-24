package fcm_test

import (
	"context"
	"os"
	"testing"

	"github.com/pure-golang/adapters/push/fcm"
)

// TestNewPusher проверяет создание Pusher с реальными Firebase credentials.
// Требует переменной окружения FCM_CREDENTIALS_FILE.
func TestNewPusher(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	credentialsFile := os.Getenv("FCM_CREDENTIALS_FILE")
	if credentialsFile == "" {
		t.Skip("FCM_CREDENTIALS_FILE not set")
	}

	ctx := context.Background()

	cfg := fcm.Config{
		CredentialsFile: credentialsFile,
	}

	pusher, err := fcm.NewPusher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPusher() failed: %v", err)
	}
	t.Cleanup(func() { pusher.Close() })

	if pusher == nil {
		t.Fatal("NewPusher() returned nil pusher")
	}
}

// TestNewPusher_WithJSON проверяет создание Pusher с JSON-credentials.
// Требует переменной окружения FCM_CREDENTIALS_JSON.
func TestNewPusher_WithJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	credentialsJSON := os.Getenv("FCM_CREDENTIALS_JSON")
	if credentialsJSON == "" {
		t.Skip("FCM_CREDENTIALS_JSON not set")
	}

	ctx := context.Background()

	cfg := fcm.Config{
		CredentialsJSON: []byte(credentialsJSON),
	}

	pusher, err := fcm.NewPusher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPusher() failed: %v", err)
	}
	t.Cleanup(func() { pusher.Close() })

	if pusher == nil {
		t.Fatal("NewPusher() returned nil pusher")
	}
}
