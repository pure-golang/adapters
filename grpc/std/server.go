package std

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	adaptergrpc "github.com/pure-golang/adapters/grpc"
	"github.com/pure-golang/adapters/grpc/middleware"
	"github.com/pure-golang/adapters/logger"
)

const ShutdownTimeout = 15 * time.Second

var _ adaptergrpc.RunableProvider = (*Server)(nil)

type Config struct {
	Host          string `envconfig:"GRPC_HOST"`
	Port          int    `envconfig:"GRPC_PORT" required:"true"`
	TLSCertPath   string `envconfig:"GRPC_TLS_CERT_PATH"`
	TLSKeyPath    string `envconfig:"GRPC_TLS_KEY_PATH"`
	EnableReflect bool   `envconfig:"GRPC_ENABLE_REFLECTION" default:"true"`
}

type ServerOption func(*Server)

type Server struct {
	logger             *slog.Logger
	server             *grpc.Server
	config             Config
	interceptors       []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
	serverOpts         []grpc.ServerOption
	monitoringOpts     *middleware.MonitoringOptions
}

func WithUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) ServerOption {
	return func(s *Server) {
		s.interceptors = append(s.interceptors, interceptor)
	}
}

func WithStreamInterceptor(interceptor grpc.StreamServerInterceptor) ServerOption {
	return func(s *Server) {
		s.streamInterceptors = append(s.streamInterceptors, interceptor)
	}
}

func WithServerOption(opt grpc.ServerOption) ServerOption {
	return func(s *Server) {
		s.serverOpts = append(s.serverOpts, opt)
	}
}

// WithMonitoringOptions provides custom monitoring options
// If not set, DefaultMonitoringOptions will be used
func WithMonitoringOptions(opts *middleware.MonitoringOptions) ServerOption {
	return func(s *Server) {
		s.monitoringOpts = opts
	}
}

func NewDefault(c Config, registrationFunc func(*grpc.Server)) *Server {
	s := New(c, registrationFunc)
	return s
}

func New(c Config, registrationFunc func(*grpc.Server), opts ...ServerOption) *Server {
	s := &Server{
		logger:             logger.FromContext(context.Background()).WithGroup("grpcserver"),
		config:             c,
		interceptors:       []grpc.UnaryServerInterceptor{},
		streamInterceptors: []grpc.StreamServerInterceptor{},
		serverOpts:         []grpc.ServerOption{},
	}

	for _, opt := range opts {
		opt(s)
	}

	// Настраиваем мониторинг
	monitoringOptions := s.monitoringOpts
	if monitoringOptions == nil {
		monitoringOptions = middleware.DefaultMonitoringOptions(s.logger)
	}
	unaryInterceptors, streamInterceptors, monitoringOpts := middleware.SetupMonitoring(
		context.Background(),
		monitoringOptions,
	)

	// Добавляем пользовательские интерцепторы
	unaryInterceptors = append(unaryInterceptors, s.interceptors...)
	streamInterceptors = append(streamInterceptors, s.streamInterceptors...)

	// Настройки сервера
	serverOpts := make([]grpc.ServerOption, 0, len(monitoringOpts)+len(s.serverOpts))
	serverOpts = append(serverOpts, monitoringOpts...)
	serverOpts = append(serverOpts, s.serverOpts...)
	serverOpts = append(serverOpts,
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	)

	serverOpts = append(serverOpts, grpc.KeepaliveParams(keepalive.ServerParameters{
		// ... keepalive настройки
	}))

	// Настройка TLS если необходимо
	if c.TLSCertPath != "" && c.TLSKeyPath != "" {
		creds, err := credentials.NewServerTLSFromFile(c.TLSCertPath, c.TLSKeyPath)
		if err != nil {
			s.logger.With("error", err).Error("failed to create TLS credentials")
		} else {
			serverOpts = append(serverOpts, grpc.Creds(creds))
		}
	}

	// Создаем сервер
	s.server = grpc.NewServer(serverOpts...)

	// Регистрируем сервисы
	registrationFunc(s.server)

	// Добавляем reflection API если нужно
	if c.EnableReflect {
		reflection.Register(s.server)
	}

	return s
}

func (s *Server) listen() (net.Listener, error) {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info("gRPC server starting", slog.String("addr", addr))
	ln, err := net.Listen("tcp", addr)
	return ln, errors.Wrap(err, "failed to listen")
}

func (s *Server) serve(ln net.Listener) error {
	err := s.server.Serve(ln)
	if err == nil || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return errors.Wrap(err, "serve failed")
}

func (s *Server) Start() error {
	ln, err := s.listen()
	if err != nil {
		return err
	}
	go func() {
		if err := s.serve(ln); err != nil {
			s.logger.With("error", err).Error("gRPC server crashed")
		}
	}()
	return nil
}

func (s *Server) Run() {
	ln, err := s.listen()
	if err != nil {
		s.logger.With("error", err).Error("gRPC server failed to run")
		return
	}
	if err := s.serve(ln); err != nil {
		s.logger.With("error", err).Error("gRPC server crashed")
	}
}

func (s *Server) Shutdown() error {
	stopped := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	select {
	case <-stopped:
		s.logger.Info("gRPC server gracefully stopped")
	case <-ctx.Done():
		s.logger.Warn("gRPC server shutdown timeout exceeded, forcing stop")
		s.server.Stop()
	}

	return nil
}

func (s *Server) Close() error {
	return s.Shutdown()
}
