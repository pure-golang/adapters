# PostgreSQL с PGX

Пакет представляет собой перемещенный код из db/pg для работы с PostgreSQL через библиотеку pgx.

## Миграция

Для перемещения кода следуйте инструкциям:

1. Создайте директорию `db/pg/pgx`
2. Скопируйте все файлы из `db/pg` в `db/pg/pgx`
3. Обновите названия пакетов с `package pg` на `package pgx`
4. Обновите импорты в проекте, заменив 
   ```go
   import "github.com/pure-golang/adapters/db/pg"
   ```
   на 
   ```go
   import "github.com/pure-golang/adapters/db/pg/pgx"
   ```

## Использование

Использование остается тем же, что и ранее с `db/pg`, просто импорты изменились. 