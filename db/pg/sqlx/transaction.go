package sqlx

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Tx представляет транзакцию в базе данных
type Tx struct {
	tx  *sqlx.Tx
	cfg Config
}

// TxFunc определяет функцию, которая будет выполняться в рамках транзакции
type TxFunc func(ctx context.Context, tx *Tx) error

// TxOptions определяет опции транзакции
type TxOptions struct {
	Isolation  sql.IsolationLevel
	ReadOnly   bool
	Deferrable bool
}

// DefaultTxOptions возвращает опции транзакции по умолчанию
func DefaultTxOptions() *TxOptions {
	return &TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	}
}

// BeginTx начинает новую транзакцию с заданными опциями
func (c *Connection) BeginTx(ctx context.Context, opts *TxOptions) (*Tx, error) {
	var txOpts *sql.TxOptions
	if opts != nil {
		txOpts = &sql.TxOptions{
			Isolation: opts.Isolation,
			ReadOnly:  opts.ReadOnly,
		}
	}

	tx, err := c.DB.BeginTxx(ctx, txOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}

	return &Tx{
		tx:  tx,
		cfg: c.cfg,
	}, nil
}

// RunTx выполняет функцию в рамках транзакции
func (c *Connection) RunTx(ctx context.Context, opts *TxOptions, fn TxFunc) (err error) {
	tx, err := c.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	ctx, span := c.WithTracing(ctx, "RunTx", "")
	defer span.End()

	// Автоматический Rollback при панике или ошибке
	defer func() {
		if p := recover(); p != nil {
			rbErr := tx.Rollback()
			span.RecordError(rbErr)
			err = errors.Wrap(rbErr, "panic during transaction") // Сохраняем ошибку отката
			panic(p)                                             // Перебрасываем панику дальше
		} else if err != nil {
			rbErr := tx.Rollback()
			if rbErr != nil {
				span.RecordError(rbErr)
				err = errors.Wrap(err, rbErr.Error()) // Объединяем ошибки
			}
		}
	}()

	if err = fn(ctx, tx); err != nil {
		span.RecordError(err)
		return err // Rollback будет выполнен в defer
	}

	if err = tx.Commit(); err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// Commit фиксирует транзакцию
func (tx *Tx) Commit() error {
	_, span := tx.WithTracing(context.Background(), "Commit", "")
	defer span.End()

	if err := tx.tx.Commit(); err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
}

// Rollback откатывает транзакцию
func (tx *Tx) Rollback() error {
	_, span := tx.WithTracing(context.Background(), "Rollback", "")
	defer span.End()

	if err := tx.tx.Rollback(); err != nil && err != sql.ErrTxDone {
		span.RecordError(err)
		return errors.Wrap(err, "failed to rollback transaction")
	}
	return nil
}

// Get выполняет запрос в транзакции и заполняет одну запись
func (tx *Tx) Get(ctx context.Context, dst interface{}, query string, args ...interface{}) error {
	ctx, cancel := WithTimeout(ctx, tx.cfg.QueryTimeout)
	defer cancel()

	ctx, span := tx.WithTracing(ctx, "Get", query)
	defer span.End()

	err := tx.tx.GetContext(ctx, dst, query, args...)
	if err != nil {
		span.RecordError(err)
		if err == sql.ErrNoRows {
			return err
		}
		return errors.Wrap(err, "failed to execute get query in transaction")
	}
	return nil
}

// Select выполняет запрос в транзакции и заполняет срез записей
func (tx *Tx) Select(ctx context.Context, dst interface{}, query string, args ...interface{}) error {
	ctx, cancel := WithTimeout(ctx, tx.cfg.QueryTimeout)
	defer cancel()

	ctx, span := tx.WithTracing(ctx, "Select", query)
	defer span.End()

	err := tx.tx.SelectContext(ctx, dst, query, args...)
	if err != nil {
		span.RecordError(err)
		return errors.Wrap(err, "failed to execute select query in transaction")
	}
	return nil
}

// Exec выполняет запрос в транзакции и возвращает результат
func (tx *Tx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := WithTimeout(ctx, tx.cfg.QueryTimeout)
	defer cancel()

	ctx, span := tx.WithTracing(ctx, "Exec", query)
	defer span.End()

	result, err := tx.tx.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to execute query in transaction")
	}
	return result, nil
}

// Query выполняет запрос в транзакции и возвращает строки результата
func (tx *Tx) Query(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	ctx, cancel := WithTimeout(ctx, tx.cfg.QueryTimeout)
	defer cancel()

	ctx, span := tx.WithTracing(ctx, "Query", query)
	defer span.End()

	rows, err := tx.tx.QueryxContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to execute query in transaction")
	}
	return rows, nil
}

// QueryRow выполняет запрос в транзакции и возвращает одну строку результата
// Note: This method returns a lazy *sqlx.Row. The query is executed when Scan() is called.
// The context timeout is applied per the original context, as we cannot defer cleanup
// before Scan() is called by the caller.
func (tx *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	ctx, span := tx.WithTracing(ctx, "QueryRow", query)
	// End the span after the row is created (the actual query happens during Scan)
	// This is a limitation of the lazy evaluation pattern in sqlx.Row
	defer span.End()

	// Note: We don't apply QueryTimeout here because sqlx.Row is lazy-evaluated.
	// The query is executed when Scan() is called, so canceling the context here
	// would cause "context canceled" errors. The caller should manage context lifetime.
	return tx.tx.QueryRowxContext(ctx, query, args...)
}
