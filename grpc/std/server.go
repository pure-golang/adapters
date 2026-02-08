package std

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	adaptergrpc "github.com/pure-golang/adapters/grpc"
	"github.com/pure-golang/adapters/grpc/middleware"
	"github.com/pure-golang/adapters/logger"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
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
	listener           net.Listener
	listenerMu         sync.RWMutex
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
	serverOpts := append(monitoringOpts, s.serverOpts...)
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

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "failed to listen on %s", addr)
	}

	s.listenerMu.Lock()
	s.listener = lis
	s.listenerMu.Unlock()

	s.logger.Info("gRPC server starting", "addr", addr)

	err = s.server.Serve(lis)
	if err != nil && !errors.Is(err, net.ErrClosed) {
		return errors.Wrap(err, "failed to serve gRPC")
	}

	return nil
}

func (s *Server) Close() error {
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

	s.listenerMu.RLock()
	listener := s.listener
	s.listenerMu.RUnlock()

	if listener != nil {
		err := listener.Close()
		if err != nil {
			return errors.Wrap(err, "failed to close listener")
		}
	}

	return nil
}

func (s *Server) Run() {
	go func() {
		err := s.Start()
		if err != nil {
			s.logger.With("error", err).Error("gRPC server crashed")
		}
	}()
}

// GetListener returns the server's listener in a thread-safe manner.
// This is primarily used in tests to check if the listener has been set.
func (s *Server) GetListener() net.Listener {
	s.listenerMu.RLock()
	defer s.listenerMu.RUnlock()
	return s.listener
}
