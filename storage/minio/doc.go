// Package minio реализует [storage.Storage] для объектного хранилища MinIO / S3.
//
// Поддерживает:
//   - загрузку и скачивание объектов
//   - мультичастную загрузку
//   - presigned URL для временного доступа
//   - OpenTelemetry tracing
//
// Использование:
//
//	import "github.com/pure-golang/adapters/storage/minio"
//
//	client, err := minio.Connect(ctx, minio.ConfigFromEnv())
//	err = client.Upload(ctx, bucket, key, reader, size)
//
// Конфигурация через переменные окружения:
//
//	MINIO_ENDPOINT   — адрес сервера (default: localhost:9000)
//	MINIO_ACCESS_KEY — access key
//	MINIO_SECRET_KEY — secret key
//	MINIO_USE_SSL    — использовать TLS (default: false)
//	MINIO_BUCKET     — bucket по умолчанию
package minio
