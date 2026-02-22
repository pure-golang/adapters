package sqlx

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Стандартный драйвер PostgreSQL
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
)

// Connection представляет соединение с базой данных PostgreSQL через sqlx
type Connection struct {
	*sqlx.DB
	cfg Config
}

// Connect создает новое соединение с базой данных PostgreSQL
func Connect(ctx context.Context, cfg Config) (*Connection, error) {
	ctx, span := tracer.Start(ctx, "sqlx.Connect")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.host", cfg.Host),
		attribute.Int("db.port", cfg.Port),
		attribute.String("db.name", cfg.Database),
		attribute.String("db.user", cfg.User),
	)

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	if cfg.ConnectTimeout > 0 {
		dsn += fmt.Sprintf(" connect_timeout=%d", cfg.ConnectTimeout)
	}

	dsn += " application_name=sqlx"

	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to connect to PostgreSQL")
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
		span.SetAttributes(attribute.Int("db.max_open_conns", cfg.MaxOpenConns))
	}

	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
		span.SetAttributes(attribute.Int("db.max_idle_conns", cfg.MaxIdleConns))
	}

	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
		span.SetAttributes(attribute.String("db.conn_max_lifetime", cfg.ConnMaxLifetime.String()))
	}

	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
		span.SetAttributes(attribute.String("db.conn_max_idle_time", cfg.ConnMaxIdleTime.String()))
	}

	// Проверка соединения
	if err := db.PingContext(ctx); err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to ping PostgreSQL")
	}

	return &Connection{
		DB:  db,
		cfg: cfg,
	}, nil
}

// Close закрывает соединение с базой данных
func (c *Connection) Close() error {
	_, span := tracer.Start(context.Background(), "sqlx.Close")
	defer span.End()

	if err := c.DB.Close(); err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to close database connection")
	}
	return nil
}
