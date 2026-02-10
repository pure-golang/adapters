package executor

import (
	"context"
	"io"
)

// Provider определяет интерфейс для запуска и остановки провайдера
type Provider interface {
	Start() error
	io.Closer
}

// Executor определяет интерфейс для выполнения внешних команд
type Executor interface {
	// Execute выполняет команду с заданным контекстом и аргументами
	Execute(ctx context.Context, args ...string) ([]byte, error)
	Provider
}
