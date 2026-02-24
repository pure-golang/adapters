// Package std реализует [httpserver.RunableProvider] для стандартного HTTP сервера.
//
// Поддерживает:
//   - TLS шифрование
//   - gracefull shutdown с таймаутом
//   - конфигурируемый read timeout
//
// Использование:
//
//	import httpstd "github.com/pure-golang/adapters/httpserver/std"
//
//	cfg := httpstd.Config{
//	    Port: 8080,
//	}
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/", handler)
//
//	server := httpstd.NewDefault(cfg, mux)
//
//	// Запуск в горутине
//	server.Run()
//	defer server.Close()
//
// Конфигурация через переменные окружения:
//
//	WEBSERVER_HOST           — хост сервера (default: "")
//	WEBSERVER_PORT           — порт сервера (required)
//	WEBSERVER_TLS_CERT_PATH  — путь к TLS сертификату
//	WEBSERVER_TLS_KEY_PATH   — путь к TLS ключу
//	WEBSERVER_READ_TIMEOUT   — таймаут чтения в секундах (default: 30)
//
// Особенности:
//   - ReadHeaderTimeout установлен в 10s для защиты от Slowloris атак
//   - Graceful shutdown с таймаутом 15 секунд
//   - При shutdown timeout принудительно закрывает соединения
package std
