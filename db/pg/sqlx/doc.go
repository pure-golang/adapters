// Package sqlx реализует адаптер PostgreSQL на базе sqlx поверх database/sql.
//
// Реализует интерфейс работы с базой данных через sqlx библиотеку
// с поддержкой именованных запросов, транзакций, OpenTelemetry tracing.
//
// Использование:
//
//	import sqlxadapter "github.com/pure-golang/adapters/db/pg/sqlx"
//
//	cfg := sqlxadapter.Config{
//	    Host:     "localhost",
//	    Port:     5432,
//	    User:     "postgres",
//	    Password: "secret",
//	    Database: "mydb",
//	    SSLMode:  "disable",
//	}
//	db, err := sqlxadapter.Connect(context.Background(), cfg)
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
//	PG_CONNECT_TIMEOUT   — таймаут подключения в секундах (default: 5)
//	PG_MAX_OPEN_CONNS    — макс. число соединений
//	PG_MAX_IDLE_CONNS    — макс. число простаивающих соединений
//	PG_CONN_MAX_LIFETIME — время жизни соединения
//	PG_CONN_MAX_IDLE_TIME — время простоя соединения
//	PG_QUERY_TIMEOUT     — таймаут запросов (default: 10s)
//
// Особенности:
//   - Именованные запросы через NamedExec и NamedQuery
//   - Транзакции с автоматическим откатом при ошибке (RunTx)
//   - OpenTelemetry tracing для всех операций
//   - Хелперы для проверки constraint ошибок (IsUniqueViolation, etc.)
package sqlx
