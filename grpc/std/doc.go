// Package std реализует [grpc.RunableProvider] для стандартного gRPC сервера.
//
// Поддерживает:
//   - автоматическое подключение мониторинга (tracing, metrics, logging)
//   - TLS шифрование
//   - gracefull shutdown
//   - gRPC reflection
//
// Использование:
//
//	import grpcstd "github.com/pure-golang/adapters/grpc/std"
//
//	cfg := grpcstd.Config{
//	    Port: 50051,
//	    EnableReflect: true,
//	}
//
//	server := grpcstd.NewDefault(cfg, func(s *grpc.Server) {
//	    pb.RegisterMyServiceServer(s, myServiceImpl)
//	})
//
//	// Запуск в горутине
//	server.Run()
//	defer server.Close()
//
// Конфигурация через переменные окружения:
//
//	GRPC_HOST              — хост сервера (default: "")
//	GRPC_PORT              — порт сервера (required)
//	GRPC_TLS_CERT_PATH     — путь к TLS сертификату
//	GRPC_TLS_KEY_PATH      — путь к TLS ключу
//	GRPC_ENABLE_REFLECTION — включить reflection API (default: true)
//
// Особенности:
//   - По умолчанию включает tracing, metrics и logging через SetupMonitoring
//   - Graceful shutdown с таймаутом 15 секунд
//   - Поддержка кастомных интерцепторов через WithUnaryInterceptor
//   - Потокобезопасное управление listener'ом
package std
