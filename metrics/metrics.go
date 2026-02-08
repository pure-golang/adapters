package metrics

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Host                  string `envconfig:"METRICS_HOST" required:"true"`
	Port                  int    `envconfig:"METRICS_PORT" required:"true"`
	HttpServerReadTimeout int    `envconfig:"METRICS_READ_TIMEOUT" default:"30"`
}

type Metrics struct {
	io.Closer
	config Config
	server *http.Server
}

func InitDefault(config Config) (io.Closer, error) {
	provider := New(config)
	if err := provider.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start metrics server")
	}

	return provider, nil
}

func New(config Config) *Metrics {
	return &Metrics{
		config: config,
		server: NewHttpServer(config),
	}
}

func (s *Metrics) Start() error {
	if err := InitPrometheus(); err != nil {
		return errors.Wrap(err, "failed to init prometheus")
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Default().Warn("metrics server failed", "error", err.Error())
		}
	}()

	return nil
}

func (s *Metrics) Close() error {
	return errors.Wrap(s.server.Close(), "failed to close metrics")
}

func NewHttpServer(conf Config) *http.Server {
	r := http.NewServeMux()
	r.Handle("/metrics", promhttp.Handler())
	return &http.Server{
		Addr:        fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		Handler:     r,
		ReadTimeout: time.Duration(conf.HttpServerReadTimeout) * time.Second,
	}
}
