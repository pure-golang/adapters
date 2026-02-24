// Rate limiter middleware protects HTTP endpoints from abuse using fixed window algorithm
package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config defines rate limiting behavior
type Config struct {
	// Window is the time window for rate limiting (e.g., 1m, 1h)
	Window time.Duration `envconfig:"RATE_LIMIT_WINDOW" default:"1m"`
	// MaxRequests is the maximum number of requests allowed per window per IP
	MaxRequests int `envconfig:"RATE_LIMIT_MAX_REQUESTS" default:"60"`
	// TrustProxy enables trusting X-Real-IP and X-Forwarded-For headers
	TrustProxy bool `envconfig:"RATE_LIMIT_TRUST_PROXY" default:"false"`
	// AddHeaders adds X-RateLimit-* headers to all responses
	AddHeaders bool `envconfig:"RATE_LIMIT_ADD_HEADERS" default:"true"`
	// CleanupInterval is the interval for cleaning up stale client entries
	CleanupInterval time.Duration `envconfig:"RATE_LIMIT_CLEANUP_INTERVAL" default:"5m"`
}

// clientTracker tracks request count for a single client
type clientTracker struct {
	count       int64
	windowStart int64
	mu          sync.RWMutex
}

// RateLimiter manages rate limiting for all clients
type RateLimiter struct {
	config      Config
	clients     sync.Map
	cleanupDone chan struct{}
	cleanupOnce sync.Once
	logger      *slog.Logger
}

// RateLimit returns a middleware function that implements rate limiting
// using the fixed window algorithm with in-memory storage.
func RateLimit(config Config) func(http.Handler) http.Handler {
	rl := &RateLimiter{
		config:      config,
		cleanupDone: make(chan struct{}),
		logger:      slog.Default(),
	}

	// Start cleanup goroutine to prevent memory leaks
	rl.startCleanup()

	// Return the middleware function
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP from request
			ip := extractIP(r, rl.config.TrustProxy)

			// Check if request is allowed
			allowed, remaining, resetTime := rl.checkLimit(ip)

			// Add rate limit headers if enabled
			if rl.config.AddHeaders {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.config.MaxRequests))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
			}

			// If rate limit exceeded, return 429
			if !allowed {
				retryAfter := calculateRetryAfter(resetTime)
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			// Request allowed, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// extractIP extracts the client IP address from the request
func extractIP(r *http.Request, trustProxy bool) string {
	// If trusting proxy, check X-Real-IP first
	if trustProxy {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return sanitizeIP(ip)
		}
		// Then check X-Forwarded-For (first IP is the original client)
		if ips := r.Header.Get("X-Forwarded-For"); ips != "" {
			if parts := strings.Split(ips, ","); len(parts) > 0 {
				return sanitizeIP(strings.TrimSpace(parts[0]))
			}
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If RemoteAddr is already just an IP (no port)
		return sanitizeIP(r.RemoteAddr)
	}
	return sanitizeIP(ip)
}

// sanitizeIP validates and returns a safe IP string
func sanitizeIP(ip string) string {
	// First try to split host:port format
	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		ip = host
	}

	// Parse the IP to ensure it's valid
	parsed := net.ParseIP(ip)
	if parsed == nil {
		// Return "unknown" for invalid IPs rather than failing
		return "unknown"
	}
	return parsed.String()
}

// calculateRetryAfter calculates the retry-after duration with fallback for negative values
// This is exported for testing purposes
func calculateRetryAfter(resetTime time.Time) time.Duration {
	retryAfter := time.Until(resetTime)
	if retryAfter < 0 {
		retryAfter = time.Second
	}
	return retryAfter
}

// checkLimit checks if the request from the given IP is allowed
// Returns: (allowed bool, remaining int, resetTime time.Time)
func (rl *RateLimiter) checkLimit(ip string) (bool, int, time.Time) {
	now := time.Now()
	windowStart := rl.getWindowStartAt(now)

	// Get or create tracker for this IP
	value, _ := rl.clients.LoadOrStore(ip, &clientTracker{
		windowStart: windowStart,
	})
	tracker := value.(*clientTracker)

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	// If window has expired, reset counter
	if tracker.windowStart != windowStart {
		tracker.count = 0
		tracker.windowStart = windowStart
	}

	// Check if limit exceeded
	if tracker.count >= int64(rl.config.MaxRequests) {
		resetTime := time.Unix(tracker.windowStart, 0).Add(rl.config.Window)
		return false, 0, resetTime
	}

	// Increment counter
	tracker.count++
	resetTime := time.Unix(tracker.windowStart, 0).Add(rl.config.Window)
	remaining := int(int64(rl.config.MaxRequests) - tracker.count)

	return true, remaining, resetTime
}

// getWindowStart calculates the start timestamp of the current window
func (rl *RateLimiter) getWindowStart() int64 {
	return rl.getWindowStartAt(time.Now())
}

// getWindowStartAt calculates the start timestamp of the window for a given time
func (rl *RateLimiter) getWindowStartAt(t time.Time) int64 {
	windowDuration := rl.config.Window
	if windowDuration < time.Second {
		// For sub-second windows, use milliseconds
		windowMs := int64(windowDuration.Milliseconds())
		return (t.UnixMilli() / windowMs) * windowMs
	}
	// For windows >= 1 second, use seconds
	windowSeconds := int64(windowDuration.Seconds())
	return (t.Unix() / windowSeconds) * windowSeconds
}

// startCleanup starts a background goroutine to clean up stale entries
func (rl *RateLimiter) startCleanup() {
	go rl.cleanup()
}

// cleanup periodically removes stale client tracker entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.removeStaleEntries()
		case <-rl.cleanupDone:
			return
		}
	}
}

// removeStaleEntries removes client trackers that are outside the current window
func (rl *RateLimiter) removeStaleEntries() {
	currentWindowStart := rl.getWindowStart()

	rl.clients.Range(func(key, value interface{}) bool {
		tracker := value.(*clientTracker)

		tracker.mu.RLock()
		isStale := tracker.windowStart < currentWindowStart-int64(rl.config.Window.Seconds())
		tracker.mu.RUnlock()

		if isStale {
			rl.clients.Delete(key)
		}

		return true // continue iteration
	})
}

// stopCleanup gracefully stops the cleanup goroutine
func (rl *RateLimiter) stopCleanup() {
	rl.cleanupOnce.Do(func() {
		close(rl.cleanupDone)
	})
}
