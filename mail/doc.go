// Package mail определяет интерфейс [Sender] для отправки email.
//
// Пакет предоставляет базовые типы и интерфейс для email-клиентов.
// Реализации находятся в дочерних пакетах:
//   - [mail/smtp] — SMTP клиент для отправки писем
//   - [mail/noop] — заглушка для тестирования
//
// Использование:
//
//	var sender mail.Sender = smtp.NewSender(cfg)
//	err := sender.Send(ctx, mail.Email{
//	    From:    mail.Address{Address: "noreply@example.com"},
//	    To:      []mail.Address{{Address: "user@example.com"}},
//	    Subject: "Welcome",
//	    Body:    "Hello!",
//	})
//	defer sender.Close()
//
// Интерфейсы:
//   - [Sender] — отправка email сообщений
//
// Типы:
//   - [Email] — структура email сообщения
//   - [Address] — email адрес с опциональным именем
package mail
