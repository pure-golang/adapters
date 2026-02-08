package sqlx

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

// Querier определяет интерфейс для выполнения запросов к базе данных
type Querier interface {
	Get(ctx context.Context, dst interface{}, query string, args ...interface{}) error
	Select(ctx context.Context, dst interface{}, query string, args ...interface{}) error
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row
	NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error)
}

// Get выполняет запрос и заполняет одну запись
func (c *Connection) Get(ctx context.Context, dst interface{}, query string, args ...interface{}) error {
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)
	defer cancel()

	ctx, span := c.WithTracing(ctx, "Get", query)
	defer span.End()

	err := c.DB.GetContext(ctx, dst, query, args...)
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
func (c *Connection) Select(ctx context.Context, dst interface{}, query string, args ...interface{}) error {
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)
	defer cancel()

	ctx, span := c.WithTracing(ctx, "Select", query)
	defer span.End()

	err := c.DB.SelectContext(ctx, dst, query, args...)
	if err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to execute select query")
	}
	return nil
}

// Exec выполняет запрос и возвращает результат
func (c *Connection) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)
	defer cancel()

	ctx, span := c.WithTracing(ctx, "Exec", query)
	defer span.End()

	result, err := c.DB.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to execute query")
	}
	return result, nil
}

// Query выполняет запрос и возвращает строки результата
func (c *Connection) Query(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)
	defer cancel()

	ctx, span := c.WithTracing(ctx, "Query", query)
	defer span.End()

	rows, err := c.DB.QueryxContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to execute query")
	}
	return rows, nil
}

// QueryRow выполняет запрос и возвращает одну строку результата
func (c *Connection) QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	ctx, span := c.WithTracing(ctx, "QueryRow", query)
	defer span.End()

	// Note: We don't apply QueryTimeout here because sqlx.Row is lazy-evaluated.
	// The query is executed when Scan() is called, so canceling the context here
	// would cause "context canceled" errors. The caller should manage context lifetime.
	return c.DB.QueryRowxContext(ctx, query, args...)
}

// NamedExec выполняет именованный запрос
func (c *Connection) NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)
	defer cancel()

	ctx, span := c.WithTracing(ctx, "NamedExec", query)
	defer span.End()

	result, err := c.DB.NamedExecContext(ctx, query, arg)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to execute named query")
	}
	return result, nil
}

// NamedQuery выполняет именованный запрос и возвращает строки результата
func (c *Connection) NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	// Не отменяем контекст пока rows не будут закрыты
	// Вызывающий должен закрыть rows через defer rows.Close()
	ctx, cancel := WithTimeout(ctx, c.cfg.QueryTimeout)

	ctx, span := c.WithTracing(ctx, "NamedQuery", query)

	rows, err := c.DB.NamedQueryContext(ctx, query, arg)
	if err != nil {
		cancel()
		span.RecordError(err)
		span.End()
		return nil, errors.Wrap(err, "failed to execute named query")
	}

	// Wrap rows to cancel context and end span when closed
	wrappedRows := &namedQueryRows{
		Rows:   rows,
		cancel: cancel,
		span:   span,
		closed: false,
	}
	return wrappedRows.Rows, nil
}

// namedQueryRows wraps sqlx.Rows to cleanup context and span
type namedQueryRows struct {
	*sqlx.Rows
	cancel context.CancelFunc
	span   trace.Span
	closed bool
	mu     sync.Mutex
}

func (r *namedQueryRows) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	r.cancel()
	r.span.End()
	return r.Rows.Close()
}

// WithTimeout добавляет таймаут к контексту
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}
