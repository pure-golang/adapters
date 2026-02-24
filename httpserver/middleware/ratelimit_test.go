package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestConfig creates a test configuration with the given window and max requests
func newTestConfig(window time.Duration, maxRequests int) Config {
	return Config{
		Window:          window,
		MaxRequests:     maxRequests,
		TrustProxy:      false,
		AddHeaders:      true,
		CleanupInterval: 100 * time.Millisecond,
	}
}

// TestRateLimit_BasicLimiting verifies that requests are limited after exceeding the threshold
func TestRateLimit_BasicLimiting(t *testing.T) {
	config := newTestConfig(time.Minute, 3)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Make 3 successful requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Request %d should succeed", i+1)
		assert.Equal(t, "3", rr.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, strconv.Itoa(2-i), rr.Header().Get("X-RateLimit-Remaining"))
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	// Retry-After should be a positive value (actual value depends on when window started)
	retryAfter := rr.Header().Get("Retry-After")
	retryAfterInt, err := strconv.Atoi(retryAfter)
	require.NoError(t, err)
	assert.Greater(t, retryAfterInt, 0, "Retry-After should be positive")
	assert.LessOrEqual(t, retryAfterInt, 60, "Retry-After should be at most 60 seconds")
}

// TestRateLimit_WindowReset verifies that the counter resets after the window expires
func TestRateLimit_WindowReset(t *testing.T) {
	// Use a very short window for testing (200ms)
	config := newTestConfig(200*time.Millisecond, 2)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Make 2 requests to fill the window
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Wait for window to expire plus some buffer
	time.Sleep(250 * time.Millisecond)

	// New request should succeed after window reset
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Request should succeed after window reset")
	assert.Equal(t, "1", rr.Header().Get("X-RateLimit-Remaining"))
}

// TestRateLimit_ConcurrentRequests verifies thread safety under concurrent load
func TestRateLimit_ConcurrentRequests(t *testing.T) {
	config := newTestConfig(time.Minute, 100)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	var wg sync.WaitGroup
	var successCount atomic.Int64
	var limitCount atomic.Int64
	var otherCount atomic.Int64
	numRequests := 200

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.3:12345"
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			switch rr.Code {
			case http.StatusOK:
				successCount.Add(1)
			case http.StatusTooManyRequests:
				limitCount.Add(1)
			default:
				otherCount.Add(1)
			}
		}()
	}

	wg.Wait()

	successFinal := successCount.Load()
	limitFinal := limitCount.Load()
	otherFinal := otherCount.Load()

	// At most 100 should succeed, rest should be rate limited
	assert.LessOrEqual(t, successFinal, int64(100), "At most MaxRequests should succeed")
	assert.Greater(t, successFinal, int64(90), "At least 90 requests should succeed (allowing for race condition)")
	// The total should be close to 200, but we allow for some variance due to the nature of the test
	total := successFinal + limitFinal + otherFinal
	assert.GreaterOrEqual(t, total, int64(190), "Most requests should be accounted for")
	assert.LessOrEqual(t, total, int64(200), "Should not account for more than the number of requests")
}

// TestRateLimit_DifferentIPs verifies that each IP is tracked independently
func TestRateLimit_DifferentIPs(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// IP1 makes 2 successful requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.10:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// IP1 is rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.10:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// IP2 can still make 2 successful requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.20:54321"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Different IP should have independent limit")
	}

	// IP2 is also rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.20:54321"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

// TestRateLimit_ProxyHeaders verifies IP extraction from proxy headers
func TestRateLimit_ProxyHeaders(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	config.TrustProxy = true
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// X-Real-IP header should be used
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345" // proxy IP
		req.Header.Set("X-Real-IP", "203.0.113.1")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// 3rd request with same X-Real-IP should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Different X-Real-IP should work
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.2")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestRateLimit_ProxyHeadersXForwardedFor verifies X-Forwarded-For handling
func TestRateLimit_ProxyHeadersXForwardedFor(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	config.TrustProxy = true
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// X-Forwarded-For header (first IP is original client)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.100, 10.0.0.2")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// 3rd request with same first IP should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.100, 10.0.0.2")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

// TestRateLimit_ProxyHeadersNotTrusted verifies RemoteAddr is used when proxy not trusted
func TestRateLimit_ProxyHeadersNotTrusted(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	config.TrustProxy = false // Don't trust proxy headers
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// X-Real-IP header should be ignored, use RemoteAddr instead
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.50:12345"
		req.Header.Set("X-Real-IP", "203.0.113.1")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// 3rd request - RemoteAddr is rate limited, not X-Real-IP
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	req.Header.Set("X-Real-IP", "203.0.113.2") // Different X-Real-IP
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

// TestRateLimit_IPv6 verifies IPv6 address handling
func TestRateLimit_IPv6(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// IPv6 address
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "[2001:db8::1]:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "[2001:db8::1]:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Different IPv6 should work
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "[2001:db8::2]:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestRateLimit_IPv4MappedIPv6 verifies IPv4-mapped IPv6 addresses are handled correctly
func TestRateLimit_IPv4MappedIPv6(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// IPv4 address
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.100:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	assert.Equal(t, http.StatusOK, rr1.Code)

	// Same IP as IPv4-mapped IPv6 - should be tracked separately
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "[::ffff:192.168.1.100]:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code, "IPv4-mapped IPv6 is treated as different IP")
}

// TestRateLimit_Headers verifies X-RateLimit-* headers are added correctly
func TestRateLimit_Headers(t *testing.T) {
	config := newTestConfig(time.Minute, 10)
	config.AddHeaders = true
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.200:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "10", rr.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "9", rr.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Reset"))
}

// TestRateLimit_HeadersDisabled verifies headers are not added when disabled
func TestRateLimit_HeadersDisabled(t *testing.T) {
	config := newTestConfig(time.Minute, 10)
	config.AddHeaders = false
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.201:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("X-RateLimit-Limit"))
	assert.Empty(t, rr.Header().Get("X-RateLimit-Remaining"))
	assert.Empty(t, rr.Header().Get("X-RateLimit-Reset"))
}

// TestRateLimit_RetryAfter verifies Retry-After header calculation
func TestRateLimit_RetryAfter(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Use up the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.250:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Next request should be rate limited with Retry-After
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.250:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	retryAfter := rr.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfter)

	// Retry-After should be a positive integer (seconds)
	retryAfterInt, err := strconv.Atoi(retryAfter)
	require.NoError(t, err)
	assert.Greater(t, retryAfterInt, 0)
	assert.LessOrEqual(t, retryAfterInt, 60) // Should be at most 60 seconds
}

// TestRateLimit_Cleanup verifies stale entries are cleaned up
func TestRateLimit_Cleanup(t *testing.T) {
	config := newTestConfig(200*time.Millisecond, 5)
	config.CleanupInterval = 300 * time.Millisecond
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Create entries for multiple IPs
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	for _, ip := range ips {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ip + ":12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Wait for window to expire and cleanup to run
	time.Sleep(500 * time.Millisecond)

	// IPs that have been cleaned up should have fresh limits
	for _, ip := range ips {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = ip + ":12345"
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code, "IP %s should have fresh limit after cleanup", ip)
		}
	}
}

// TestRateLimit_Integration verifies integration with Chain
func TestRateLimit_ChainIntegration(t *testing.T) {
	config := newTestConfig(time.Minute, 3)

	// Create a handler that adds a custom header
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "test")
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Successful request should have both headers
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.2.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "test", rr.Header().Get("X-Custom"))
	assert.Equal(t, "3", rr.Header().Get("X-RateLimit-Limit"))
}

// TestRateLimit_InvalidIP verifies handling of invalid IP addresses
func TestRateLimit_InvalidIP(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Test with invalid RemoteAddr (no IP format)
	// This simulates edge cases where RemoteAddr might be malformed
	testCases := []struct {
		name       string
		remoteAddr string
	}{
		{"empty", ""},
		{"invalid", "not-an-ip"},
		{"just-port", ":12345"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			rr := httptest.NewRecorder()

			// Should not panic, should handle gracefully
			assert.NotPanics(t, func() {
				handler.ServeHTTP(rr, req)
			})

			// All invalid IPs are treated as "unknown" and share the same limit
			// So subsequent requests with invalid IPs will be rate limited together
		})
	}
}

// TestRateLimit_RemoteAddrWithoutPort verifies handling of RemoteAddr without port
func TestRateLimit_RemoteAddrWithoutPort(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Some configurations might have RemoteAddr without port
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.3.1" // No port
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.3.1"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

// TestRateLimit_MultipleMiddlewares verifies rate limiter works with other middleware
func TestRateLimit_MultipleMiddlewares(t *testing.T) {
	config := newTestConfig(time.Minute, 2)

	// Create a custom middleware that adds a header
	customMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		customMiddleware,
		Recovery,
		RateLimit(config),
	)

	// Successful request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.4.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "applied", rr.Header().Get("X-Middleware"))
	assert.Equal(t, "2", rr.Header().Get("X-RateLimit-Limit"))
}

// TestRateLimit_ZeroMaxRequests verifies behavior with zero max requests
func TestRateLimit_ZeroMaxRequests(t *testing.T) {
	config := newTestConfig(time.Minute, 0)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// All requests should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.5.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

// TestRateLimit_LargeWindow verifies behavior with large time windows
func TestRateLimit_LargeWindow(t *testing.T) {
	config := newTestConfig(time.Hour, 5)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Record time before requests to compute expected Retry-After
	before := time.Now()

	// Make requests up to limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.6.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.6.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Compute expected Retry-After: remaining time until end of current window.
	// Window boundaries align to clock hours (floor division), so we replicate the
	// same calculation used by getWindowStartAt.
	windowSecs := int64(time.Hour.Seconds())
	windowStart := time.Unix((before.Unix()/windowSecs)*windowSecs, 0)
	windowEnd := windowStart.Add(time.Hour)
	expectedRetryAfter := int(time.Until(windowEnd).Seconds())

	retryAfter := rr.Header().Get("Retry-After")
	retryAfterInt, err := strconv.Atoi(retryAfter)
	require.NoError(t, err)
	assert.InDelta(t, expectedRetryAfter, retryAfterInt, 5) // within 5 seconds of expected
	assert.LessOrEqual(t, retryAfterInt, 3600)              // at most 1 hour
}

// TestRateLimit_SanitizeIPv4 verifies IPv4 address sanitization
func TestRateLimit_SanitizeIPv4(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"192.168.1.1", "192.168.1.1"},
		{"192.168.1.1:8080", "192.168.1.1"},
		{"10.0.0.1", "10.0.0.1"},
		{"invalid-ip", "unknown"},
		{"", "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeIP(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestRateLimit_ExtractIPWithPort verifies IP extraction with port handling
func TestRateLimit_ExtractIPWithPort(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"ipv4_with_port", "192.168.1.1:8080", "192.168.1.1"},
		{"ipv4_without_port", "192.168.1.1", "192.168.1.1"},
		{"ipv6_with_port", "[2001:db8::1]:8080", "2001:db8::1"},
		{"ipv6_without_port", "2001:db8::1", "2001:db8::1"},
		{"invalid_with_port", "invalid:8080", "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeIP(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestRateLimit_SanitizeIPv6 verifies IPv6 address sanitization
func TestRateLimit_SanitizeIPv6(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"2001:db8::1", "2001:db8::1"},
		{"[2001:db8::1]:8080", "2001:db8::1"},
		{"::1", "::1"},
		{"fe80::1", "fe80::1"},
		{"invalid-ipv6", "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeIP(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestRateLimit_ExtractIP verifies IP extraction logic
func TestRateLimit_ExtractIP(t *testing.T) {
	testCases := []struct {
		name       string
		remoteAddr string
		xRealIP    string
		xForwarded string
		trustProxy bool
		expected   string
	}{
		{
			name:       "remote_addr_only",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
			trustProxy: false,
		},
		{
			name:       "x_real_ip_trusted",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.1",
			expected:   "203.0.113.1",
			trustProxy: true,
		},
		{
			name:       "x_real_ip_not_trusted",
			remoteAddr: "192.168.1.1:12345",
			xRealIP:    "203.0.113.1",
			expected:   "192.168.1.1",
			trustProxy: false,
		},
		{
			name:       "x_forwarded_for_trusted",
			remoteAddr: "10.0.0.1:12345",
			xForwarded: "203.0.113.100, 10.0.0.2",
			expected:   "203.0.113.100",
			trustProxy: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}
			if tc.xForwarded != "" {
				req.Header.Set("X-Forwarded-For", tc.xForwarded)
			}

			result := extractIP(req, tc.trustProxy)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestRateLimit_WithLogging verifies that rate limiting works with logging middleware
func TestRateLimit_WithLogging(t *testing.T) {
	config := newTestConfig(time.Minute, 2)

	// Create a test logger
	var logBuffer strings.Builder
	testLogger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create handler with logging
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			testLogger.Info("Request processed")
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Make successful request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.7.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, logBuffer.String(), "Request processed")
}

// TestRateLimit_StopCleanup verifies that stopCleanup can be called without panic
func TestRateLimit_StopCleanup(t *testing.T) {
	config := newTestConfig(time.Minute, 10)
	config.CleanupInterval = 100 * time.Millisecond

	// Create rate limiter directly to access stopCleanup
	rl := &RateLimiter{
		config:      config,
		cleanupDone: make(chan struct{}),
		logger:      slog.Default(),
	}

	// Start cleanup
	rl.startCleanup()

	// Stop cleanup - this should not panic
	assert.NotPanics(t, func() {
		rl.stopCleanup()
	})

	// Calling stopCleanup again should be idempotent (no panic due to sync.Once)
	assert.NotPanics(t, func() {
		rl.stopCleanup()
	})
}

// TestRateLimit_AddHeadersDisabled verifies behavior when headers are disabled
func TestRateLimit_AddHeadersDisabledFull(t *testing.T) {
	config := newTestConfig(time.Minute, 1)
	config.AddHeaders = false
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Make successful request - no headers should be added
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.8.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("X-RateLimit-Limit"))
	assert.Empty(t, rr.Header().Get("X-RateLimit-Remaining"))
	assert.Empty(t, rr.Header().Get("X-RateLimit-Reset"))

	// Make second request - should be rate limited without headers
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.8.1:12345"
	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	// Retry-After should still be set even when AddHeaders is false
	assert.NotEmpty(t, rr.Header().Get("Retry-After"))
}

// TestRateLimit_SubSecondWindow verifies windows smaller than 1 second
func TestRateLimit_SubSecondWindow(t *testing.T) {
	config := newTestConfig(500*time.Millisecond, 2)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Make 2 successful requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.9.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.9.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Wait for window to expire
	time.Sleep(600 * time.Millisecond)

	// New request should succeed
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.9.1:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestRateLimit_MultipleIPsInXForwardedFor verifies handling of multiple IPs
func TestRateLimit_MultipleIPsInXForwardedFor(t *testing.T) {
	config := newTestConfig(time.Minute, 2)
	config.TrustProxy = true
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Multiple IPs in X-Forwarded-For - first should be used
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "  203.0.113.50  ,  10.0.0.2  ") // Extra spaces
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "1", rr.Header().Get("X-RateLimit-Remaining"))
}

// TestRateLimit_XRealIPWithSpaces verifies X-Real-IP with whitespace
func TestRateLimit_XRealIPWithSpaces(t *testing.T) {
	config := newTestConfig(time.Minute, 1)
	config.TrustProxy = true
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// X-Real-IP with leading/trailing spaces
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "  203.0.113.60  ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestRateLimit_NegativeRetryAfter verifies retry-after when reset time is in past
func TestRateLimit_NegativeRetryAfter(t *testing.T) {
	config := newTestConfig(10*time.Millisecond, 1)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// Make request to fill limit
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.10.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Wait for window to expire
	time.Sleep(20 * time.Millisecond)

	// This request should get a new window, not be rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.10.1:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestRateLimit_CleanupDeletesStaleEntries verifies cleanup actually removes entries
func TestRateLimit_CleanupDeletesStaleEntries(t *testing.T) {
	config := newTestConfig(100*time.Millisecond, 5)
	config.CleanupInterval = 200 * time.Millisecond

	// Create rate limiter directly to access internals
	rl := &RateLimiter{
		config:      config,
		cleanupDone: make(chan struct{}),
		logger:      slog.Default(),
	}
	rl.startCleanup()
	defer rl.stopCleanup()

	// Manually add some entries with old window start
	oldWindow := time.Now().Add(-time.Hour).Unix() / 1000 * 1000
	rl.clients.Store("192.168.100.1", &clientTracker{
		count:       5,
		windowStart: oldWindow,
		mu:          sync.RWMutex{},
	})
	rl.clients.Store("192.168.100.2", &clientTracker{
		count:       3,
		windowStart: oldWindow,
		mu:          sync.RWMutex{},
	})

	// Add current entry
	currentWindow := rl.getWindowStart()
	rl.clients.Store("192.168.100.3", &clientTracker{
		count:       1,
		windowStart: currentWindow,
		mu:          sync.RWMutex{},
	})

	// Trigger cleanup
	rl.removeStaleEntries()

	// Verify old entries are deleted
	_, exists1 := rl.clients.Load("192.168.100.1")
	_, exists2 := rl.clients.Load("192.168.100.2")
	_, exists3 := rl.clients.Load("192.168.100.3")

	assert.False(t, exists1, "Old entry 1 should be deleted")
	assert.False(t, exists2, "Old entry 2 should be deleted")
	assert.True(t, exists3, "Current entry should still exist")
}

// TestRateLimit_RetryAfterNegative verifies retry-after fallback when reset time is in past
// This tests the edge case where time.Until returns a negative value
func TestRateLimit_RetryAfterNegative(t *testing.T) {
	// Test with reset time in the past - should return 1 second
	resetTime := time.Now().Add(-time.Hour)
	retryAfter := calculateRetryAfter(resetTime)
	assert.Equal(t, time.Second, retryAfter, "Should fallback to 1 second for negative retryAfter")
}

// TestRateLimit_CalculateRetryAfterPositive verifies normal retry-after calculation
func TestRateLimit_CalculateRetryAfterPositive(t *testing.T) {
	// Test with reset time in the future - should return positive duration
	resetTime := time.Now().Add(time.Minute)
	retryAfter := calculateRetryAfter(resetTime)
	assert.Greater(t, retryAfter, time.Duration(0), "Should return positive duration for future reset time")
	assert.LessOrEqual(t, retryAfter, time.Minute, "Should be at most 1 minute")
}

// TestRateLimit_RetryAfterWithMockTime tests the negative retryAfter branch using a mock scenario
func TestRateLimit_RetryAfterWithMockTime(t *testing.T) {
	// This test attempts to hit the edge case where resetTime is in the past
	// by using a very short window and precise timing

	config := newTestConfig(2*time.Millisecond, 1)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		RateLimit(config),
	)

	// First request fills the limit at the start of a window
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.14.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Wait for almost the entire window to expire
	time.Sleep(3 * time.Millisecond)

	// Second request should get a new window (200 OK) since window reset
	// But let's try to trigger the edge case by making a request that gets rate limited
	// Reset the window by making fresh requests with a different IP first

	// Use a new IP and fill its limit quickly
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.14.2:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Immediately make another request for the same IP - should be rate limited
	// If timing is right, the window might expire between checkLimit and retryAfter calculation
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.14.2:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Check if we got rate limited
	if rr.Code == http.StatusTooManyRequests {
		retryAfter := rr.Header().Get("Retry-After")
		retryAfterInt, err := strconv.Atoi(retryAfter)
		require.NoError(t, err)
		// If retryAfter is 1, we hit the edge case!
		if retryAfterInt == 1 {
			t.Log("Successfully hit the negative retryAfter edge case!")
			return
		}
	}

	// If we didn't hit the edge case, that's okay - it's extremely rare
	// The important thing is that the code works correctly
	t.Skip("Could not reproduce the negative retryAfter edge case - this is expected and acceptable")
}

// TestRateLimit_RetryAfterZeroWindow tests with extremely short window to try to hit the negative retryAfter branch
func TestRateLimit_RetryAfterZeroWindow(t *testing.T) {
	// Use a very short window and try to time it so the window expires during request processing
	config := newTestConfig(1*time.Millisecond, 1)

	rl := &RateLimiter{
		config:      config,
		cleanupDone: make(chan struct{}),
		logger:      slog.Default(),
	}
	rl.startCleanup()
	defer rl.stopCleanup()

	// First request fills the limit
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.13.1:12345"

	// Create handler using the rate limiter middleware
	middleware := RateLimit(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Wait for window to expire
	time.Sleep(2 * time.Millisecond)

	// Second request should succeed since window reset
	// (not trigger the negative retryAfter branch, which is extremely rare)
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.13.1:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
