package std

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/pure-golang/adapters/httpserver"
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

func New(c Config, h http.Handler /* ext: option functions */) *Server {
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

func (s *Server) listen() (net.Listener, error) {
	s.logger.Info("server starting", slog.String("addr", s.server.Addr))
	ln, err := net.Listen("tcp", s.server.Addr)
	return ln, errors.Wrap(err, "failed to listen")
}

func (s *Server) serve(ln net.Listener) error {
	var err error

	if s.config.TLSCertPath == "" {
		err = s.server.Serve(ln)
	} else {
		err = s.server.ServeTLS(ln, s.config.TLSCertPath, s.config.TLSKeyPath)
	}

	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return errors.Wrapf(err, "serve failed")
}

func (s *Server) Start() error {
	ln, err := s.listen()
	if err != nil {
		return err
	}
	go func() {
		if err := s.serve(ln); err != nil {
			s.logger.With("error", err).Error("webserver crushed")
		}
	}()
	return nil
}

func (s *Server) Run() {
	ln, err := s.listen()
	if err != nil {
		s.logger.With("error", err).Error("webserver failed to run")
		return
	}
	if err := s.serve(ln); err != nil {
		s.logger.With("error", err).Error("webserver crushed")
	}
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	if err != nil {
		err = stderrors.Join(err, errors.Wrapf(s.server.Close(), "failed to close server"))
	}

	s.logger.Info("server closed")

	return errors.Wrapf(err, "server shutdown failed")
}

func (s *Server) Close() error {
	return s.Shutdown()
}
