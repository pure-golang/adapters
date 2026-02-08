package std

import (
	"context"
	stdErr "errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/pure-golang/adapters/httpserver"
	"github.com/pkg/errors"
)

const ShutdownTimeout = 15 * time.Second

var _ httpserver.RunableProvider = (*Server)(nil)

type Config struct {
	Host        string `envconfig:"WEBSERVER_HOST"`
	Port        int    `envconfig:"WEBSERVER_PORT" required:"true"`
	TLSCertPath string `envconfig:"WEBSERVER_TLS_CERT_PATH"`
	TLSKeyPath  string `envconfig:"WEBSERVER_TLS_KEY_PATH"`
	ReadTimeout string `envconfig:"WEBSERVER_READ_TIMEOUT" default:"30"`
}

type Server struct {
	logger *slog.Logger
	server *http.Server
	config Config
}

func NewDefault(c Config, h http.Handler) *Server {
	s := New(c, h)

	s.server.ErrorLog = slog.NewLogLogger(s.logger.Handler(), slog.LevelError)

	return s
}

func New(c Config, h http.Handler /*ext: option functions*/) *Server {
	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", c.Host, c.Port),
			Handler:           h,
			ReadHeaderTimeout: 10 * time.Second, // Prevent Slowloris attacks
		},
		logger: slog.Default().WithGroup("webserver"),
		config: c,
	}
}

func (s *Server) Start() error {
	var err error
	s.logger.With().Info("server starting", slog.String("addr", s.server.Addr))

	if s.config.TLSCertPath == "" {
		err = s.server.ListenAndServe()
	} else {
		err = s.server.ListenAndServeTLS(s.config.TLSCertPath, s.config.TLSKeyPath)
	}

	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return errors.Wrapf(err, "serve failed")
}
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if err != nil {
		err = stdErr.Join(err, errors.Wrapf(s.server.Close(), "failed to close server"))
	}

	s.logger.Info("server closed")

	return errors.Wrapf(err, "server shutdown failed")
}

func (s *Server) Run() {
	go func() {
		err := s.Start()
		if err != nil {
			s.logger.With("error", err).Error("webserver crushed")
		}
	}()
}
