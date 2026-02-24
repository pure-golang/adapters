// Package redis реализует [kv.Store] для Redis.
//
// Использование:
//
//	client, err := redis.NewDefault(redis.Config{Addr: "localhost:6379"})
//	if err != nil {
//		return err
//	}
//	defer client.Close()
//
// Конфигурация через переменные окружения:
//
//	REDIS_ADDR               — адрес сервера (default: localhost:6379)
//	REDIS_PASSWORD           — пароль (default: пусто)
//	REDIS_DB                 — номер базы данных (default: 0)
//	REDIS_MAX_RETRIES        — количество повторов (default: 3)
//	REDIS_MIN_RETRY_BACKOFF  — мин. задержка между повторами (default: 8ms)
//	REDIS_MAX_RETRY_BACKOFF  — макс. задержка между повторами (default: 512ms)
//	REDIS_DIAL_TIMEOUT       — таймаут соединения (default: 5s)
//	REDIS_READ_TIMEOUT       — таймаут чтения (default: 3s)
//	REDIS_WRITE_TIMEOUT      — таймаут записи (default: 3s)
//	REDIS_POOL_SIZE          — размер пула соединений (default: 10)
//
// Thread-safe: да. Требует вызова [Client.Close] при завершении работы.
package redis
