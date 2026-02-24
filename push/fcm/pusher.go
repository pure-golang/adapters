package fcm

import (
	"context"
	"fmt"
	"time"

	firebaseLib "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// messagingClient defines the interface for FCM messaging operations
type messagingClient interface {
	Send(ctx context.Context, msg *messaging.Message) (string, error)
}

// firebaseAppInterface defines the interface for Firebase app operations
type firebaseAppInterface interface {
	Messaging(ctx context.Context) (messagingClient, error)
}

// firebaseAppFactory defines the interface for creating Firebase apps
type firebaseAppFactory interface {
	NewApp(ctx context.Context, config *firebaseLib.Config, opts ...option.ClientOption) (firebaseAppInterface, error)
}

// Notification represents a push notification message for FCM.
// Supports unified messaging for Android, iOS, and Web platforms.
// Firebase automatically determines the platform based on the token format.
type Notification struct {
	// Target (only one should be set)
	Token     string // Device token (Android, iOS, or Web)
	Topic     string // Topic for multicast messaging
	Condition string // Condition for topic-based messaging

	// Content
	Title string            // Notification title
	Body  string            // Notification body
	Data  map[string]string // Custom data payload
	Image string            // Optional image URL

	// Sound
	Sound string // Sound name (default: "default")

	// Android-specific options
	AndroidPriority string // "high" or "normal" (default: "high")
	AndroidTTL      string // Time to live (e.g., "3600s")
	AndroidChannel  string // Notification channel ID (default: "default")
	CollapseKey     string // Collapse key for grouping notifications

	// iOS-specific options
	IOSCategory         string // Category for actionable notifications
	IOSSound            string // Sound name (overrides Sound if set)
	IOSBadge            int    // Badge count (0 to clear, -1 for unchanged)
	IOSContentAvailable bool   // Enable silent notifications with background update
}

// realMessagingClient wraps the real Firebase messaging.Client
type realMessagingClient struct {
	client *messaging.Client
}

func (r *realMessagingClient) Send(ctx context.Context, msg *messaging.Message) (string, error) {
	return r.client.Send(ctx, msg)
}

// realFirebaseApp wraps the real Firebase App
type realFirebaseApp struct {
	app *firebaseLib.App
}

func (r *realFirebaseApp) Messaging(ctx context.Context) (messagingClient, error) {
	client, err := r.app.Messaging(ctx)
	if err != nil {
		return nil, err
	}
	return &realMessagingClient{client: client}, nil
}

// realFirebaseAppFactory is the default implementation using real Firebase
type realFirebaseAppFactory struct{}

func (f *realFirebaseAppFactory) NewApp(ctx context.Context, config *firebaseLib.Config, opts ...option.ClientOption) (firebaseAppInterface, error) {
	app, err := firebaseLib.NewApp(ctx, config, opts...)
	if err != nil {
		return nil, err
	}
	return &realFirebaseApp{app: app}, nil
}

// Pusher sends push notifications via Firebase Cloud Messaging.
type Pusher struct {
	client messagingClient
	app    firebaseAppInterface
}

// Config holds the configuration for FCM Pusher.
type Config struct {
	CredentialsFile string // Path to service account JSON file
	CredentialsJSON []byte // Service account JSON as byte slice
	ProjectID       string // GCP project ID (optional, inferred from credentials)
}

// defaultFactory is the default Firebase app factory
var defaultFactory firebaseAppFactory = &realFirebaseAppFactory{}

// NewPusher creates a new FCM Pusher.
func NewPusher(ctx context.Context, cfg Config) (*Pusher, error) {
	return NewPusherWithFactory(ctx, cfg, defaultFactory)
}

// NewPusherWithFactory creates a new FCM Pusher with a custom factory for testing.
func NewPusherWithFactory(ctx context.Context, cfg Config, factory firebaseAppFactory) (*Pusher, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	var opts []option.ClientOption
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	} else if len(cfg.CredentialsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(cfg.CredentialsJSON))
	}

	// Initialize Firebase app
	app, err := factory.NewApp(ctx, &firebaseLib.Config{
		ProjectID: cfg.ProjectID,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firebase app: %w", err)
	}

	// Get messaging client
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create FCM client: %w", err)
	}

	return &Pusher{
		client: client,
		app:    app,
	}, nil
}

// Push sends notifications to devices via FCM.
func (p *Pusher) Push(ctx context.Context, notifications ...Notification) error {
	if len(notifications) == 0 {
		return nil
	}

	// Send each notification individually
	for _, n := range notifications {
		if err := p.pushSingle(ctx, n); err != nil {
			return err
		}
	}

	return nil
}

// pushSingle sends a single notification.
func (p *Pusher) pushSingle(ctx context.Context, n Notification) error {
	if n.Token == "" && n.Topic == "" && n.Condition == "" {
		return fmt.Errorf("notification must have either Token, Topic, or Condition")
	}

	msg, err := p.buildMessage(n)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	// Send the message
	_, err = p.client.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Close closes the FCM client.
func (p *Pusher) Close() error {
	return nil
}

// buildMessage converts a Notification to FCM message format.
// Supports unified messaging for Android, iOS, and Web platforms.
func (p *Pusher) buildMessage(n Notification) (*messaging.Message, error) {
	msg := &messaging.Message{
		Data: convertData(n.Data),
	}

	// Set target
	switch {
	case n.Token != "":
		msg.Token = n.Token
	case n.Topic != "":
		msg.Topic = n.Topic
	case n.Condition != "":
		msg.Condition = n.Condition
	default:
		return nil, fmt.Errorf("message must have Token, Topic, or Condition")
	}

	// Set base notification (cross-platform)
	if n.Title != "" || n.Body != "" || n.Image != "" {
		msg.Notification = &messaging.Notification{
			Title:    n.Title,
			Body:     n.Body,
			ImageURL: n.Image,
		}
	}

	// Set Android config
	msg.Android = p.buildAndroidConfig(n)

	// Set APNS config for iOS
	msg.APNS = p.buildAPNSConfig(n)

	return msg, nil
}

// buildAndroidConfig creates Android-specific configuration.
func (p *Pusher) buildAndroidConfig(n Notification) *messaging.AndroidConfig {
	config := &messaging.AndroidConfig{
		Priority: p.getAndroidPriority(n.AndroidPriority),
	}

	// Set TTL if specified
	if n.AndroidTTL != "" {
		if duration, err := time.ParseDuration(n.AndroidTTL); err == nil {
			config.TTL = &duration
		}
	}

	// Set collapse key for grouping
	if n.CollapseKey != "" {
		config.CollapseKey = n.CollapseKey
	}

	// Set notification if title or body is provided
	if n.Title != "" || n.Body != "" || n.Image != "" {
		sound := n.Sound
		if sound == "" {
			sound = "default"
		}

		channelID := n.AndroidChannel
		if channelID == "" {
			channelID = "default"
		}

		config.Notification = &messaging.AndroidNotification{
			Title:     n.Title,
			Body:      n.Body,
			ImageURL:  n.Image,
			Sound:     sound,
			ChannelID: channelID,
		}
	}

	return config
}

// buildAPNSConfig creates iOS-specific configuration.
func (p *Pusher) buildAPNSConfig(n Notification) *messaging.APNSConfig {
	aps := &messaging.Aps{
		Sound:            p.getIOSSound(n),
		ContentAvailable: n.IOSContentAvailable,
	}

	// Set badge count
	if n.IOSBadge >= 0 {
		aps.Badge = &n.IOSBadge
	}

	// Set alert if title or body is provided
	if n.Title != "" || n.Body != "" {
		aps.Alert = &messaging.ApsAlert{
			Title: n.Title,
			Body:  n.Body,
		}
	}

	// Set category if specified
	if n.IOSCategory != "" {
		aps.Category = n.IOSCategory
	}

	return &messaging.APNSConfig{
		Payload: &messaging.APNSPayload{
			Aps: aps,
		},
	}
}

// getAndroidPriority converts string priority to Android priority string.
func (p *Pusher) getAndroidPriority(priority string) string {
	switch priority {
	case "high", "":
		return "high"
	case "normal":
		return "normal"
	default:
		return "high" // default to high for better delivery
	}
}

// getIOSSound returns the sound name for iOS notifications.
func (p *Pusher) getIOSSound(n Notification) string {
	if n.IOSSound != "" {
		return n.IOSSound
	}
	if n.Sound != "" {
		return n.Sound
	}
	return "default"
}

// newPusherWithClient creates a Pusher with a custom messaging client for testing.
func newPusherWithClient(client messagingClient) *Pusher {
	return &Pusher{client: client}
}

// newPusherWithDeps creates a Pusher with all custom dependencies for testing.
func newPusherWithDeps(app firebaseAppInterface, client messagingClient) *Pusher {
	return &Pusher{
		client: client,
		app:    app,
	}
}

// validateConfig validates the FCM configuration.
func validateConfig(cfg Config) error {
	if cfg.CredentialsFile == "" && len(cfg.CredentialsJSON) == 0 {
		return fmt.Errorf("either CredentialsFile or CredentialsJSON must be provided")
	}
	return nil
}

// convertData converts data map to FCM format.
func convertData(data map[string]string) map[string]string {
	if data == nil {
		return make(map[string]string)
	}
	return data
}

// NoopPusher is a no-op push notification sender for testing.
type NoopPusher struct{}

// NewNoopPusher creates a new no-op Pusher.
func NewNoopPusher() *NoopPusher {
	return &NoopPusher{}
}

// Push silently discards notifications.
func (n *NoopPusher) Push(ctx context.Context, notifications ...Notification) error {
	for _, notification := range notifications {
		_ = notification // Discard
	}
	return nil
}

// Close is a no-op for compatibility.
func (n *NoopPusher) Close() error {
	return nil
}
