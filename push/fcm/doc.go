// Пакет fcm реализует отправку push-уведомлений через Firebase Cloud Messaging (FCM).
//
// Поддерживает унифицированную отправку на Android, iOS и Web — платформа
// определяется автоматически по формату токена.
//
// Использование:
//
//	pusher, err := fcm.NewPusher(ctx, fcm.Config{
//	    CredentialsFile: "/path/to/service-account.json",
//	})
//	if err != nil {
//	    return err
//	}
//	defer pusher.Close()
//
//	err = pusher.Push(ctx, fcm.Notification{
//	    Token: deviceToken,
//	    Title: "Hello",
//	    Body:  "World",
//	})
//
// Конфигурация (задаётся вручную через Config, env-теги не используются):
//
//	CredentialsFile — путь к JSON-файлу сервисного аккаунта GCP
//	CredentialsJSON — содержимое JSON-файла сервисного аккаунта (альтернатива CredentialsFile)
//	ProjectID       — идентификатор GCP-проекта (опционально, выводится из учётных данных)
//
// Ограничения:
//
//   - Требуется либо CredentialsFile, либо CredentialsJSON — иначе NewPusher вернёт ошибку
//   - Thread-safe: да
//   - Close() не освобождает ресурсы (FCM SDK не требует явного закрытия)
//   - Для тестов используйте NoopPusher или NewPusherWithFactory с mock-фабрикой
package fcm
