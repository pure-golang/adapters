package sqlx

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Querier определяет интерфейс для выполнения запросов к базе данных
type Querier interface {
	Get(ctx context.Context, dst any, query string, args ...any) error
	Select(ctx context.Context, dst any, query string, args ...any) error
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sqlx.Row
	NamedExec(ctx context.Context, query string, arg any) (sql.Result, error)
	NamedQuery(ctx context.Context, query string, arg any) (*sqlx.Rows, error)
}

// Get выполняет запрос и заполняет одну запись
func (c *Connection) Get(ctx context.Context, dst any, query string, args ...any) error {
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)
	defer cancel()

	ctx, span := c.WithTracing(ctx, "Get", query)
	defer span.End()

	err := c.GetContext(ctx, dst, query, args...)
	if err != nil {
		span.RecordError(err)
		if err == sql.ErrNoRows {
			return err
		}
		return errors.Wrap(err, "failed to execute get query")
	}
	return nil
}

// Select выполняет запрос и заполняет срез записей
func (c *Connection) Select(ctx context.Context, dst any, query string, args ...any) error {
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)
	defer cancel()

	ctx, span := c.WithTracing(ctx, "Select", query)
	defer span.End()

	err := c.SelectContext(ctx, dst, query, args...)
	if err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to execute select query")
	}
	return nil
}

// Exec выполняет запрос и возвращает результат
func (c *Connection) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)
	defer cancel()

	ctx, span := c.WithTracing(ctx, "Exec", query)
	defer span.End()

	result, err := c.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to execute query")
	}
	return result, nil
}

// Query выполняет запрос и возвращает строки результата.
// QueryTimeout не применяется, так как rows потребляются после возврата.
// Вызывающий управляет временем жизни контекста и должен закрыть rows через defer rows.Close().
func (c *Connection) Query(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	ctx, span := c.WithTracing(ctx, "Query", query)
	defer span.End()

	rows, err := c.QueryxContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to execute query")
	}
	return rows, nil
}

// QueryRow выполняет запрос и возвращает одну строку результата
func (c *Connection) QueryRow(ctx context.Context, query string, args ...any) *sqlx.Row {
	ctx, span := c.WithTracing(ctx, "QueryRow", query)
	defer span.End()

	// Note: We don't apply QueryTimeout here because sqlx.Row is lazy-evaluated.
	// The query is executed when Scan() is called, so canceling the context here
	// would cause "context canceled" errors. The caller should manage context lifetime.
	return c.QueryRowxContext(ctx, query, args...)
}

// NamedExec выполняет именованный запрос
func (c *Connection) NamedExec(ctx context.Context, query string, arg any) (sql.Result, error) {
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)
	defer cancel()

	ctx, span := c.WithTracing(ctx, "NamedExec", query)
	defer span.End()

	result, err := c.NamedExecContext(ctx, query, arg)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to execute named query")
	}
	return result, nil
}

// NamedQuery выполняет именованный запрос и возвращает строки результата.
// QueryTimeout не применяется, так как rows потребляются после возврата.
// Вызывающий управляет временем жизни контекста и должен закрыть rows через defer rows.Close().
func (c *Connection) NamedQuery(ctx context.Context, query string, arg any) (*sqlx.Rows, error) {
	ctx, span := c.WithTracing(ctx, "NamedQuery", query)
	defer span.End()

	rows, err := c.NamedQueryContext(ctx, query, arg)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to execute named query")
	}
	return rows, nil
}

// WithTimeout добавляет таймаут к контексту
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}
