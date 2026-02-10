package cli

import (
	"context"
	"os/exec"
	"sync"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/codes"
)

// Executor реализует интерфейс executor.Executor для CLI утилит
type Executor struct {
	cmd    string
	closed bool
	mx     sync.RWMutex
}

// New создаёт новый CLI executor
func New(cfg Config) *Executor {
	return &Executor{
		cmd:    cfg.Command,
		closed: false,
	}
}

// Start проверяет наличие команды в системе
func (e *Executor) Start() error {
	if _, err := exec.LookPath(e.cmd); err != nil {
		return errors.Wrapf(err, "command %s not found", e.cmd)
	}
	return nil
}

// Execute выполняет команду с заданным контекстом и аргументами
func (e *Executor) Execute(ctx context.Context, args ...string) ([]byte, error) {
	stdout, stderr, err := e.executeWithOutput(ctx, args)
	if err != nil {
		return stdout, errors.Wrapf(err, "command failed: %s", string(stderr))
	}
	return stdout, nil
}

// executeWithOutput выполняет команду и возвращает stdout и stderr
func (e *Executor) executeWithOutput(ctx context.Context, args []string) ([]byte, []byte, error) {
	ctx, span := tracer.Start(ctx, "executor.Execute")
	defer span.End()

	e.mx.RLock()
	defer e.mx.RUnlock()

	if e.closed {
		return nil, nil, errors.New("executor is closed")
	}

	// Создание команды
	cmd := exec.CommandContext(ctx, e.cmd, args...)

	// Получение stdout и stderr
	stdout, err := cmd.Output()
	if err != nil {
		var stderr []byte
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = exitErr.Stderr
		}
		recordError(span, err)
		return stdout, stderr, err
	}

	span.SetStatus(codes.Ok, "")
	return stdout, nil, nil
}

// Close закрывает executor
func (e *Executor) Close() error {
	e.mx.Lock()
	defer e.mx.Unlock()

	if e.closed {
		return nil
	}
	e.closed = true
	return nil
}
