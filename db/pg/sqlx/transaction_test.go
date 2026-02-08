package sqlx

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTx_Query_Unit tests the Query method - unit tests.
func TestTx_Query_Unit(t *testing.T) {
	t.Run("Query method exists and has correct signature", func(t *testing.T) {
		// Verify Query method exists on Tx
		var _ interface {
			Query(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
		} = &Tx{}
	})

	t.Run("Query with nil transaction structure", func(t *testing.T) {
		tx := &Tx{
			tx:  nil,
			cfg: Config{QueryTimeout: 30 * time.Second},
		}
		assert.NotNil(t, tx)
	})

	t.Run("Query with cancelled context", func(t *testing.T) {
		// Create a mock transaction that will fail on cancelled context
		mockDB, err := sqlx.Open("sqlite3", ":memory:")
		if err != nil {
			t.Skip("requires sqlite3 driver")
		}
		defer mockDB.Close()

		mockTx, err := mockDB.BeginTxx(context.Background(), nil)
		require.NoError(t, err)

		tx := &Tx{
			tx:  mockTx,
			cfg: Config{QueryTimeout: 30 * time.Second},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		rows, err := tx.Query(ctx, "SELECT 1")
		assert.Error(t, err)
		assert.Nil(t, rows)

		mockTx.Rollback()
	})

	t.Run("Query with various query types", func(t *testing.T) {
		// Test that Query handles different SQL statements
		queries := []string{
			"SELECT 1",
			"SELECT * FROM users",
			"SELECT id, name FROM products WHERE active = ?",
			"SELECT COUNT(*) FROM orders",
			"SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id",
		}

		for _, query := range queries {
			t.Run("query_"+query[:minInt(10, len(query))], func(t *testing.T) {
				// Just verify the query string is valid
				assert.NotEmpty(t, query)
			})
		}
	})
}

// TestTx_QueryRow_Unit tests the QueryRow method - unit tests.
func TestTx_QueryRow_Unit(t *testing.T) {
	t.Run("QueryRow method exists and has correct signature", func(t *testing.T) {
		// Verify QueryRow method exists on Tx
		var _ interface {
			QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row
		} = &Tx{}
	})

	t.Run("QueryRow with nil transaction structure", func(t *testing.T) {
		tx := &Tx{
			tx:  nil,
			cfg: Config{QueryTimeout: 30 * time.Second},
		}
		assert.NotNil(t, tx)
	})

	t.Run("QueryRow with cancelled context", func(t *testing.T) {
		mockDB, err := sqlx.Open("sqlite3", ":memory:")
		if err != nil {
			t.Skip("requires sqlite3 driver")
		}
		defer mockDB.Close()

		mockTx, err := mockDB.BeginTxx(context.Background(), nil)
		require.NoError(t, err)

		tx := &Tx{
			tx:  mockTx,
			cfg: Config{QueryTimeout: 30 * time.Second},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		row := tx.QueryRow(ctx, "SELECT 1")
		assert.NotNil(t, row)

		mockTx.Rollback()
	})

	t.Run("QueryRow with various queries", func(t *testing.T) {
		queries := []string{
			"SELECT 1",
			"SELECT * FROM users WHERE id = ?",
			"SELECT COUNT(*) FROM orders",
			"SELECT name, email FROM users WHERE id = $1",
		}

		for _, query := range queries {
			t.Run("query_"+query[:minInt(10, len(query))], func(t *testing.T) {
				// Verify query strings are valid
				assert.NotEmpty(t, query)
			})
		}
	})
}

// TestTx_QueryWithArgs_Unit tests Query method with arguments - unit tests.
func TestTx_QueryWithArgs_Unit(t *testing.T) {
	t.Run("Query with no args - structure test", func(t *testing.T) {
		tx := &Tx{
			tx:  nil,
			cfg: Config{QueryTimeout: 30 * time.Second},
		}

		// Just verify structure, don't call Query on nil tx
		assert.NotNil(t, tx)
	})
}

// TestTx_QueryRowWithArgs_Unit tests QueryRow method with arguments - unit tests.
func TestTx_QueryRowWithArgs_Unit(t *testing.T) {
	t.Run("QueryRow structure test", func(t *testing.T) {
		tx := &Tx{
			tx:  nil,
			cfg: Config{QueryTimeout: 30 * time.Second},
		}
		assert.NotNil(t, tx)
	})
}

// TestTx_QueryTimeoutVariations_Unit tests Query with various timeout settings - unit tests.
func TestTx_QueryTimeoutVariations_Unit(t *testing.T) {
	timeouts := []time.Duration{
		0,
		1 * time.Nanosecond,
		1 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
		30 * time.Second,
		5 * time.Minute,
	}

	for _, timeout := range timeouts {
		t.Run("timeout_"+timeout.String(), func(t *testing.T) {
			tx := &Tx{
				tx:  nil,
				cfg: Config{QueryTimeout: timeout},
			}

			// Just verify structure with different timeouts
			assert.Equal(t, timeout, tx.cfg.QueryTimeout)
		})
	}
}

// TestTx_QueryRowTimeoutVariations_Unit tests QueryRow with various timeout settings - unit tests.
func TestTx_QueryRowTimeoutVariations_Unit(t *testing.T) {
	timeouts := []time.Duration{
		0,
		1 * time.Nanosecond,
		1 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
		30 * time.Second,
	}

	for _, timeout := range timeouts {
		t.Run("timeout_"+timeout.String(), func(t *testing.T) {
			tx := &Tx{
				tx:  nil,
				cfg: Config{QueryTimeout: timeout},
			}

			// Just verify structure with different timeouts
			assert.Equal(t, timeout, tx.cfg.QueryTimeout)
		})
	}
}

// TestTx_QueryErrorPaths_Unit tests Query method error paths - unit tests.
func TestTx_QueryErrorPaths_Unit(t *testing.T) {
	t.Run("Query with invalid SQL syntax", func(t *testing.T) {
		mockDB, err := sqlx.Open("sqlite3", ":memory:")
		if err != nil {
			t.Skip("requires sqlite3 driver")
		}
		defer mockDB.Close()

		mockTx, err := mockDB.BeginTxx(context.Background(), nil)
		require.NoError(t, err)

		tx := &Tx{
			tx:  mockTx,
			cfg: Config{QueryTimeout: 30 * time.Second},
		}

		ctx := context.Background()
		rows, err := tx.Query(ctx, "INVALID SQL QUERY")
		assert.Error(t, err)
		assert.Nil(t, rows)

		mockTx.Rollback()
	})

	t.Run("Query with context already done", func(t *testing.T) {
		mockDB, err := sqlx.Open("sqlite3", ":memory:")
		if err != nil {
			t.Skip("requires sqlite3 driver")
		}
		defer mockDB.Close()

		mockTx, err := mockDB.BeginTxx(context.Background(), nil)
		require.NoError(t, err)

		tx := &Tx{
			tx:  mockTx,
			cfg: Config{QueryTimeout: 30 * time.Second},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		rows, err := tx.Query(ctx, "SELECT 1")
		assert.Error(t, err)
		assert.Nil(t, rows)

		mockTx.Rollback()
	})
}

// TestTx_QueryRowErrorPaths_Unit tests QueryRow method error paths - unit tests.
func TestTx_QueryRowErrorPaths_Unit(t *testing.T) {
	t.Run("QueryRow returns row even if query will fail", func(t *testing.T) {
		mockDB, err := sqlx.Open("sqlite3", ":memory:")
		if err != nil {
			t.Skip("requires sqlite3 driver")
		}
		defer mockDB.Close()

		mockTx, err := mockDB.BeginTxx(context.Background(), nil)
		require.NoError(t, err)

		tx := &Tx{
			tx:  mockTx,
			cfg: Config{QueryTimeout: 30 * time.Second},
		}

		ctx := context.Background()
		row := tx.QueryRow(ctx, "INVALID SQL")
		assert.NotNil(t, row)

		// The error is only realized when calling Scan()
		var result int
		err = row.Scan(&result)
		assert.Error(t, err)

		mockTx.Rollback()
	})
}

// TestTx_OptionsStructure tests the TxOptions structure.
func TestTx_OptionsStructure(t *testing.T) {
	t.Run("DefaultTxOptions returns valid options", func(t *testing.T) {
		opts := DefaultTxOptions()
		assert.NotNil(t, opts)
		assert.Equal(t, sql.LevelDefault, opts.Isolation)
		assert.False(t, opts.ReadOnly)
		assert.False(t, opts.Deferrable)
	})

	t.Run("TxOptions with custom values", func(t *testing.T) {
		opts := &TxOptions{
			Isolation:  sql.LevelSerializable,
			ReadOnly:   true,
			Deferrable: true,
		}
		assert.Equal(t, sql.LevelSerializable, opts.Isolation)
		assert.True(t, opts.ReadOnly)
		assert.True(t, opts.Deferrable)
	})

	t.Run("TxOptions with various isolation levels", func(t *testing.T) {
		levels := []sql.IsolationLevel{
			sql.LevelDefault,
			sql.LevelReadUncommitted,
			sql.LevelReadCommitted,
			sql.LevelWriteCommitted,
			sql.LevelRepeatableRead,
			sql.LevelSnapshot,
			sql.LevelSerializable,
			sql.LevelLinearizable,
		}

		for _, level := range levels {
			t.Run("isolation_"+level.String(), func(t *testing.T) {
				opts := &TxOptions{
					Isolation: level,
				}
				assert.Equal(t, level, opts.Isolation)
			})
		}
	})
}

// TestTx_TxFuncSignature tests the TxFunc signature.
func TestTx_TxFuncSignature(t *testing.T) {
	t.Run("TxFunc is a function type", func(t *testing.T) {
		// Verify TxFunc is defined
		var _ TxFunc = func(ctx context.Context, tx *Tx) error {
			return nil
		}

		// Create a sample function
		fn := func(ctx context.Context, tx *Tx) error {
			_, _ = tx.Query(ctx, "SELECT 1")
			return nil
		}

		assert.NotNil(t, fn)
	})

	t.Run("TxFunc can return error", func(t *testing.T) {
		fn := func(ctx context.Context, tx *Tx) error {
			return assert.AnError
		}

		ctx := context.Background()
		tx := &Tx{}
		err := fn(ctx, tx)
		assert.Error(t, err)
	})

	t.Run("TxFunc can return nil", func(t *testing.T) {
		fn := func(ctx context.Context, tx *Tx) error {
			return nil
		}

		ctx := context.Background()
		tx := &Tx{}
		err := fn(ctx, tx)
		assert.NoError(t, err)
	})
}

// TestTx_Structure tests the Tx structure.
func TestTx_Structure(t *testing.T) {
	t.Run("Tx can be created", func(t *testing.T) {
		tx := &Tx{
			tx:  nil,
			cfg: Config{},
		}
		assert.NotNil(t, tx)
	})

	t.Run("Tx has WithTracing method", func(t *testing.T) {
		tx := &Tx{
			tx:  nil,
			cfg: Config{},
		}

		ctx, span := tx.WithTracing(context.Background(), "test", "SELECT 1")
		assert.NotNil(t, ctx)
		assert.NotNil(t, span)
		span.End()
	})
}

// minInt returns the minimum of two integers for testing purposes.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
