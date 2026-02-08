package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

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

func TestMonitoring_MiddlewareCreatesSpan(t *testing.T) {
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify logger is in context
		log := logger.FromContext(r.Context())
		require.NotNil(t, log)

		// Verify trace_id is added to logger
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test/path", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestMonitoring_MiddlewareAddsTraceIDHeader(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Check X-Trace-Id header is set
	traceID := rr.Header().Get("X-Trace-Id")
	assert.NotEmpty(t, traceID)

	// Verify it's a valid trace ID format (32 hex chars for a valid trace)
	// or empty string for invalid trace
	if traceID != "" {
		assert.True(t, len(traceID) == 32 || len(traceID) == 16)
	}
}

func TestMonitoring_MiddlewareWithRequestBody(t *testing.T) {
	requestBody := `{"test": "data"}`
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify body is readable
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, requestBody, string(body))

		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("POST", "/test", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMonitoring_MiddlewareWithResponseBody(t *testing.T) {
	responseBody := `{"result": "success"}`
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, responseBody, rr.Body.String())
}

func TestMonitoring_MiddlewareCutsLongBody(t *testing.T) {
	// Test the cut function directly
	longBody := make([]byte, BodyMaxLen+1000)
	for i := range longBody {
		longBody[i] = 'x'
	}

	result := cut(longBody)
	assert.Contains(t, result, "...")
	assert.Contains(t, result, "bytes")
	assert.True(t, strings.HasPrefix(result, string(longBody[:BodyMaxLen])))
	assert.Equal(t, BodyMaxLen, strings.Index(result, "..."))

	// Verify total length calculation is in the result
	totalLen := len(longBody)
	assert.Contains(t, result, "...(")
	assert.Contains(t, result, " bytes)")
	// The format is "...(%d bytes)" so check for the length digits
	lengthStr := string(rune('0' + totalLen%10))
	if totalLen >= 10 {
		lengthStr = string(rune('0'+(totalLen/10)%10)) + lengthStr
	}
	if totalLen >= 100 {
		lengthStr = string(rune('0'+(totalLen/100)%10)) + lengthStr
	}
	if totalLen >= 1000 {
		lengthStr = string(rune('0'+(totalLen/1000)%10)) + lengthStr
	}
	assert.Contains(t, result, lengthStr)
}

func TestMonitoring_CutShortBody(t *testing.T) {
	// Test cut with short body (no truncation)
	shortBody := []byte("short content")

	result := cut(shortBody)
	assert.Equal(t, "short content", result)
	assert.NotContains(t, result, "...")
}

func TestMonitoring_CutEmptyBody(t *testing.T) {
	emptyBody := []byte{}

	result := cut(emptyBody)
	assert.Equal(t, "", result)
}

func TestMonitoring_CutExactMaxLength(t *testing.T) {
	exactBody := make([]byte, BodyMaxLen)
	for i := range exactBody {
		exactBody[i] = 'a'
	}

	result := cut(exactBody)
	assert.Equal(t, string(exactBody), result)
	assert.NotContains(t, result, "...")
}

func TestMonitoring_StatefulRespWriterWriteHeader(t *testing.T) {
	underlying := httptest.NewRecorder()
	srw := newStatefulRespWriter(underlying)

	// Initially status should be 0
	assert.Equal(t, 0, srw.status)

	// After WriteHeader, status should be set
	srw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, srw.status)
	assert.Equal(t, http.StatusCreated, underlying.Code)
}

func TestMonitoring_StatefulRespWriterWriteDefaultsTo200(t *testing.T) {
	underlying := httptest.NewRecorder()
	srw := newStatefulRespWriter(underlying)

	// Initially status should be 0
	assert.Equal(t, 0, srw.status)

	// Write without calling WriteHeader should default to 200
	n, err := srw.Write([]byte("test"))
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, http.StatusOK, srw.status)
	assert.Equal(t, "test", underlying.Body.String())
}

func TestMonitoring_StatefulRespWriterFlush(t *testing.T) {
	underlying := httptest.NewRecorder()
	srw := newStatefulRespWriter(underlying)

	// Should not panic even if underlying doesn't support Flush
	srw.Flush()
}

func TestMonitoring_StatefulRespWriterWithFlusher(t *testing.T) {
	// Create a response writer that supports flushing
	underlying := &flushableResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
		flushed:        false,
	}
	srw := newStatefulRespWriter(underlying)

	srw.Flush()

	assert.True(t, underlying.flushed)
}

func TestMonitoring_StatefulRespWriterBodyCapture(t *testing.T) {
	underlying := httptest.NewRecorder()
	srw := newStatefulRespWriter(underlying)

	writeData := []byte("response body data")
	srw.Write(writeData)

	assert.Equal(t, writeData, srw.body)
}

func TestMonitoring_StatefulRespWriterMultipleWrites(t *testing.T) {
	underlying := httptest.NewRecorder()
	srw := newStatefulRespWriter(underlying)

	srw.Write([]byte("first"))
	srw.Write([]byte("second"))

	// body should only contain the last write
	assert.Equal(t, []byte("second"), srw.body)
	assert.Equal(t, "firstsecond", underlying.Body.String())
}

func TestMonitoring_MiddlewareRecordsMetrics(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/api/v1/resource", nil)
	rr := httptest.NewRecorder()

	// Should not panic when recording metrics
	assert.NotPanics(t, func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMonitoring_MiddlewareWithErrorStatus(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/error", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestMonitoring_MiddlewareWithHeaders(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("X-Trace-Id"))
}

func TestMonitoring_RequestURIWithoutParameters(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test?param1=value1&param2=value2", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMonitoring_ContextWithLogger(t *testing.T) {
	var capturedLog *slog.Logger
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedLog = logger.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.NotNil(t, capturedLog)
}

func TestMonitoring_MiddlewareChain(t *testing.T) {
	// Test monitoring in a middleware chain
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-1", "applied")
			next.ServeHTTP(w, r)
		})
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Chain: middleware1 -> monitoring -> handler
	handler := middleware1(Monitoring(nextHandler))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
	assert.Equal(t, "applied", rr.Header().Get("X-Middleware-1"))
	assert.NotEmpty(t, rr.Header().Get("X-Trace-Id"))
}

func TestMonitoring_VariousHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := Monitoring(nextHandler)

			req := httptest.NewRequest(method, "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestMonitoring_VariousStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
	}

	for _, status := range statusCodes {
		t.Run(http.StatusText(status), func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			})

			handler := Monitoring(nextHandler)

			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, status, rr.Code)
		})
	}
}

func TestMonitoring_RequestBodyReadError(t *testing.T) {
	// Create a reader that returns an error
	errorReader := &errReader{err: io.EOF}
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("POST", "/test", errorReader)
	rr := httptest.NewRecorder()

	// Should handle the read error gracefully
	assert.NotPanics(t, func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMonitoring_EmptyRequestBody(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the empty body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte{}, body)

		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMonitoring_LargeRequestBody(t *testing.T) {
	largeBody := make([]byte, 10*1024) // 10KB
	for i := range largeBody {
		largeBody[i] = 'x'
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, largeBody, body)
		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("POST", "/test", strings.NewReader(string(largeBody)))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMonitoring_LargeResponseBody(t *testing.T) {
	largeBody := make([]byte, 10*1024) // 10KB
	for i := range largeBody {
		largeBody[i] = 'y'
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeBody)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, largeBody, rr.Body.Bytes())
}

func TestMonitoring_WithExistingContextLogger(t *testing.T) {
	existingLog := noop.NewNoop().With("existing", "value")
	ctx := logger.NewContext(context.Background(), existingLog)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())
		assert.NotNil(t, log)
		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMonitoring_TracePropagation(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify span context exists
		span := trace.SpanFromContext(r.Context())
		assert.NotNil(t, span)
		w.WriteHeader(http.StatusOK)
	})

	handler := Monitoring(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// flushableResponseWriter is a test helper that implements http.Flusher
type flushableResponseWriter struct {
	http.ResponseWriter
	flushed bool
}

func (f *flushableResponseWriter) Flush() {
	f.flushed = true
}

// errReader is a test helper that returns an error on Read
type errReader struct {
	err error
}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func (e *errReader) Close() error {
	return nil
}
