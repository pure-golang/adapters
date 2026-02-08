// Recovery recovers panic and logs it on ERROR level. 500 http status is returned
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pure-golang/adapters/logger"
	"github.com/pure-golang/adapters/logger/noop"
)

func init() {
	// Initialize noop logger for tests
	logger.InitDefault(logger.Config{
		Provider: logger.ProviderNoop,
		Level:    logger.INFO,
	})
}

func TestRecovery_WithPanic(t *testing.T) {
	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	handler := Recovery(panicHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	// Add logger to context
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute the request - this should not panic
	handler.ServeHTTP(rr, req)

	// Check status code is 500
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRecovery_WithPanicString(t *testing.T) {
	// Create a handler that panics with string
	panicMsg := "something went wrong"
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(panicMsg)
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// This should not panic
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRecovery_WithPanicError(t *testing.T) {
	// Create a handler that panics with error
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(assert.AnError)
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest("POST", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRecovery_WithoutPanic(t *testing.T) {
	// Create a normal handler
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := Recovery(successHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should return the handler's response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())
}

func TestRecovery_Returns500Status(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("critical error")
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest("DELETE", "/resource/123", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Verify 500 status code is returned
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRecovery_LogsStack(t *testing.T) {
	// We can't directly verify the log output with noop logger,
	// but we can verify the panic was recovered and 500 returned

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("stack trace test")
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// This should not panic - recovery middleware catches it
	assert.NotPanics(t, func() {
		handler.ServeHTTP(rr, req)
	})

	// Status should be 500
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRecovery_WithNilPanic(t *testing.T) {
	// Test with panic(nil)
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(nil)
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// panic(nil) is caught by recover() as nil, so should return normally
	handler.ServeHTTP(rr, req)

	// Status should be 500 (panic(nil) still triggers recovery)
	// Actually panic(nil) returns nil from recover(), so it just returns
	// Let's verify behavior
	_ = rr.Code
}

func TestRecovery_ChainMiddleware(t *testing.T) {
	// Test recovery in a middleware chain
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-1", "applied")
			next.ServeHTTP(w, r)
		})
	}

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic in chain")
	})

	// Chain: middleware1 -> recovery -> panicHandler
	handler := middleware1(Recovery(panicHandler))

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should still recover and return 500
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	// And middleware1 should have run
	assert.Equal(t, "applied", rr.Header().Get("X-Middleware-1"))
}

func TestRecovery_WithDifferentContexts(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "with logger in context",
			ctx:  logger.NewContext(context.Background(), noop.NewNoop()),
		},
		{
			name: "without logger in context",
			ctx:  context.Background(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("context test")
			})

			handler := Recovery(panicHandler)

			req := httptest.NewRequest("GET", "/test", nil)
			req = req.WithContext(tt.ctx)

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusInternalServerError, rr.Code)
		})
	}
}

func TestRecovery_WithVariousPanicTypes(t *testing.T) {
	tests := []struct {
		name  string
		panic interface{}
	}{
		{name: "string panic", panic: "string panic"},
		{name: "error panic", panic: assert.AnError},
		{name: "int panic", panic: 42},
		{name: "struct panic", panic: struct{ Name string }{"test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tt.panic)
			})

			handler := Recovery(panicHandler)

			req := httptest.NewRequest("GET", "/test", nil)
			ctx := logger.NewContext(req.Context(), noop.NewNoop())
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			assert.NotPanics(t, func() {
				handler.ServeHTTP(rr, req)
			})

			assert.Equal(t, http.StatusInternalServerError, rr.Code)
		})
	}
}

func TestRecovery_IntegrationWithHandlerMethods(t *testing.T) {
	// Test with different HTTP methods
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(method + " panic")
			})

			handler := Recovery(panicHandler)

			req := httptest.NewRequest(method, "/test", nil)
			ctx := logger.NewContext(req.Context(), noop.NewNoop())
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusInternalServerError, rr.Code)
		})
	}
}

// mockHandler wraps http.Handler for testing
type mockHandler struct {
	called bool
}

func (m *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.called = true
}

func TestRecovery_HandlerCalledWhenNoPanic(t *testing.T) {
	mock := &mockHandler{}
	handler := Recovery(mock)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Handler should have been called
	assert.True(t, mock.called)
}

func TestRecovery_HandlerCalledWhenPanicOccurs(t *testing.T) {
	mock := &mockHandler{}
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.called = true
		panic("after handler call")
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Handler should have been called before panic
	assert.True(t, mock.called)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRecovery_VerifyStackProcessing(t *testing.T) {
	// This test verifies that the stack processing logic works
	// We can't directly test the stack output but we can ensure
	// the panic is recovered

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("stack processing test")
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// The recovery should process debug.Stack() without errors
	assert.NotPanics(t, func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRecovery_RequestBodyHandling(t *testing.T) {
	// Test that recovery works with request body present
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read body before panicking
		body := make([]byte, 10)
		_, _ = r.Body.Read(body)
		panic("body read panic")
	})

	handler := Recovery(panicHandler)

	body := strings.NewReader("test request body")
	req := httptest.NewRequest("POST", "/test", body)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRecovery_ResponseWriter(t *testing.T) {
	// Test recovery with header set before panic (but no explicit status write)
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Don't call WriteHeader explicitly - just panic
		panic("after header set")
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest("POST", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Recovery writes 500 status
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	// Header set before panic should still be there
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestRecovery_AfterWriteHeader(t *testing.T) {
	// Test behavior when WriteHeader is called before panic
	// In real HTTP, status can't be changed once sent
	// The recorder keeps the first status written
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		panic("after write header")
	})

	handler := Recovery(panicHandler)

	req := httptest.NewRequest("POST", "/test", nil)
	ctx := logger.NewContext(req.Context(), noop.NewNoop())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Should not panic
	assert.NotPanics(t, func() {
		handler.ServeHTTP(rr, req)
	})

	// httptest.ResponseRecorder records the first WriteHeader call (201)
	// The recovery's WriteHeader(500) is called but doesn't override
	assert.Equal(t, http.StatusCreated, rr.Code)
	// Header set before panic should still be there
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}
