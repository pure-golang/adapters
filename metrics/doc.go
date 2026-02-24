// Package metrics предоставляет HTTP сервер для Prometheus метрик.
//
// Пакет запускает отдельный HTTP сервер с эндпоинтом /metrics
// для сбора метрик Prometheus.
//
// Использование:
//
//	import "github.com/pure-golang/adapters/metrics"
//
//	cfg := metrics.Config{
//	    Host: "0.0.0.0",
//	    Port: 9090,
//	}
//	closer, err := metrics.InitDefault(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer closer.Close()
//
// Конфигурация через переменные окружения:
//
//	METRICS_HOST        — хост сервера метрик (required)
//	METRICS_PORT        — порт сервера метрик (required)
//	METRICS_READ_TIMEOUT — таймаут чтения в секундах (default: 30)
//
// Особенности:
//   - Автоматическая инициализация Prometheus провайдера
//   - Эндпоинт /metrics для scrape
//   - Запуск сервера в горутине
//   - Graceful shutdown через Close()
package metrics
