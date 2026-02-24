// Package httpserver определяет интерфейсы для HTTP серверов.
//
// Пакет предоставляет базовые интерфейсы для HTTP компонентов.
// Реализации находятся в дочерних пакетах:
//   - [httpserver/std] — стандартная реализация HTTP сервера
//   - [httpserver/middleware] — middleware для мониторинга
//
// Интерфейсы:
//   - [Provider] — запуск и остановка HTTP сервера
//   - [Runner] — запуск сервера в горутине
//   - [RunableProvider] — объединение Provider и Runner
//
// Использование:
//
//	var server httpserver.RunableProvider = std.NewDefault(cfg, handler)
//	server.Run()  // запуск в горутине
//	defer server.Close()
package httpserver
