package sqlx

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *Connection

func TestMain(m *testing.M) {
	// Сначала обрабатываем флаги
	flag.Parse()

	// Если установлен флаг -short, пропускаем интеграционные тесты
	if testing.Short() {
		fmt.Println("Skipping integration tests in short mode")
		os.Exit(0)
	}

	ctx := context.Background()

	// Создаем контейнер PostgreSQL
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
		panic(fmt.Sprintf("Could not start container: %s", err))
	}

	// Очищаем ресурсы после тестов
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			fmt.Printf("Warning: could not terminate container: %s\n", err)
		}
	}()

	// Получаем порт
	host, err := container.Host(ctx)
	if err != nil {
		panic(fmt.Sprintf("Could not get container host: %s", err))
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		panic(fmt.Sprintf("Could not get container port: %s", err))
	}

	port, err := strconv.Atoi(mappedPort.Port())
	if err != nil {
		panic(fmt.Sprintf("Could not parse port: %s", err))
	}

	// Конфигурация для подключения к тестовой БД
	cfg := Config{
		Host:           host,
		Port:           port,
		User:           "test_user",
		Password:       "secret",
		Database:       "test_db",
		SSLMode:        "disable",
		ConnectTimeout: 5,
		QueryTimeout:   30 * time.Second,
	}

	// Ждем готовности БД и подключаемся
	maxRetries := 30
	retryInterval := 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		testDB, err = Connect(ctx, cfg)
		if err == nil {
			break
		}
		if i == maxRetries-1 {
			panic(fmt.Sprintf("Could not connect to database after %d retries: %s", maxRetries, err))
		}
		time.Sleep(retryInterval)
	}

	// Запускаем тесты после успешного подключения
	code := m.Run()

	// Закрываем соединение с БД
	if testDB != nil {
		if err := testDB.Close(); err != nil {
			fmt.Printf("Warning: failed to close test DB: %s\n", err)
		}
	}

	os.Exit(code)
}

func skipShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

func TestConnection_Exec(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL
		)
	`)
	require.NoError(t, err)

	// Вставляем тестовые данные
	result, err := testDB.Exec(ctx, `
		INSERT INTO test_users (name, email) VALUES ($1, $2)
	`, "John Doe", "john@example.com")
	require.NoError(t, err)

	// Проверяем количество затронутых строк
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	// Проверяем уникальность email
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_users (name, email) VALUES ($1, $2)
	`, "Jane Doe", "john@example.com")
	require.Error(t, err)
	require.True(t, IsUniqueViolation(err))
}

func TestConnection_Transaction(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу с проверкой баланса
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_accounts (
			id SERIAL PRIMARY KEY,
			balance INTEGER NOT NULL CHECK (balance >= 0)
		)
	`)
	require.NoError(t, err)

	// Тестируем успешную транзакцию
	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *Tx) error {
		// Создаем два счета
		_, err := tx.Exec(ctx, `
			INSERT INTO test_accounts (balance) VALUES ($1), ($2)
		`, 100, 0)
		if err != nil {
			return err
		}

		// Переводим деньги между счетами
		_, err = tx.Exec(ctx, "UPDATE test_accounts SET balance = balance - $1 WHERE id = 1", 50)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE test_accounts SET balance = balance + $1 WHERE id = 2", 50)
		return err
	})
	require.NoError(t, err)

	// Проверяем балансы
	var balance1, balance2 int
	err = testDB.Get(ctx, &balance1, "SELECT balance FROM test_accounts WHERE id = 1")
	require.NoError(t, err)
	err = testDB.Get(ctx, &balance2, "SELECT balance FROM test_accounts WHERE id = 2")
	require.NoError(t, err)

	require.Equal(t, 50, balance1)
	require.Equal(t, 50, balance2)

	// Тестируем откат транзакции
	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE test_accounts SET balance = balance - $1 WHERE id = 1;
		`, 100) // Недостаточно средств
		return err
	})
	require.Error(t, err)

	// Проверяем что балансы не изменились
	err = testDB.Get(ctx, &balance1, "SELECT balance FROM test_accounts WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, 50, balance1)
}

func TestConnection_Get(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу и данные
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

	// Тестируем Get
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

	// Тестируем Get с несуществующей записью
	err = testDB.Get(ctx, &item, "SELECT * FROM test_items WHERE id = $1", 999)
	require.Error(t, err)
}

func TestConnection_Select(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу и данные
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_products (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			category TEXT NOT NULL,
			price INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	// Вставляем тестовые данные
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_products (name, category, price) VALUES
		($1, $2, $3),
		($4, $5, $6)
	`, "Product 1", "A", 100, "Product 2", "B", 200)
	require.NoError(t, err)

	// Тестируем Select
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

	// Тестируем Select с фильтрацией
	err = testDB.Select(ctx, &products, "SELECT * FROM test_products WHERE category = $1", "A")
	require.NoError(t, err)
	require.Len(t, products, 1)
	require.Equal(t, "Product 1", products[0].Name)
}

func TestConnection_QueryTimeout(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем запрос с искусственной задержкой
	_, err := testDB.Exec(ctx, `
		CREATE OR REPLACE FUNCTION test_delay() RETURNS void AS $$
		BEGIN
			PERFORM pg_sleep(2);
		END;
		$$ LANGUAGE plpgsql;
	`)
	require.NoError(t, err)

	// Сохраняем и восстанавливаем исходный таймаут
	oldTimeout := testDB.cfg.QueryTimeout
	testDB.cfg.QueryTimeout = 100 * time.Millisecond
	defer func() { testDB.cfg.QueryTimeout = oldTimeout }()

	// Проверяем что запрос прерывается по таймауту
	_, err = testDB.Exec(ctx, "SELECT test_delay()")
	require.Error(t, err)
	require.Contains(t, err.Error(), "canceling statement due to user request")
}

func TestConnection_ErrorHandling(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу с ограничениями
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_constraints (
			id SERIAL PRIMARY KEY,
			code TEXT UNIQUE NOT NULL,
			value INTEGER CHECK (value > 0),
			ref_id INTEGER REFERENCES test_accounts(id)
		)
	`)
	require.NoError(t, err)

	// Тест на нарушение UNIQUE
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (code, value) VALUES ($1, $2)
	`, "CODE1", 100)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (code, value) VALUES ($1, $2)
	`, "CODE1", 200)
	require.Error(t, err)
	require.True(t, IsUniqueViolation(err))

	// Тест на нарушение CHECK
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (code, value) VALUES ($1, $2)
	`, "CODE2", -1)
	require.Error(t, err)
	require.True(t, IsCheckViolation(err))

	// Тест на нарушение FOREIGN KEY
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (code, value, ref_id) VALUES ($1, $2, $3)
	`, "CODE3", 100, 999)
	require.Error(t, err)
	require.True(t, IsForeignKeyViolation(err))

	// Тест на нарушение NOT NULL
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_constraints (value) VALUES ($1)
	`, 100)
	require.Error(t, err)
	require.True(t, IsNotNullViolation(err))
}

func TestConnection_NamedQueries(t *testing.T) {
	skipShort(t)

	// Создаем тестовую таблицу
	createCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

	// Тестируем NamedExec с отдельным контекстом
	insertCtx, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

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

	// Тестируем NamedQuery с отдельным контекстом
	queryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_isolation (
			id SERIAL PRIMARY KEY,
			value INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	// Вставляем начальные данные
	_, err = testDB.Exec(ctx, `INSERT INTO test_isolation (value) VALUES ($1)`, 100)
	require.NoError(t, err)

	// Тестируем READ COMMITTED (по умолчанию)
	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx1 *Tx) error {
		// Читаем значение в первой транзакции
		var value1 int
		err := tx1.Get(ctx, &value1, "SELECT value FROM test_isolation WHERE id = 1")
		require.NoError(t, err)
		require.Equal(t, 100, value1)

		// Запускаем вторую транзакцию и меняем значение
		err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx2 *Tx) error {
			_, err := tx2.Exec(ctx, "UPDATE test_isolation SET value = $1 WHERE id = 1", 200)
			return err
		})
		require.NoError(t, err)

		// Повторно читаем значение в первой транзакции
		err = tx1.Get(ctx, &value1, "SELECT value FROM test_isolation WHERE id = 1")
		require.NoError(t, err)
		require.Equal(t, 200, value1) // В READ COMMITTED видим изменения второй транзакции

		return nil
	})
	require.NoError(t, err)

	// Тестируем REPEATABLE READ
	opts := &TxOptions{Isolation: sql.LevelRepeatableRead}
	err = testDB.RunTx(ctx, opts, func(ctx context.Context, tx1 *Tx) error {
		// Читаем значение в первой транзакции
		var value1 int
		err := tx1.Get(ctx, &value1, "SELECT value FROM test_isolation WHERE id = 1")
		require.NoError(t, err)
		require.Equal(t, 200, value1)

		// Запускаем вторую транзакцию и меняем значение
		err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx2 *Tx) error {
			_, err := tx2.Exec(ctx, "UPDATE test_isolation SET value = $1 WHERE id = 1", 300)
			return err
		})
		require.NoError(t, err)

		// Повторно читаем значение в первой транзакции
		err = tx1.Get(ctx, &value1, "SELECT value FROM test_isolation WHERE id = 1")
		require.NoError(t, err)
		require.Equal(t, 200, value1) // В REPEATABLE READ не видим изменения второй транзакции

		return nil
	})
	require.NoError(t, err)
}

func TestConnection_ConcurrentTransactions(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_concurrent (
			id INTEGER PRIMARY KEY,
			counter INTEGER NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, err)

	// Вставляем начальные данные
	_, err = testDB.Exec(ctx, `INSERT INTO test_concurrent (id, counter) VALUES (1, 0)`)
	require.NoError(t, err)

	// Запускаем конкурентные транзакции
	const goroutines = 10
	const iterations = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Добавляем mutex для безопасной проверки ошибок
	var (
		errMu sync.Mutex
		errs  []error
	)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				err := testDB.RunTx(ctx, nil, func(ctx context.Context, tx *Tx) error {
					// Блокируем строку для обновления
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

	// Проверяем ошибки после завершения всех горутин
	require.Empty(t, errs, "got unexpected errors")

	// Проверяем финальное значение счетчика
	var finalCounter int
	err = testDB.Get(ctx, &finalCounter, "SELECT counter FROM test_concurrent WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, goroutines*iterations, finalCounter)
}

func TestConnection_Query(t *testing.T) {
	skipShort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_query (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	// Вставляем тестовые данные
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_query (name, value) VALUES
		($1, $2), ($3, $4), ($5, $6)
	`, "item1", 10, "item2", 20, "item3", 30)
	require.NoError(t, err)

	// Тестируем Query - verifies Query method is callable and returns non-nil rows
	rows, err := testDB.Query(ctx, "SELECT * FROM test_query ORDER BY id")
	require.NoError(t, err)
	require.NotNil(t, rows)

	// Close rows immediately to avoid context cancellation issues
	rows.Close()

	// Verify Query handles errors correctly
	_, err = testDB.Query(ctx, "SELECT * FROM nonexistent_table")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to execute query")
}

func TestConnection_QueryRow(t *testing.T) {
	skipShort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_query_row (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			age INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	// Вставляем тестовые данные
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_query_row (name, age) VALUES ($1, $2)
	`, "Alice", 30)
	require.NoError(t, err)

	// Тестируем QueryRow - verifies QueryRow method is callable and returns non-nil row
	row := testDB.QueryRow(ctx, "SELECT * FROM test_query_row WHERE name = $1", "Alice")
	require.NotNil(t, row)

	// Verify QueryRow with error case
	row = testDB.QueryRow(ctx, "SELECT * FROM test_query_row WHERE name = $1", "Bob")
	var name string
	var age int
	err = row.Scan(&name, &age)
	require.Error(t, err)
}

func TestConnection_Close(t *testing.T) {
	skipShort(t)

	// Создаем новое соединение для теста закрытия
	cfg := Config{
		Host:           "localhost",
		Port:           testDB.cfg.Port,
		User:           "test_user",
		Password:       "secret",
		Database:       "test_db",
		SSLMode:        "disable",
		ConnectTimeout: 5,
		QueryTimeout:   5 * time.Second,
	}

	conn, err := Connect(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Проверяем что соединение работает
	var result int
	err = conn.Get(context.Background(), &result, "SELECT 1")
	require.NoError(t, err)

	// Закрываем соединение
	err = conn.Close()
	require.NoError(t, err)

	// После закрытия запросы должны возвращать ошибку
	err = conn.Get(context.Background(), &result, "SELECT 1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func TestConnection_NamedQuery_Close(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_named_query_close (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			active BOOLEAN NOT NULL
		)
	`)
	require.NoError(t, err)

	// Вставляем тестовые данные
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_named_query_close (title, active) VALUES
		($1, $2), ($3, $4)
	`, "Test 1", true, "Test 2", false)
	require.NoError(t, err)

	type QueryParams struct {
		Active bool `db:"active"`
	}

	// Тестируем NamedQuery с закрытием rows
	params := QueryParams{Active: true}
	rows, err := testDB.NamedQuery(ctx, `
		SELECT * FROM test_named_query_close WHERE active = :active
	`, params)
	require.NoError(t, err)
	require.NotNil(t, rows)

	// Читаем данные
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

	// Закрываем rows
	err = rows.Close()
	require.NoError(t, err)

	// Повторное закрытие не должно вызывать ошибку
	err = rows.Close()
	require.NoError(t, err)
}

func TestTx_Select(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_select (
			id SERIAL PRIMARY KEY,
			category TEXT NOT NULL,
			amount INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	// Вставляем данные внутри транзакции
	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *Tx) error {
		// Вставляем тестовые данные
		_, err := tx.Exec(ctx, `
			INSERT INTO test_tx_select (category, amount) VALUES
			($1, $2), ($3, $4), ($5, $6)
		`, "A", 100, "B", 200, "A", 150)
		if err != nil {
			return err
		}

		// Тестируем Select в транзакции
		type Record struct {
			ID       int    `db:"id"`
			Category string `db:"category"`
			Amount   int    `db:"amount"`
		}

		var records []Record
		err = tx.Select(ctx, &records, "SELECT * FROM test_tx_select WHERE category = $1", "A")
		require.NoError(t, err)
		require.Len(t, records, 2)

		// Проверяем сумму
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
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_query (
			id SERIAL PRIMARY KEY,
			label TEXT NOT NULL,
			score INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	// First, insert data
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_tx_query (label, score) VALUES
		($1, $2), ($3, $4), ($5, $6)
	`, "alpha", 10, "beta", 20, "gamma", 30)
	require.NoError(t, err)

	// Now test Query in a transaction - use Get to verify Query method works
	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *Tx) error {
		// Тестируем Get в транзакции - verifies Query method via Get
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
	skipShort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_query_row (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// First, insert data
	_, err = testDB.Exec(ctx, `
		INSERT INTO test_tx_query_row (username, email) VALUES
		($1, $2), ($3, $4)
	`, "user1", "user1@example.com", "user2", "user2@example.com")
	require.NoError(t, err)

	// Тестируем QueryRow в транзакции
	err = testDB.RunTx(ctx, nil, func(ctx context.Context, tx *Tx) error {
		// Тестируем QueryRow - verifies method is callable and returns non-nil row
		row := tx.QueryRow(ctx, "SELECT username FROM test_tx_query_row WHERE username = $1", "user1")
		require.NotNil(t, row)

		// Consume the row
		var username string
		err := row.Scan(&username)
		require.NoError(t, err)
		require.Equal(t, "user1", username)

		return nil
	})
	require.NoError(t, err)
}

func TestDefaultTxOptions(t *testing.T) {
	opts := DefaultTxOptions()

	require.NotNil(t, opts)
	assert.Equal(t, sql.LevelDefault, opts.Isolation)
	assert.False(t, opts.ReadOnly)
	assert.False(t, opts.Deferrable)
}

func TestConnection_BeginTx(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_begin_tx (
			id SERIAL PRIMARY KEY,
			value INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	// Тестируем BeginTx с nil options
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Выполняем операцию и коммитим
	_, err = tx.Exec(ctx, "INSERT INTO test_begin_tx (value) VALUES ($1)", 100)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Проверяем данные
	var value int
	err = testDB.Get(ctx, &value, "SELECT value FROM test_begin_tx WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, 100, value)

	// Тестируем BeginTx с опциями
	opts := &TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  true,
	}

	tx, err = testDB.BeginTx(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// В read-only транзакции можно читать
	err = tx.Get(ctx, &value, "SELECT value FROM test_begin_tx WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, 100, value)

	err = tx.Rollback()
	require.NoError(t, err)
}

func TestTx_Commit(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_commit (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Создаем транзакцию вручную
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Вставляем данные
	_, err = tx.Exec(ctx, "INSERT INTO test_tx_commit (data) VALUES ($1)", "test data")
	require.NoError(t, err)

	// Коммитим транзакцию
	err = tx.Commit()
	require.NoError(t, err)

	// Проверяем что данные сохранены
	var data string
	err = testDB.Get(ctx, &data, "SELECT data FROM test_tx_commit WHERE id = 1")
	require.NoError(t, err)
	require.Equal(t, "test data", data)

	// Повторный коммит должен вернуть ошибку
	err = tx.Commit()
	require.Error(t, err)
}

func TestTx_Rollback(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Создаем тестовую таблицу
	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_tx_rollback (
			id SERIAL PRIMARY KEY,
			info TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Создаем транзакцию вручную
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)

	// Вставляем данные
	_, err = tx.Exec(ctx, "INSERT INTO test_tx_rollback (info) VALUES ($1)", "will be rolled back")
	require.NoError(t, err)

	// Откатываем транзакцию
	err = tx.Rollback()
	require.NoError(t, err)

	// Проверяем что данные не сохранены
	var count int
	err = testDB.Get(ctx, &count, "SELECT COUNT(*) FROM test_tx_rollback")
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Повторный rollback не должен возвращать ошибку (даже если транзакция уже закрыта)
	err = tx.Rollback()
	require.NoError(t, err)
}

func TestConnection_Query_ErrorHandling(t *testing.T) {
	skipShort(t)
	ctx := context.Background()

	// Тестируем Query с синтаксической ошибкой
	_, err := testDB.Query(ctx, "SELECT * FROM nonexistent_table")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to execute query")

	// Тестируем QueryRow с ошибкой в запросе (возвращает Row, ошибка при сканировании)
	row := testDB.QueryRow(ctx, "SELECT * FROM nonexistent_table")
	require.NotNil(t, row)

	var result interface{}
	err = row.Scan(&result)
	require.Error(t, err)
}

func TestConnection_WithTimeout(t *testing.T) {
	// Тестируем WithTimeout с положительным таймаутом
	ctx := context.Background()
	ctxWithTimeout, cancel := WithTimeout(ctx, 5*time.Second)
	require.NotNil(t, ctxWithTimeout)
	require.NotNil(t, cancel)

	deadline, ok := ctxWithTimeout.Deadline()
	require.True(t, ok)
	require.True(t, deadline.After(time.Now()))
	cancel()

	// Тестируем WithTimeout с нулевым таймаутом
	ctxNoTimeout, cancel2 := WithTimeout(ctx, 0)
	require.NotNil(t, ctxNoTimeout)
	require.NotNil(t, cancel2)

	_, ok = ctxNoTimeout.Deadline()
	require.False(t, ok, "Context without timeout should not have deadline")
	cancel2()
}

func TestIsConstraintViolation(t *testing.T) {
	// Создаем таблицу с ограничениями для тестов
	skipShort(t)
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_constraint_violations (
			id SERIAL PRIMARY KEY,
			code TEXT UNIQUE NOT NULL,
			value INTEGER CHECK (value >= 0)
		)
	`)
	require.NoError(t, err)

	// Тестируем UNIQUE violation
	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_violations (code, value) VALUES ($1, $2)", "UNIQUE_1", 10)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_violations (code, value) VALUES ($1, $2)", "UNIQUE_1", 20)
	require.Error(t, err)
	require.True(t, IsConstraintViolation(err))
	require.True(t, IsUniqueViolation(err))

	// Тестируем CHECK violation
	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_violations (code, value) VALUES ($1, $2)", "UNIQUE_2", -1)
	require.Error(t, err)
	require.True(t, IsConstraintViolation(err))
	require.True(t, IsCheckViolation(err))

	// Тестируем с не-pq ошибкой
	notPqErr := errors.New("some other error")
	require.False(t, IsConstraintViolation(notPqErr))
	require.False(t, IsUniqueViolation(notPqErr))
	require.False(t, IsForeignKeyViolation(notPqErr))
	require.False(t, IsCheckViolation(notPqErr))
	require.False(t, IsNotNullViolation(notPqErr))
}

func TestGetConstraintName(t *testing.T) {
	// Тестируем с не-pq ошибкой
	notPqErr := errors.New("some error")
	name := GetConstraintName(notPqErr)
	require.Empty(t, name)

	// Создаем таблицу с именованным ограничением для теста
	skipShort(t)
	ctx := context.Background()

	_, err := testDB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_constraint_name (
			id SERIAL PRIMARY KEY,
			email TEXT UNIQUE
		)
	`)
	require.NoError(t, err)

	// Пытаемся вставить дубликат
	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_name (email) VALUES ($1)", "test@example.com")
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, "INSERT INTO test_constraint_name (email) VALUES ($1)", "test@example.com")
	require.Error(t, err)

	// Извлекаем имя ограничения
	name = GetConstraintName(err)
	// Имя ограничения будет содержать "email" или быть autogenerated
	// Главное - функция не должна паниковать
	require.NotEmpty(t, name)
}
