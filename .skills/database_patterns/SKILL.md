---
name: "database_patterns"
description: "Паттерны работы с PostgreSQL: подключение, транзакции, named queries, выбор драйвера"
---
# Database Patterns

## Connection (sqlx)
```go
cfg := sqlx.Config{
    Host:           "localhost",
    Port:           5432,
    User:           "postgres",
    Password:       "secret",
    Database:       "mydb",
    SSLMode:        "disable",
    ConnectTimeout: 5,
    QueryTimeout:   10 * time.Second,
}

db, err := sqlx.Connect(context.Background(), cfg)
defer db.Close()
```

## Transactions
```go
err := db.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
    _, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", 100, 1)
    if err != nil {
        return err  // Auto rollback on error
    }
    _, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", 100, 2)
    return err  // Commit on nil, rollback on error
})
```

## Transaction Isolation Levels
```go
opts := &sqlx.TxOptions{
    Isolation: sql.LevelRepeatableRead,
    ReadOnly:  false,
}
err := db.RunTx(ctx, opts, func(ctx context.Context, tx *sqlx.Tx) error {
    // operations
    return nil
})
```

## Named Queries (sqlx)
```go
type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
    Age  int    `db:"age"`
}

user := User{Name: "John", Age: 30}
result, err := db.NamedExec(ctx,
    "INSERT INTO users (name, age) VALUES (:name, :age)",
    user)
```

## Constraint Violation Helpers
```go
// PostgreSQL constraint checks
IsUniqueViolation(err)
IsForeignKeyViolation(err)
IsCheckViolation(err)
IsNotNullViolation(err)
IsConstraintViolation(err)
```

## Driver Selection
| Adapter | Driver | Use when |
|---------|--------|----------|
| `db/pg/sqlx` | `lib/pq` | Traditional projects, simple queries |
| `db/pg/pgx` | `jackc/pgx/v5` | New projects, connection pooling, high performance |

**pgx is recommended for new projects.**

## Query Timeout
- Applied via context wrapping (`WithTimeout()`)
- Default timeout configured in `Config.QueryTimeout`
- SQL queries include timeout automatically through the wrapper

## Transaction Rollback Notes
- `RunTx()` automatically rolls back on error or panic
- Manual `Rollback()` must check for `sql.ErrTxDone` (already committed/rolled back)
- Transactions use defer-based rollback pattern internally
