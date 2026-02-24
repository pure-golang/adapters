// Package grpc определяет интерфейсы для gRPC серверов.
//
// Пакет предоставляет базовые интерфейсы для gRPC компонентов.
// Реализации находятся в дочерних пакетах:
//   - [grpc/std] — стандартная реализация gRPC сервера
//   - [grpc/middleware] — интерцепторы для мониторинга
//   - [grpc/errors] — утилиты для обработки ошибок
//
// Интерфейсы:
//   - [Provider] — запуск и остановка gRPC сервера
//   - [Runner] — запуск сервера в горутине
//   - [RunableProvider] — объединение Provider и Runner
//
// Использование:
//
//	var server grpc.RunableProvider = std.NewDefault(cfg, registrationFunc)
//	server.Run()  // запуск в горутине
//	defer server.Close()
package grpc
