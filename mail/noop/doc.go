// Package noop реализует [mail.Sender] как заглушку для тестирования.
//
// Использование:
//
//	var sender mail.Sender = noop.NewSender()
//	err := sender.Send(ctx, email) // молча игнорирует отправку
//	defer sender.Close()
//
// Особенности:
//   - Send() всегда возвращает nil
//   - Close() всегда возвращает nil
//   - Не отправляет реальные письма
//   - Используется в unit-тестах
package noop
