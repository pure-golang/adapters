package executor

import (
	"context"
	"io"
)

// Executor определяет интерфейс для выполнения внешних команд
type Executor interface {
	// Execute выполняет команду с заданным контекстом и аргументами
	Execute(ctx context.Context, args ...string) ([]byte, error)
	io.Closer
}
