package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestChain_NoMiddleware returns the original handler when no middlewares are provided
func TestChain_NoMiddleware(t *testing.T) {
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("original"))
	})

	// Chain with no middlewares
	chained := Chain(originalHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "original", rr.Body.String())
}

// TestChain_SingleMiddleware applies one middleware correctly
func TestChain_SingleMiddleware(t *testing.T) {
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("handler"))
	})

	// Middleware that adds a header
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	chained := Chain(originalHandler, middleware)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "applied", rr.Header().Get("X-Middleware"))
	assert.Equal(t, "handler", rr.Body.String())
}

// TestChain_MultipleMiddlewares applies middlewares in correct (reverse) order
func TestChain_MultipleMiddlewares(t *testing.T) {
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("handler"))
	})

	// Middleware 1
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-1", "first")
			next.ServeHTTP(w, r)
		})
	}

	// Middleware 2
	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-2", "second")
			next.ServeHTTP(w, r)
		})
	}

	// Middleware 3
	middleware3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-3", "third")
			next.ServeHTTP(w, r)
		})
	}

	// Chain: middleware3 -> middleware2 -> middleware1 -> handler
	// Execution order: middleware1 -> middleware2 -> middleware3 -> handler
	// Because Chain iterates in reverse
	chained := Chain(originalHandler, middleware1, middleware2, middleware3)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "first", rr.Header().Get("X-Middleware-1"))
	assert.Equal(t, "second", rr.Header().Get("X-Middleware-2"))
	assert.Equal(t, "third", rr.Header().Get("X-Middleware-3"))
	assert.Equal(t, "handler", rr.Body.String())
}

// TestChain_MiddlewareExecutionOrder verifies that middlewares execute in the correct order
func TestChain_MiddlewareExecutionOrder(t *testing.T) {
	var executionOrder []string

	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware1-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware2-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware2-after")
		})
	}

	middleware3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware3-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware3-after")
		})
	}

	// Chain in order: middleware1, middleware2, middleware3
	// Applied in reverse: middleware3 wraps middleware2 wraps middleware1 wraps handler
	// Execution: middleware1 -> middleware2 -> middleware3 -> handler
	chained := Chain(originalHandler, middleware1, middleware2, middleware3)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	expectedOrder := []string{
		"middleware1-before",
		"middleware2-before",
		"middleware3-before",
		"handler",
		"middleware3-after",
		"middleware2-after",
		"middleware1-after",
	}

	assert.Equal(t, expectedOrder, executionOrder)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestChain_MiddlewareCanShortCircuit tests that middleware can stop the chain
func TestChain_MiddlewareCanShortCircuit(t *testing.T) {
	handlerCalled := false
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// First middleware allows chain to continue
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-1", "passed")
			next.ServeHTTP(w, r)
		})
	}

	// Second middleware short-circuits
	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-2", "short-circuited")
			w.WriteHeader(http.StatusForbidden)
			// Don't call next.ServeHTTP
		})
	}

	chained := Chain(originalHandler, middleware1, middleware2)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Equal(t, "passed", rr.Header().Get("X-Middleware-1"))
	assert.Equal(t, "short-circuited", rr.Header().Get("X-Middleware-2"))
	assert.False(t, handlerCalled, "Original handler should not be called when middleware short-circuits")
}

// TestChain_MiddlewareCanModifyRequest tests middleware modifying the request
func TestChain_MiddlewareCanModifyRequest(t *testing.T) {
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if header was added by middleware
		customHeader := r.Header.Get("X-Custom-Header")
		w.Header().Set("X-Response-Header", customHeader)
		w.WriteHeader(http.StatusOK)
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add custom header to request
			r.Header.Set("X-Custom-Header", "custom-value")
			next.ServeHTTP(w, r)
		})
	}

	chained := Chain(originalHandler, middleware)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "custom-value", rr.Header().Get("X-Response-Header"))
}

// TestChain_MiddlewareCanModifyResponse tests middleware modifying the response
func TestChain_MiddlewareCanModifyResponse(t *testing.T) {
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("original body"))
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Call next to get the response
			next.ServeHTTP(w, r)

			// Modify the response
			// Note: This won't work for body since response writer wraps it,
			// but headers can be modified
			w.Header().Set("X-Modified", "true")
		})
	}

	chained := Chain(originalHandler, middleware)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, "true", rr.Header().Get("X-Modified"))
	assert.Equal(t, "original body", rr.Body.String())
}

// TestChain_EmptyMiddlewareSlice returns original handler when middleware slice is empty
func TestChain_EmptyMiddlewareSlice(t *testing.T) {
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	// Chain with empty middleware slice
	var noMiddlewares []func(http.Handler) http.Handler
	chained := Chain(originalHandler, noMiddlewares...)

	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, "created", rr.Body.String())
}

// TestChain_WithRecoveryMiddleware tests Chain with Recovery middleware
func TestChain_WithRecoveryMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Add another middleware before recovery
	additionalMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Before-Recovery", "yes")
			next.ServeHTTP(w, r)
		})
	}

	chained := Chain(panicHandler, additionalMiddleware, Recovery)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic due to Recovery
	assert.NotPanics(t, func() {
		chained.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "yes", rr.Header().Get("X-Before-Recovery"))
}

// TestChain_MiddlewareContextPassing tests that middleware can modify request context
func TestChain_MiddlewareContextPassing(t *testing.T) {
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context value was set by middleware
		val := r.Context().Value("test-key")
		assert.NotNil(t, val, "Context value should be available in handler")
		assert.Equal(t, "test-value", val)
		w.WriteHeader(http.StatusOK)
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add value to context using context.WithValue
			ctx := context.WithValue(r.Context(), "test-key", "test-value")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	chained := Chain(originalHandler, middleware)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestChain_PanicInMiddleware tests behavior when middleware itself panics
func TestChain_PanicInMiddleware(t *testing.T) {
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	panickingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("middleware panic")
		})
	}

	chained := Chain(originalHandler, panickingMiddleware)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Should panic because no recovery middleware
	assert.Panics(t, func() {
		chained.ServeHTTP(rr, req)
	})
}

// TestChain_MiddlewarePanicWithRecovery tests that panic in handler is caught by Recovery
func TestChain_MiddlewarePanicWithRecovery(t *testing.T) {
	panickingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("handler panic")
	})

	safeMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Safe-Middleware", "yes")
			next.ServeHTTP(w, r)
		})
	}

	// Chain with Recovery to catch the panic from handler
	// Recovery wraps the handler, so it catches panics from the handler
	chained := Chain(panickingHandler, safeMiddleware, Recovery)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic because Recovery catches it
	assert.NotPanics(t, func() {
		chained.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "yes", rr.Header().Get("X-Safe-Middleware"))
}

// TestChain_MultipleHandlersInChain tests creating chains from chains
func TestChain_MultipleHandlersInChain(t *testing.T) {
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Handler", "1")
		w.WriteHeader(http.StatusOK)
	})

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Handler", "2")
		w.WriteHeader(http.StatusAccepted)
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	// Create chain with handler1
	chained1 := Chain(handler1, middleware)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained1.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "1", rr.Header().Get("X-Handler"))
	assert.Equal(t, "applied", rr.Header().Get("X-Middleware"))

	// Create chain with handler2
	chained2 := Chain(handler2, middleware)

	req2 := httptest.NewRequest("GET", "/test2", nil)
	rr2 := httptest.NewRecorder()

	chained2.ServeHTTP(rr2, req2)

	assert.Equal(t, http.StatusAccepted, rr2.Code)
	assert.Equal(t, "2", rr2.Header().Get("X-Handler"))
	assert.Equal(t, "applied", rr2.Header().Get("X-Middleware"))
}

// TestChain_ChainOfChains tests chaining already chained handlers
func TestChain_ChainOfChains(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("base"))
	})

	middlewareA := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-A", "true")
			next.ServeHTTP(w, r)
		})
	}

	middlewareB := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-B", "true")
			next.ServeHTTP(w, r)
		})
	}

	middlewareC := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-C", "true")
			next.ServeHTTP(w, r)
		})
	}

	// First chain: base + A + B
	firstChain := Chain(baseHandler, middlewareA, middlewareB)

	// Second chain: firstChain + C
	secondChain := Chain(firstChain, middlewareC)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	secondChain.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "true", rr.Header().Get("X-A"))
	assert.Equal(t, "true", rr.Header().Get("X-B"))
	assert.Equal(t, "true", rr.Header().Get("X-C"))
	assert.Equal(t, "base", rr.Body.String())
}

// TestChain_WithDifferentHTTPMethods tests chain works with different HTTP methods
func TestChain_WithDifferentHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Method", r.Method)
				w.WriteHeader(http.StatusOK)
			})

			middleware := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Middleware", "applied")
					next.ServeHTTP(w, r)
				})
			}

			chained := Chain(originalHandler, middleware)

			req := httptest.NewRequest(method, "/test", nil)
			rr := httptest.NewRecorder()

			chained.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "applied", rr.Header().Get("X-Middleware"))
			assert.Equal(t, method, rr.Header().Get("X-Method"))
		})
	}
}

// TestChain_LargeNumberOfMiddlewares tests chain with many middlewares
func TestChain_LargeNumberOfMiddlewares(t *testing.T) {
	handlerCallCount := 0
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCallCount++
		w.WriteHeader(http.StatusOK)
	})

	// Create 10 middlewares
	var middlewares []func(http.Handler) http.Handler
	for i := 0; i < 10; i++ {
		i := i // Capture loop variable
		middlewares = append(middlewares, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware-"+string(rune('0'+i)), "applied")
				next.ServeHTTP(w, r)
			})
		})
	}

	chained := Chain(originalHandler, middlewares...)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, handlerCallCount, "Handler should be called exactly once")
}

// TestChain_NilMiddlewareFunc tests behavior with nil middleware function in the chain
// Note: Chain will panic when it tries to call the nil middleware function during wrapping
func TestChain_NilMiddlewareFunc(t *testing.T) {
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var nilMiddleware func(http.Handler) http.Handler

	// Chain with a nil middleware - this should panic when Chain tries to call it
	assert.Panics(t, func() {
		Chain(originalHandler, nilMiddleware)
	}, "Chain should panic when given a nil middleware function")
}

// TestChain_VerifyReverseOrderLoop explicitly tests the reverse order loop behavior
func TestChain_VerifyReverseOrderLoop(t *testing.T) {
	var callOrder []int

	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, 0)
		w.WriteHeader(http.StatusOK)
	})

	// Create 3 middlewares that record their call order
	middlewares := []func(http.Handler) http.Handler{
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callOrder = append(callOrder, 1)
				next.ServeHTTP(w, r)
			})
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callOrder = append(callOrder, 2)
				next.ServeHTTP(w, r)
			})
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callOrder = append(callOrder, 3)
				next.ServeHTTP(w, r)
			})
		},
	}

	// Chain iterates from len(middlewares)-1 to 0
	// So it wraps in order: 3 -> 2 -> 1 -> handler
	// Execution: 1 -> 2 -> 3 -> handler
	chained := Chain(originalHandler, middlewares...)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	// Verify execution order
	expectedOrder := []int{1, 2, 3, 0}
	assert.Equal(t, expectedOrder, callOrder)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// mockHandlerWithFlag is a mock handler for testing
type mockHandlerWithFlag struct {
	called bool
}

func (m *mockHandlerWithFlag) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.called = true
	w.WriteHeader(http.StatusOK)
}

// TestChain_WithMockHandler tests chain with custom handler type
func TestChain_WithMockHandler(t *testing.T) {
	mock := &mockHandlerWithFlag{called: false}

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "yes")
			next.ServeHTTP(w, r)
		})
	}

	chained := Chain(mock, middleware)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.True(t, mock.called, "Mock handler should be called")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "yes", rr.Header().Get("X-Middleware"))
}
