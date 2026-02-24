// Package smtp реализует отправку email через SMTP.
//
// Поддерживает:
//   - plaintext SMTP
//   - STARTTLS
//   - TLS
//   - OpenTelemetry tracing
//
// Использование:
//
//	import "github.com/pure-golang/adapters/mail/smtp"
//
//	sender, err := smtp.NewSender(smtp.Config{
//	    Host:     "smtp.example.com",
//	    Port:     587,
//	    Username: "user@example.com",
//	    Password: "secret",
//	})
//	err = sender.Send(ctx, mail.Message{...})
//
// Конфигурация через переменные окружения:
//
//	SMTP_HOST     — хост SMTP-сервера
//	SMTP_PORT     — порт (default: 25)
//	SMTP_USERNAME — имя пользователя
//	SMTP_PASSWORD — пароль
//	SMTP_FROM     — адрес отправителя
package smtp
