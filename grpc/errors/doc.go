// Package errors предоставляет утилиты для обработки gRPC ошибок.
//
// Пакет содержит функции для преобразования стандартных ошибок Go
// в gRPC статусы и обратно.
//
// Использование:
//
//	import grpcerrors "github.com/pure-golang/adapters/grpc/errors"
//
//	// Преобразование ошибки в gRPC статус
//	err := grpcerrors.FromError(ctx.Err())
//
//	// Создание ошибки с кодом
//	err := grpcerrors.NewError(codes.NotFound, "resource not found")
//
//	// Обёртка ошибки с кодом
//	err := grpcerrors.WrapError(err, codes.Internal, "failed to process")
//
// Функции:
//   - [FromError] — преобразует error в gRPC статус
//   - [WrapError] — оборачивает ошибку с gRPC кодом
//   - [NewError] — создаёт новую ошибку с gRPC кодом
//
// Маппинг стандартных ошибок:
//   - context.Canceled → codes.Canceled
//   - context.DeadlineExceeded → codes.DeadlineExceeded
//   - прочие → codes.Internal
package errors
