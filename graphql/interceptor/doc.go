// Package interceptor реализует расширение ObservabilityExtension для gqlgen.
//
// Расширение добавляет трейсинг, метрики и логирование на уровне GraphQL-операций.
// Работает поверх HTTP middleware Monitoring: обогащает уже существующий HTTP span
// атрибутами операции и создаёт дочерний span для самой операции.
//
// Использование:
//
//	srv := handler.New(schema)
//	srv.Use(interceptor.NewObservabilityExtension(func(ctx context.Context) (int, bool) {
//	    id, err := platformjwt.GetUserID(ctx)
//	    return id, err == nil
//	}))
//
// Метрики:
//
//	graphql.request_count — счётчик операций (с метками has_errors, operation_type, operation_name)
//	graphql.request_time  — гистограмма времени выполнения операции (ms)
//
// Ограничения:
//
//   - Требует наличия graphql.OperationContext в контексте (устанавливается gqlgen автоматически).
//   - Если HTTP middleware Monitoring не используется, HTTP span в контексте будет noop.
//   - Потокобезопасность: да.
package interceptor
