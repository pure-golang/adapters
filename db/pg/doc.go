// Package pg содержит адаптеры для PostgreSQL.
//
// Доступные реализации:
//   - db/pg/pgx  — нативный pgx драйвер (рекомендуется)
//   - db/pg/sqlx — sqlx поверх database/sql
//
// Обе реализации поддерживают:
//   - OpenTelemetry tracing
//   - структурированное логирование через slog
//   - именованные запросы и транзакции
//
// Использование (pgx):
//
//	import pgxadapter "github.com/pure-golang/adapters/db/pg/pgx"
//
//	db, err := pgxadapter.Connect(ctx, pgxadapter.ConfigFromEnv())
//
// Конфигурация через переменные окружения:
//
//	PG_DSN — строка подключения (postgres://user:pass@host/db)
package pg
