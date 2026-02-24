// Package env предоставляет утилиты для загрузки конфигурации из переменных окружения.
//
// Пакет объединяет github.com/joho/godotenv для загрузки .env файлов
// и github.com/kelseyhightower/envconfig для парсинга переменных в структуры.
//
// Использование:
//
//	import "github.com/pure-golang/adapters/env"
//
//	type Config struct {
//	    Host string `envconfig:"HOST" required:"true"`
//	    Port int    `envconfig:"PORT" default:"8080"`
//	}
//
//	var cfg Config
//	if err := env.InitConfig(&cfg); err != nil {
//	    log.Fatal(err)
//	}
//
// Особенности:
//   - Автоматически загружает .env файл из текущей директории
//   - Не падает если .env файл отсутствует
//   - Поддерживает теги envconfig для маппинга переменных
//   - Поддерживает значения по умолчанию через тег default
//   - Поддерживает обязательные поля через тег required
//
// Теги конфигурации:
//
//	`envconfig:"VAR_NAME"`     — имя переменной окружения
//	`default:"value"`          — значение по умолчанию
//	`required:"true"`          — обязательное поле
//	`ignore:"true"`            — игнорировать поле
package env
