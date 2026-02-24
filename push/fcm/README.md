# FCM Push Adapter

Universal push notification adapter for Firebase Cloud Messaging (FCM) with unified messaging support for Android, iOS, and Web platforms.

## Quick Start

### 1. Setup (one-time)

```bash
cd adapters/push/fcm
./setup.sh
```

This will:
- Enable required APIs
- Create service account
- Grant IAM roles
- Generate credentials file

### 2. Initialize in Go

```go
import (
    "context"
    fcm "git.korputeam.ru/newbackend/adapters/push/fcm"
)

pusher, err := fcm.NewPusher(ctx, fcm.Config{
    CredentialsFile: "credentials.json",
    ProjectID:       "YOUR_PROJECT_ID",
})
defer pusher.Close()
```

### 3. Send Notification

```go
err := pusher.Push(ctx, fcm.Notification{
    Token: "DEVICE_TOKEN",
    Title: "Hello!",
    Body:  "World",
})
```

## Features

- ✅ **Unified Messaging** - Single notification works on all platforms
- ✅ **Platform-Specific Config** - Separate Android and iOS options
- ✅ **Automatic Platform Detection** - Firebase detects platform from token
- ✅ **Multiple Targets** - Device token, topic, or conditional messaging
- ✅ **Rich Notifications** - Title, body, image, sound, custom data
- ✅ **Graceful Degradation** - NoopPusher for testing

## Configuration

### Credentials

```go
// From file
pusher, err := fcm.NewPusher(ctx, fcm.Config{
    CredentialsFile: "path/to/credentials.json",
})

// From JSON (e.g., environment variable)
pusher, err := fcm.NewPusher(ctx, fcm.Config{
    CredentialsJSON: []byte(os.Getenv("FIREBASE_CREDENTIALS")),
    ProjectID:       os.Getenv("FIREBASE_PROJECT_ID"),
})
```

Get credentials via the setup script:
```bash
cd adapters/push/fcm
./setup.sh
```

## Notification Structure

### Basic (All Platforms)
```go
fcm.Notification{
    Token: "device-token",
    Title: "Hello!",
    Body:  "Works on Android, iOS, and Web!",
}
```

### Android-Specific
```go
fcm.Notification{
    Token:           "android-token",
    Title:           "Android",
    Body:            "Android notification",
    AndroidPriority: "high",          // "high" or "normal"
    AndroidChannel:  "notifications",  // Notification channel
    AndroidTTL:      "3600s",         // Time to live
    CollapseKey:     "group_id",      // Group notifications
    Sound:           "default",
}
```

### iOS-Specific
```go
fcm.Notification{
    Token:       "ios-token",
    Title:       "iOS",
    Body:        "iOS notification",
    IOSBadge:    5,                  // Badge count (0 to clear)
    IOSCategory: "MESSAGE",          // Actionable category
    IOSSound:    "default",          // Sound
    IOSContentAvailable: false,      // Background notification
}
```

### Unified (Android + iOS + Web)
```go
fcm.Notification{
    Title: "Cross-platform",
    Body:  "Works everywhere!",
    Sound: "default",

    // Android
    AndroidPriority: "high",
    AndroidChannel:  "messages",
    AndroidTTL:      "86400s",
    CollapseKey:     "msg_123",

    // iOS
    IOSBadge:        1,
    IOSCategory:     "NEW_MESSAGE",
}
```

## Reference

### Notification Fields

| Field | Type | Platform | Default |
|-------|------|----------|---------|
| **Target** | | | |
| `Token` | string | All | - |
| `Topic` | string | All | - |
| `Condition` | string | All | - |
| **Content** | | | |
| `Title` | string | All | - |
| `Body` | string | All | - |
| `Image` | string | All | - |
| `Sound` | string | All | "default" |
| `Data` | map[string]string | All | - |
| **Android** | | | |
| `AndroidPriority` | string | Android | "high" |
| `AndroidChannel` | string | Android | "default" |
| `AndroidTTL` | string | Android | - |
| `CollapseKey` | string | Android | - |
| **iOS** | | | |
| `IOSBadge` | int | iOS | 0 |
| `IOSCategory` | string | iOS | - |
| `IOSSound` | string | iOS | - |
| `IOSContentAvailable` | bool | iOS | false |

## Topic Messaging

```go
// Send to topic
pusher.Push(ctx, fcm.Notification{
    Topic: "updates",
    Title: "New Update",
    Body:  "Version 2.0 available",
})

// Conditional topic
pusher.Push(ctx, fcm.Notification{
    Condition: "'tech' in topics || 'news' in topics",
    Title: "Breaking News",
    Body: "Important announcement",
})
```

## Testing

### Prerequisites
- GCP project with Firebase enabled
- Service account with Firebase Admin SDK role
- FCM device token (from web/mobile app)

### Get FCM Token (Web)

1. Run `./setup.sh` to create credentials
2. Complete Firebase integration (see script output)
3. Create Web app in Firebase Console
4. Run test server and get token from web page

### Send Test

```go
ctx := context.Background()
pusher, _ := fcm.NewPusher(ctx, fcm.Config{
    CredentialsFile: "credentials.json",
    ProjectID:       "YOUR_PROJECT_ID",
})

pusher.Push(ctx, fcm.Notification{
    Token: "YOUR_FCM_TOKEN",
    Title: "Test",
    Body:  "Test notification",
})
```

## Usage Examples

See `examples/` directory for more examples:
- Basic notification
- Platform-specific (Android, iOS)
- Unified messaging
- Topic messaging
- Batch sending
- With images, data, badges

## Security

- ⚠️ **Never commit `credentials.json` to version control**
- Add to `.gitignore`: `credentials.json`
- Use environment variables in production
- Store credentials securely (e.g., Vault, Secrets Manager)

## API Versions

This adapter uses Firebase Admin SDK v4:
- `firebase.google.com/go/v4`
- `firebase.google.com/go/v4/messaging`

## License

Internal use only.
