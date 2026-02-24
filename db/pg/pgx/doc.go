// Package pgx реализует адаптер PostgreSQL на базе pgx v5.
//
// Реализует интерфейс работы с базой данных через нативный pgx драйвер
// с поддержкой connection pooling, OpenTelemetry tracing и логирования.
//
// Использование:
//
//	import pgxadapter "github.com/pure-golang/adapters/db/pg/pgx"
//
//	cfg := pgxadapter.Config{
//	    Host:     "localhost",
//	    Port:     5432,
//	    User:     "postgres",
//	    Password: "secret",
//	    Database: "mydb",
//	}
//	db, err := pgxadapter.NewDefault(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
// Конфигурация через переменные окружения:
//
//	PG_HOST              — хост сервера (default: localhost)
//	PG_PORT              — порт сервера (default: 5432)
//	PG_USER              — пользователь БД
//	PG_PASSWORD          — пароль
//	PG_DATABASE          — имя базы данных
//	PG_SSLMODE           — режим SSL (default: disable)
//	PG_MAX_OPEN_CONNS    — макс. число соединений (default: 10)
//	PG_MAX_CONN_LIFETIME — время жизни соединения в секундах
//	PG_MAX_CONN_IDLE_TIME — время простоя соединения в секундах
//	PG_TRACE_LOG_LEVEL   — уровень логирования (debug, info, warn, error)
//
// Особенности:
//   - Использует pgxpool для управления пулом соединений
//   - Поддерживает OpenTelemetry tracing через otelpgx
//   - Автоматическое логирование запросов через tracelog
//   - Рекомендуется для новых проектов
package pgx
