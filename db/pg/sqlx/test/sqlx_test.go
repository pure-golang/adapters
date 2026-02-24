package sqlx_test

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/pure-golang/adapters/db/pg/sqlx"
)

var testDB *sqlx.Connection
var testCfg sqlx.Config

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		fmt.Println("integration test")
		os.Exit(0)
	}

	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_DB":       "test_db",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Printf("Could not start container: %s", err)
		return 1
	}

	defer func() {
		if err := container.Terminate(ctx); err != nil {
			fmt.Printf("Warning: could not terminate container: %s\n", err)
		}
	}()

	host, err := container.Host(ctx)
	if err != nil {
		log.Printf("Could not get container host: %s", err)
		return 1
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		log.Printf("Could not get container port: %s", err)
		return 1
	}

	port, err := strconv.Atoi(mappedPort.Port())
	if err != nil {
		log.Printf("Could not parse port: %s", err)
		return 1
	}

	testCfg = sqlx.Config{
		Host:           host,
		Port:           port,
		User:           "test_user",
		Password:       "secret",
		Database:       "test_db",
		SSLMode:        "disable",
		ConnectTimeout: 5,
		QueryTimeout:   30 * time.Second,
	}

	maxRetries := 30
	retryInterval := 500 * time.Millisecond

	for i := range maxRetries {
		testDB, err = sqlx.Connect(ctx, testCfg)
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(retryInterval)
		}
	}
	if err != nil {
		log.Printf("Could not connect to database after %d retries: %s", maxRetries, err)
		return 1
	}

	code := m.Run()

	if testDB != nil {
		if err := testDB.Close(); err != nil {
			fmt.Printf("Warning: failed to close test DB: %s\n", err)
		}
	}

	return code
}

func TestConnection_Exec(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL
		)
	`)
	require.NoError(t, err)

	result, err := testDB.Exec(ctx, `
		INSERT INTO test_users (name, email) VALUES ($1, $2)
	`, "John Doe", "john@example.com")
	require.NoError(t, err)

	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_users (name, email) VALUES ($1, $2)
	`, "Jane Doe", "john@example.com")
	require.Error(t, err)
	require.True(t, sqlx.IsUniqueViolation(err))
}

func TestConnection_Transaction(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_accounts (
			id SERIAL PRIMARY KEY,
			balance INTEGER NOT NULL CHECK (balance >= 0)
		)
	`)
	require.NoError(t, err)

	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO test_accounts (balance) VALUES ($1), ($2)
		`, 100, 0)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, "UPDATE test_accounts SET balance = balance - $1 WHERE id = 1", 50)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE test_accounts SET balance = balance + $1 WHERE id = 2", 50)
		return err
	})
	require.NoError(t, err)

	var balance1, balance2 int
	err = testDB.Get(ctx, &balance1, "SELECT balance FROM test_accounts WHERE id = 1")
	require.NoError(t, err)
	err = testDB.Get(ctx, &balance2, "SELECT balance FROM test_accounts WHERE id = 2")
	require.NoError(t, err)

	require.Equal(t, 50, balance1)
	require.Equal(t, 50, balance2)

	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE test_accounts SET balance = balance - $1 WHERE id = 1;
		`, 100)
		return err
	})
	require.Error(t, err)

	err = testDB.Get(ctx, &balance1, "SELECT balance FROM test_accounts WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, 50, balance1)
}

func TestConnection_Get(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_items (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			price INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_items (name, price) VALUES ($1, $2)
	`, "test item", 100)
	require.NoError(t, err)

	type Item struct {
		ID    int    `db:"id"`
		Name  string `db:"name"`
		Price int    `db:"price"`
	}

	var item Item
	err = testDB.Get(ctx, &item, "SELECT * FROM test_items WHERE id = $1", 1)
	require.NoError(t, err)
	require.Equal(t, "test item", item.Name)
	require.Equal(t, 100, item.Price)

	err = testDB.Get(ctx, &item, "SELECT * FROM test_items WHERE id = $1", 999)
	require.Error(t, err)
}

func TestConnection_Select(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_products (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			category TEXT NOT NULL,
			price INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_products (name, category, price) VALUES
		($1, $2, $3),
		($4, $5, $6)
	`, "Product 1", "A", 100, "Product 2", "B", 200)
	require.NoError(t, err)

	type Product struct {
		ID       int    `db:"id"`
		Name     string `db:"name"`
		Category string `db:"category"`
		Price    int    `db:"price"`
	}

	var products []Product
	err = testDB.Select(ctx, &products, "SELECT * FROM test_products ORDER BY id")
	require.NoError(t, err)
	require.Len(t, products, 2)
	require.Equal(t, "Product 1", products[0].Name)
	require.Equal(t, "Product 2", products[1].Name)

	err = testDB.Select(ctx, &products, "SELECT * FROM test_products WHERE category = $1", "A")
	require.NoError(t, err)
	require.Len(t, products, 1)
	require.Equal(t, "Product 1", products[0].Name)
}

func TestConnection_QueryTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE OR REPLACE FUNCTION test_delay() RETURNS void AS $$
		BEGIN
			PERFORM pg_sleep(2);
		END;
		$$ LANGUAGE plpgsql;
	`)
	require.NoError(t, err)

	shortTimeoutCfg := testCfg
	shortTimeoutCfg.QueryTimeout = 100 * time.Millisecond
	shortTimeoutDB, err := sqlx.Connect(ctx, shortTimeoutCfg)
	require.NoError(t, err)
	t.Cleanup(func() { shortTimeoutDB.Close() })

	_, err = shortTimeoutDB.Exec(ctx, "SELECT test_delay()")
	require.Error(t, err)
	require.Contains(t, err.Error(), "canceling statement due to user request")
}

func TestConnection_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_constraints (
			id SERIAL PRIMARY KEY,
			code TEXT UNIQUE NOT NULL,
			value INTEGER CHECK (value > 0),
			ref_id INTEGER REFERENCES test_accounts(id)
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (code, value) VALUES ($1, $2)
	`, "CODE1", 100)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (code, value) VALUES ($1, $2)
	`, "CODE1", 200)
	require.Error(t, err)
	require.True(t, sqlx.IsUniqueViolation(err))

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (code, value) VALUES ($1, $2)
	`, "CODE2", -1)
	require.Error(t, err)
	require.True(t, sqlx.IsCheckViolation(err))

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (code, value, ref_id) VALUES ($1, $2, $3)
	`, "CODE3", 100, 999)
	require.Error(t, err)
	require.True(t, sqlx.IsForeignKeyViolation(err))

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (value) VALUES ($1)
	`, 100)
	require.Error(t, err)
	require.True(t, sqlx.IsNotNullViolation(err))
}

func TestConnection_NamedQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	createCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	_, err := testDB.Exec(createCtx, `
		CREATE TABLE IF NOT EXISTS test_orders (
			id SERIAL PRIMARY KEY,
			customer_name TEXT NOT NULL,
			total_amount INTEGER NOT NULL,
			status TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	type Order struct {
		ID           int    `db:"id"`
		CustomerName string `db:"customer_name"`
		TotalAmount  int    `db:"total_amount"`
		Status       string `db:"status"`
	}

	order := Order{
		CustomerName: "John Doe",
		TotalAmount:  1000,
		Status:       "new",
	}

	insertCtx, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel2)

	result, err := testDB.NamedExec(insertCtx, `
		INSERT INTO test_orders (customer_name, total_amount, status)
		VALUES (:customer_name, :total_amount, :status)
	`, order)
	require.NoError(t, err)

	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	type OrderQuery struct {
		MinAmount int    `db:"min_amount"`
		Status    string `db:"status"`
	}

	query := OrderQuery{
		MinAmount: 500,
		Status:    "new",
	}

	queryCtx, cancel3 := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel3)

	rows, err := testDB.NamedQuery(queryCtx, `
		SELECT * FROM test_orders
		WHERE total_amount >= :min_amount
		AND status = :status
	`, query)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := rows.Close(); err != nil {
			t.Errorf("failed to close rows: %v", err)
		}
	})

	var orders []Order
	for rows.Next() {
		var o Order
		err := rows.StructScan(&o)
		require.NoError(t, err)
		orders = append(orders, o)
	}
	require.NoError(t, rows.Err())
	require.Len(t, orders, 1)
	require.Equal(t, "John Doe", orders[0].CustomerName)
}

func TestConnection_TransactionIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_isolation (
			id SERIAL PRIMARY KEY,
			value INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `INSERT INTO test_isolation (value) VALUES ($1)`, 100)
	require.NoError(t, err)

	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx1 *sqlx.Tx) error {
		var value1 int
		err := tx1.Get(ctx, &value1, "SELECT value FROM test_isolation WHERE id = 1")
		require.NoError(t, err)
		require.Equal(t, 100, value1)

		err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx2 *sqlx.Tx) error {
			_, err := tx2.Exec(ctx, "UPDATE test_isolation SET value = $1 WHERE id = 1", 200)
			return err
		})
		require.NoError(t, err)

		err = tx1.Get(ctx, &value1, "SELECT value FROM test_isolation WHERE id = 1")
		require.NoError(t, err)
		require.Equal(t, 200, value1)

		return nil
	})
	require.NoError(t, err)

	opts := &sqlx.TxOptions{Isolation: sql.LevelRepeatableRead}
	err = testDB.RunTx(ctx, opts, func(ctx context.Context, tx1 *sqlx.Tx) error {
		var value1 int
		err := tx1.Get(ctx, &value1, "SELECT value FROM test_isolation WHERE id = 1")
		require.NoError(t, err)
		require.Equal(t, 200, value1)

		err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx2 *sqlx.Tx) error {
			_, err := tx2.Exec(ctx, "UPDATE test_isolation SET value = $1 WHERE id = 1", 300)
			return err
		})
		require.NoError(t, err)

		err = tx1.Get(ctx, &value1, "SELECT value FROM test_isolation WHERE id = 1")
		require.NoError(t, err)
		require.Equal(t, 200, value1)

		return nil
	})
	require.NoError(t, err)
}

func TestConnection_ConcurrentTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_concurrent (
			id INTEGER PRIMARY KEY,
			counter INTEGER NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `INSERT INTO test_concurrent (id, counter) VALUES (1, 0)`)
	require.NoError(t, err)

	const goroutines = 10
	const iterations = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	var (
		errMu sync.Mutex
		errs  []error
	)

	for range goroutines {
		go func() {
			defer wg.Done()

			for range iterations {
				err := testDB.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
					var counter int
					err := tx.Get(ctx, &counter, "SELECT counter FROM test_concurrent WHERE id = 1 FOR UPDATE")
					if err != nil {
						return err
					}

					_, err = tx.Exec(ctx, "UPDATE test_concurrent SET counter = $1 WHERE id = 1", counter+1)
					return err
				})
				if err != nil {
					errMu.Lock()
					errs = append(errs, err)
					errMu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	require.Empty(t, errs, "got unexpected errors")

	var finalCounter int
	err = testDB.Get(ctx, &finalCounter, "SELECT counter FROM test_concurrent WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, goroutines*iterations, finalCounter)
}

func TestConnection_Query(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_query (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_query (name, value) VALUES
		($1, $2), ($3, $4), ($5, $6)
	`, "item1", 10, "item2", 20, "item3", 30)
	require.NoError(t, err)

	rows, err := testDB.Query(ctx, "SELECT * FROM test_query ORDER BY id")
	require.NoError(t, err)
	require.NotNil(t, rows)
	rows.Close()

	_, err = testDB.Query(ctx, "SELECT * FROM nonexistent_table")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to execute query")
}

func TestConnection_QueryRow(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_query_row (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			age INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_query_row (name, age) VALUES ($1, $2)
	`, "Alice", 30)
	require.NoError(t, err)

	row := testDB.QueryRow(ctx, "SELECT * FROM test_query_row WHERE name = $1", "Alice")
	require.NotNil(t, row)

	row = testDB.QueryRow(ctx, "SELECT * FROM test_query_row WHERE name = $1", "Bob")
	var name string
	var age int
	err = row.Scan(&name, &age)
	require.Error(t, err)
}

func TestConnection_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := sqlx.Config{
		Host:           testCfg.Host,
		Port:           testCfg.Port,
		User:           "test_user",
		Password:       "secret",
		Database:       "test_db",
		SSLMode:        "disable",
		ConnectTimeout: 5,
		QueryTimeout:   5 * time.Second,
	}

	conn, err := sqlx.Connect(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, conn)

	var result int
	err = conn.Get(context.Background(), &result, "SELECT 1")
	require.NoError(t, err)

	err = conn.Close()
	require.NoError(t, err)

	err = conn.Get(context.Background(), &result, "SELECT 1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func TestConnection_NamedQuery_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_named_query_close (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			active BOOLEAN NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_named_query_close (title, active) VALUES
		($1, $2), ($3, $4)
	`, "Test 1", true, "Test 2", false)
	require.NoError(t, err)

	type QueryParams struct {
		Active bool `db:"active"`
	}

	params := QueryParams{Active: true}
	rows, err := testDB.NamedQuery(ctx, `
		SELECT * FROM test_named_query_close WHERE active = :active
	`, params)
	require.NoError(t, err)
	require.NotNil(t, rows)

	var titles []string
	for rows.Next() {
		var id int
		var title string
		var active bool
		err := rows.Scan(&id, &title, &active)
		require.NoError(t, err)
		titles = append(titles, title)
	}
	require.NoError(t, rows.Err())
	require.Len(t, titles, 1)

	err = rows.Close()
	require.NoError(t, err)

	err = rows.Close()
	require.NoError(t, err)
}

func TestTx_Select(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_select (
			id SERIAL PRIMARY KEY,
			category TEXT NOT NULL,
			amount INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO test_tx_select (category, amount) VALUES
			($1, $2), ($3, $4), ($5, $6)
		`, "A", 100, "B", 200, "A", 150)
		if err != nil {
			return err
		}

		type Record struct {
			ID       int    `db:"id"`
			Category string `db:"category"`
			Amount   int    `db:"amount"`
		}

		var records []Record
		err = tx.Select(ctx, &records, "SELECT * FROM test_tx_select WHERE category = $1", "A")
		require.NoError(t, err)
		require.Len(t, records, 2)

		var total int
		for _, r := range records {
			total += r.Amount
		}
		require.Equal(t, 250, total)

		return nil
	})
	require.NoError(t, err)
}

func TestTx_Query(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_query (
			id SERIAL PRIMARY KEY,
			label TEXT NOT NULL,
			score INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_tx_query (label, score) VALUES
		($1, $2), ($3, $4), ($5, $6)
	`, "alpha", 10, "beta", 20, "gamma", 30)
	require.NoError(t, err)

	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
		var count int
		err := tx.Get(ctx, &count, "SELECT COUNT(*) FROM test_tx_query")
		if err != nil {
			return err
		}
		require.Equal(t, 3, count)
		return nil
	})
	require.NoError(t, err)
}

func TestTx_QueryRow(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_query_row (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_tx_query_row (username, email) VALUES
		($1, $2), ($3, $4)
	`, "user1", "user1@example.com", "user2", "user2@example.com")
	require.NoError(t, err)

	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
		row := tx.QueryRow(ctx, "SELECT username FROM test_tx_query_row WHERE username = $1", "user1")
		require.NotNil(t, row)

		var username string
		err := row.Scan(&username)
		require.NoError(t, err)
		require.Equal(t, "user1", username)

		return nil
	})
	require.NoError(t, err)
}

func TestConnection_BeginTx(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_begin_tx (
			id SERIAL PRIMARY KEY,
			value INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, tx)

	_, err = tx.Exec(ctx, "INSERT INTO test_begin_tx (value) VALUES ($1)", 100)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	var value int
	err = testDB.Get(ctx, &value, "SELECT value FROM test_begin_tx WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, 100, value)

	opts := &sqlx.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  true,
	}

	tx, err = testDB.BeginTx(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, tx)

	err = tx.Get(ctx, &value, "SELECT value FROM test_begin_tx WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, 100, value)

	err = tx.Rollback()
	require.NoError(t, err)
}

func TestTx_Commit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_commit (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, "INSERT INTO test_tx_commit (data) VALUES ($1)", "test data")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	var data string
	err = testDB.Get(ctx, &data, "SELECT data FROM test_tx_commit WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, "test data", data)

	err = tx.Commit()
	require.Error(t, err)
}

func TestTx_Rollback(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_rollback (
			id SERIAL PRIMARY KEY,
			info TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, "INSERT INTO test_tx_rollback (info) VALUES ($1)", "will be rolled back")
	require.NoError(t, err)

	err = tx.Rollback()
	require.NoError(t, err)

	var count int
	err = testDB.Get(ctx, &count, "SELECT COUNT(*) FROM test_tx_rollback")
	require.NoError(t, err)
	require.Equal(t, 0, count)

	err = tx.Rollback()
	require.NoError(t, err)
}

func TestConnection_Query_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Query(ctx, "SELECT * FROM nonexistent_table")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to execute query")

	row := testDB.QueryRow(ctx, "SELECT * FROM nonexistent_table")
	require.NotNil(t, row)

	var result any
	err = row.Scan(&result)
	require.Error(t, err)
}

func TestIsConstraintViolation(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_constraint_violations (
			id SERIAL PRIMARY KEY,
			code TEXT UNIQUE NOT NULL,
			value INTEGER CHECK (value >= 0)
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_violations (code, value) VALUES ($1, $2)", "UNIQUE_1", 10)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_violations (code, value) VALUES ($1, $2)", "UNIQUE_1", 20)
	require.Error(t, err)
	require.True(t, sqlx.IsConstraintViolation(err))
	require.True(t, sqlx.IsUniqueViolation(err))

	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_violations (code, value) VALUES ($1, $2)", "UNIQUE_2", -1)
	require.Error(t, err)
	require.True(t, sqlx.IsConstraintViolation(err))
	require.True(t, sqlx.IsCheckViolation(err))

	notPqErr := errors.New("some other error")
	require.False(t, sqlx.IsConstraintViolation(notPqErr))
	require.False(t, sqlx.IsUniqueViolation(notPqErr))
	require.False(t, sqlx.IsForeignKeyViolation(notPqErr))
	require.False(t, sqlx.IsCheckViolation(notPqErr))
	require.False(t, sqlx.IsNotNullViolation(notPqErr))
}

func TestGetConstraintName(t *testing.T) {
	notPqErr := errors.New("some error")
	name := sqlx.GetConstraintName(notPqErr)
	require.Empty(t, name)

	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_constraint_name (
			id SERIAL PRIMARY KEY,
			email TEXT UNIQUE
		)
	`)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_name (email) VALUES ($1)", "test@example.com")
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_name (email) VALUES ($1)", "test@example.com")
	require.Error(t, err)

	name = sqlx.GetConstraintName(err)
	require.NotEmpty(t, name)
}
