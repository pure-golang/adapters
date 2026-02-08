package middleware

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/pure-golang/adapters/logger"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	meter = otel.GetMeterProvider().Meter("github.com/pure-golang/adapters/httpserver/middleware")
	// nolint:errcheck // Sync OpenTelemetry instruments never return errors
	requestsCount, _       = meter.Int64Counter("http.request_count")
	requestTimeHist, _     = meter.Int64Histogram("http.request_time", metric.WithUnit("ms"))
	requestBodyLenHist, _  = meter.Int64Histogram("http.request_body_len", metric.WithUnit("KB"))
	responseBodyLenHist, _ = meter.Int64Histogram("http.response_body_len", metric.WithUnit("KB"))
	tracer                 = otel.Tracer("github.com/pure-golang/adapters/httpserver/middleware")
)

// Monitoring traces incoming http requests using open telemetry tracer + attaches logger to request context
func Monitoring(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqTime := time.Now()
		ctx := r.Context()

		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))

		uriWithoutParameters := strings.Split(r.RequestURI, "?")[0]
		ctx, span := tracer.Start(ctx, uriWithoutParameters, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		metricLabels := []attribute.KeyValue{
			attribute.String("http.method", uriWithoutParameters),
		}

		traceID := span.SpanContext().TraceID().String()

		// logger
		log := slog.Default().With(r.Method, r.RequestURI)
		if traceID != "" {
			log = log.With("trace_id", traceID)
		}

		// attributes
		attrs := semconv.NetAttributesFromHTTPRequest("tcp", r)
		attrs = append(attrs, semconv.EndUserAttributesFromHTTPRequest(r)...)
		attrs = append(attrs, semconv.HTTPServerAttributesFromHTTPRequest("webserver", r.RequestURI, r)...)
		attrs = append(attrs, attribute.String("http.request.header.Authorization", r.Header.Get("Authorization")))
		attrs = append(attrs, attribute.String("http.request.header.Cookie", r.Header.Get("Cookie")))
		attrs = append(attrs, attribute.String("http.request.header.User-Agent", r.Header.Get("User-Agent")))

		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("failed to read body", "error", err)
		} else {
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
			attrs = append(attrs, attribute.String("http.request.body_2048", cut(reqBody)))
		}

		w.Header().Set("X-Trace-Id", traceID)

		ctx = logger.NewContext(ctx, log)
		srw := newStatefulRespWriter(w)

		next.ServeHTTP(srw, r.WithContext(ctx))

		attrs = append(attrs, attribute.Int("http.response.status", srw.status))
		attrs = append(attrs, attribute.String("http.response.body_2048", cut(srw.body)))
		span.SetAttributes(attrs...)

		// metrics
		requestsCount.Add(ctx, 1, metric.WithAttributes(append(metricLabels,
			attribute.Int("http.response.code", srw.status))...))
		requestTimeHist.Record(ctx, time.Since(reqTime).Milliseconds(), metric.WithAttributes(metricLabels...))
		requestBodyLenHist.Record(ctx, int64(len(reqBody))/1024, metric.WithAttributes(metricLabels...))
		responseBodyLenHist.Record(ctx, int64(len(srw.body))/1024, metric.WithAttributes(metricLabels...))
		if srw.status >= 500 {
			span.SetStatus(codes.Error, "")
			return
		}

		span.SetStatus(codes.Ok, "")
	})
}

// statefulRespWriter keeps sent status and body after WriterHeader/Write calls
type statefulRespWriter struct {
	http.ResponseWriter
	status int
	body   []byte
}

func newStatefulRespWriter(w http.ResponseWriter) *statefulRespWriter {
	return &statefulRespWriter{ResponseWriter: w}
}

func (w *statefulRespWriter) WriteHeader(status int) {
	w.ResponseWriter.WriteHeader(status)
	w.status = status
}

func (w *statefulRespWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	w.body = b
	return w.ResponseWriter.Write(b)
}

func (w *statefulRespWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

const BodyMaxLen = 2048

func cut(body []byte) string {
	if length := len(body); length > BodyMaxLen {
		return fmt.Sprintf("%s...(%d bytes)", string(body[:BodyMaxLen]), length)
	}
	return string(body)
}
