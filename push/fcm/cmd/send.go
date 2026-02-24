package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	fcm "git.korputeam.ru/newbackend/adapters/push/fcm"
)

// Simple FCM test program
func main() {
	ctx := context.Background()

	// Check for credentials path
	credsPath := "credentials.json"
	if len(os.Args) > 1 {
		credsPath = os.Args[1]
	}

	pusher, err := fcm.NewPusher(ctx, fcm.Config{
		CredentialsFile: credsPath,
		ProjectID:       "", // Will be inferred from credentials
	})
	if err != nil {
		log.Fatalf("Failed to create pusher: %v", err)
	}

	if err := run(pusher); err != nil {
		_ = pusher.Close()
		os.Exit(1)
	}
	_ = pusher.Close()
}

func run(pusher *fcm.Pusher) error {
	ctx := context.Background()

	fmt.Println("=== FCM Test Program ===")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter FCM Token: ")
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "No token provided")
		return fmt.Errorf("no token provided")
	}

	token := strings.TrimSpace(scanner.Text())
	if token == "" {
		fmt.Fprintln(os.Stderr, "Token cannot be empty")
		return fmt.Errorf("token cannot be empty")
	}

	fmt.Printf("\nToken: %s...\n", token[:min(50, len(token))]) //nolint:revive // min is built-in in Go 1.21+

	// Get notification type
	fmt.Println("\nSelect notification type:")
	fmt.Println("1. Basic")
	fmt.Println("2. Android")
	fmt.Println("3. iOS")
	fmt.Println("4. Unified")
	fmt.Print("Choice (1-4): ")

	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "No choice provided")
		return fmt.Errorf("no choice provided")
	}

	choice := strings.TrimSpace(scanner.Text())

	var notif fcm.Notification

	switch choice {
	case "1":
		notif = fcm.Notification{
			Token: token,
			Title: "✅ Success!",
			Body:  "FCM is working correctly",
			Sound: "default",
		}
	case "2":
		notif = fcm.Notification{
			Token:           token,
			Title:           "🤖 Android Test",
			Body:            "Android-specific notification",
			AndroidPriority: "high",
			AndroidChannel:  "test",
			Sound:           "default",
		}
	case "3":
		notif = fcm.Notification{
			Token:       token,
			Title:       "🍎 iOS Test",
			Body:        "iOS-specific notification",
			IOSBadge:    1,
			IOSCategory: "TEST",
			IOSSound:    "default",
		}
	case "4":
		notif = fcm.Notification{
			Title: "🌐 Unified Test",
			Body:  "Works on all platforms!",
			Sound: "default",

			AndroidPriority: "high",
			AndroidChannel:  "unified",

			IOSBadge:    1,
			IOSCategory: "UNIFIED",
		}
	default:
		fmt.Fprintln(os.Stderr, "Invalid choice")
		return fmt.Errorf("invalid choice")
	}

	fmt.Println("\n📤 Sending notification...")

	err := pusher.Push(ctx, notif)

	if err != nil {
		log.Fatalf("Failed to send: %v", err)
	}

	fmt.Println("✅ Success! Check your device for the notification.")
	fmt.Println()
	fmt.Println("If notification not received:")
	fmt.Println("  - Check the page is open: http://localhost:8080/web_push_test.html")
	fmt.Println("  - Allow notifications in browser")
	fmt.Println("  - Check browser console (F12) for errors")
	fmt.Println("  - Try getting a new token from the web page")
	return nil
}
