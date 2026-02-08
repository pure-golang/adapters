# PostgreSQL адаптер на базе sqlx

Адаптер для работы с PostgreSQL, построенный на библиотеке [sqlx](https://github.com/jmoiron/sqlx).

![version](https://img.shields.io/badge/version-v0.2.0-blue.svg)

## История изменений

### v0.2.0
- Добавлены интеграционные тесты с dockertest
- Улучшена обработка транзакций и их изоляции
- Добавлена поддержка именованных запросов
- Улучшен трейсинг через OpenTelemetry
- Добавлена поддержка конкурентных операций

### v0.1.0
- Базовая функциональность для работы с PostgreSQL
- Поддержка транзакций
- Обработка ошибок
- Таймауты запросов

## Возможности

- Подключение к PostgreSQL с настраиваемыми параметрами
- Выполнение запросов с контекстом и таймаутами
- Поддержка транзакций с разными уровнями изоляции
- Маппинг результатов на структуры Go
- Именованные запросы с параметрами
- Трейсинг запросов через OpenTelemetry
- Обработка ошибок PostgreSQL

## Использование

### Подключение

```go
cfg := sqlx.Config{
    Host:           "localhost",
    Port:           5432,
    User:           "postgres",
    Password:       "secret",
    Database:       "mydb",
    SSLMode:        "disable",
    ConnectTimeout: 5,
    QueryTimeout:   5 * time.Second,
}

db, err := sqlx.Connect(context.Background(), cfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### Запросы

```go
// Простой запрос
result, err := db.Exec(ctx, "INSERT INTO users(name) VALUES($1)", "John")

// Получение одной записи
var user User
err := db.Get(ctx, &user, "SELECT * FROM users WHERE id = $1", 1)

// Получение нескольких записей
var users []User
err := db.Select(ctx, &users, "SELECT * FROM users WHERE age > $1", 18)
```

### Транзакции

```go
err := db.RunTx(ctx, nil, func(ctx context.Context, tx *sqlx.Tx) error {
    // Операции в транзакции
    _, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", 100, 1)
    if err != nil {
        return err // Автоматический rollback
    }
    
    _, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", 100, 2)
    return err // Commit при nil, иначе rollback
})
```

### Именованные запросы

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

## Тестирование

Для запуска всех тестов:
```bash
go test .
```

Для пропуска интеграционных тестов:
```bash
go test -short .
```

## Примечания

- Все операции поддерживают context.Context для отмены и таймаутов
- Транзакции автоматически откатываются при ошибке
- Поддерживается конкурентное выполнение запросов
- Все SQL-запросы трейсятся через OpenTelemetry