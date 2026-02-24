// Package storage определяет интерфейсы и ошибки для объектного хранилища.
//
// Пакет предоставляет базовые типы ошибок для S3-совместимых хранилищ.
// Реализации находятся в дочерних пакетах:
//   - [storage/minio] — MinIO/S3 адаптер
//
// Типы ошибок:
//   - [ErrNotFound] — объект не найден
//   - [ErrAccessDenied] — доступ запрещён
//   - [ErrBucketNotFound] — bucket не существует
//   - [StorageError] — детальная ошибка с кодом и контекстом
//
// Хелперы для проверки ошибок:
//   - [IsNotFound] — проверка ErrNotFound
//   - [IsAccessDenied] — проверка ErrAccessDenied
//   - [IsBucketNotFound] — проверка ErrBucketNotFound
//
// Использование:
//
//	_, err := storage.Get(ctx, bucket, key)
//	if storage.IsNotFound(err) {
//	    // Обработка случая "не найдено"
//	}
//
// Коды ошибок:
//   - [CodeNotFound] — объект не найден
//   - [CodeAccessDenied] — доступ запрещён
//   - [CodeBucketNotFound] — bucket не существует
//   - [CodeInternalError] — внутренняя ошибка
package storage
