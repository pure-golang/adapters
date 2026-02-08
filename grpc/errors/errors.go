package errors

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FromError преобразует ошибки в gRPC-статусы
func FromError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}

	// Если ошибка уже является gRPC-статусом, возвращаем её как есть
	if _, ok := status.FromError(err); ok {
		return err
	}

	// По умолчанию используем Internal с исходным сообщением
	return status.Error(codes.Internal, err.Error())
}

// WrapError оборачивает ошибку с заданным кодом статуса и сообщением
func WrapError(err error, code codes.Code, msg string) error {
	if err == nil {
		return nil
	}
	return status.Errorf(code, "%s: %v", msg, err)
}

// NewError создает новую ошибку с заданным кодом и сообщением
func NewError(code codes.Code, msg string) error {
	return status.Error(code, msg)
}
