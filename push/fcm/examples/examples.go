package fcm_test

import (
	"context"
	"fmt"
	"log"

	"git.korputeam.ru/newbackend/adapters/push/fcm"
)

// Example_usage demonstrates how to use the FCM pusher with unified messaging.
func Example_usage() {
	ctx := context.Background()

	// Initialize pusher
	pusher, err := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	if err != nil {
		log.Fatalf("Failed to create pusher: %v", err)
	}
	defer pusher.Close()

	// Simple notification (works on all platforms)
	err = pusher.Push(ctx, fcm.Notification{
		Token: "device-fcm-token",
		Title: "Hello!",
		Body:  "This is a test notification",
	})
	if err != nil {
		log.Printf("Failed to send: %v", err)
	}
}

// Example_androidNotification demonstrates Android-specific configuration.
func Example_androidNotification() {
	ctx := context.Background()
	pusher, _ := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	defer pusher.Close()

	err := pusher.Push(ctx, fcm.Notification{
		Token:           "android-fcm-token",
		Title:           "Android Notification",
		Body:            "This is Android-specific",
		AndroidPriority: "high",
		AndroidChannel:  "alerts",
		AndroidTTL:      "3600s",
		Sound:           "default",
		Data: map[string]string{
			"type":     "alert",
			"priority": "high",
		},
	})
	if err != nil {
		log.Printf("Failed: %v", err)
	}
}

// Example_iosNotification demonstrates iOS-specific configuration.
func Example_iosNotification() {
	ctx := context.Background()
	pusher, _ := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	defer pusher.Close()

	err := pusher.Push(ctx, fcm.Notification{
		Token:       "ios-fcm-token",
		Title:       "iOS Notification",
		Body:        "This is iOS-specific",
		IOSBadge:    1,
		IOSCategory: "MESSAGE",
		IOSSound:    "default",
		Data: map[string]string{
			"type": "message",
		},
	})
	if err != nil {
		log.Printf("Failed: %v", err)
	}
}

// Example_unifiedNotification demonstrates a notification that works on both Android and iOS.
// Firebase automatically determines the platform based on the token format.
func Example_unifiedNotification() {
	ctx := context.Background()
	pusher, _ := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	defer pusher.Close()

	// This single notification works on BOTH Android and iOS!
	// Firebase SDK automatically applies the correct platform-specific config.
	err := pusher.Push(ctx, fcm.Notification{
		Title: "New Message",
		Body:  "You have a new message from John",
		Image: "https://example.com/avatar.jpg",
		Sound: "default",
		Data: map[string]string{
			"sender_id":    "123",
			"message_id":   "456",
			"conversation": "general",
		},
		// Android-specific settings
		AndroidPriority: "high",
		AndroidChannel:  "messages",
		AndroidTTL:      "86400s", // 24 hours
		CollapseKey:     "messages_123",
		// iOS-specific settings
		IOSBadge:            5,
		IOSCategory:         "NEW_MESSAGE",
		IOSContentAvailable: false, // Foreground notification
	})
	if err != nil {
		log.Printf("Failed: %v", err)
	}
}

// Example_batchNotifications demonstrates sending multiple notifications.
func Example_batchNotifications() {
	ctx := context.Background()
	pusher, _ := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	defer pusher.Close()

	notifications := []fcm.Notification{
		{
			Token: "token-1",
			Title: "Notification 1",
			Body:  "First notification",
		},
		{
			Token: "token-2",
			Title: "Notification 2",
			Body:  "Second notification",
		},
		{
			Token: "token-3",
			Title: "Notification 3",
			Body:  "Third notification",
		},
	}

	err := pusher.Push(ctx, notifications...)
	if err != nil {
		log.Printf("Failed to send batch: %v", err)
	}
}

// Example_topicMessaging demonstrates sending to a topic.
func Example_topicMessaging() {
	ctx := context.Background()
	pusher, _ := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	defer pusher.Close()

	// Send to all users subscribed to "updates" topic
	err := pusher.Push(ctx, fcm.Notification{
		Topic: "updates",
		Title: "System Update",
		Body:  "A new version is available",
		Data: map[string]string{
			"version": "2.0.0",
			"url":     "https://example.com/download",
		},
	})
	if err != nil {
		log.Printf("Failed: %v", err)
	}
}

// Example_conditionMessaging demonstrates conditional topic messaging.
func Example_conditionMessaging() {
	ctx := context.Background()
	pusher, _ := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	defer pusher.Close()

	// Send to users who are subscribed to EITHER 'tech' OR 'news' topics
	err := pusher.Push(ctx, fcm.Notification{
		Condition: "'tech' in topics || 'news' in topics",
		Title:     "Breaking News",
		Body:      "Important announcement",
	})
	if err != nil {
		log.Printf("Failed: %v", err)
	}
}

// Example_silentNotification demonstrates a silent/background notification (iOS).
func Example_silentNotification() {
	ctx := context.Background()
	pusher, _ := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	defer pusher.Close()

	// Silent notification for background data sync
	err := pusher.Push(ctx, fcm.Notification{
		Token:               "ios-token",
		IOSContentAvailable: true, // Enables background fetch
		Data: map[string]string{
			"sync_type": "full",
			"timestamp": "1234567890",
		},
	})
	if err != nil {
		log.Printf("Failed: %v", err)
	}
}

// Example_badgeCount demonstrates updating iOS badge count.
func Example_badgeCount() {
	ctx := context.Background()
	pusher, _ := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	defer pusher.Close()

	// Set badge to 5
	err := pusher.Push(ctx, fcm.Notification{
		Token:    "ios-token",
		Title:    "5 unread messages",
		Body:     "You have 5 unread messages",
		IOSBadge: 5,
	})
	if err != nil {
		log.Printf("Failed: %v", err)
	}

	// Clear badge (set to 0)
	err = pusher.Push(ctx, fcm.Notification{
		Token:    "ios-token",
		Title:    "Messages read",
		IOSBadge: 0,
	})
	if err != nil {
		log.Printf("Failed: %v", err)
	}
}

// Example_withCredentialsJSON demonstrates using credentials from a byte slice.
func Example_withCredentialsJSON() {
	ctx := background()

	// Useful when credentials come from environment variables or secret management
	credentialsJSON := []byte(`{
		"type": "service_account",
		"project_id": "your-project-id",
		"private_key_id": "...",
		"private_key": "...",
		"client_email": "...",
		...
	}`)

	pusher, err := fcm.NewPusher(ctx, fcm.Config{
		CredentialsJSON: credentialsJSON,
		ProjectID:       "your-project-id",
	})
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	defer pusher.Close()

	// Use pusher...
}

func background() context.Context {
	return context.Background()
}

// Example_realWorldNotification demonstrates a real-world notification scenario.
func Example_realWorldNotification() {
	ctx := context.Background()
	pusher, _ := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: "firebase-admin.json",
	})
	defer pusher.Close()

	// Scenario: New like on user's post
	notification := fcm.Notification{
		Title: "New Like!",
		Body:  "John Doe liked your post",
		Image: "https://example.com/avatars/john.jpg",
		Data: map[string]string{
			"type":       "new_like",
			"actor_id":   "123",
			"actor_name": "John Doe",
			"post_id":    "456",
			"like_count": "42",
			"timestamp":  "1704067200",
		},
		// Android settings
		AndroidPriority: "high",
		AndroidChannel:  "social",
		Sound:           "default",
		// iOS settings
		IOSBadge:    1,
		IOSCategory: "NEW_LIKE",
	}

	// The same notification works for:
	// - Android devices (Android config applied)
	// - iOS devices (APNS config applied)
	// - Web browsers (Webpush config would be applied)
	// Firebase determines the platform automatically from the token!

	err := pusher.Push(ctx, notification)
	if err != nil {
		log.Printf("Failed to send like notification: %v", err)
	} else {
		fmt.Println("Notification sent successfully to all platforms!")
	}
}
