package cli

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/codes"

	"github.com/pure-golang/adapters/executor"
)

var _ executor.Executor = (*Executor)(nil)

// Executor реализует интерфейс executor.Executor для CLI утилит
type Executor struct {
	logger *slog.Logger
	cmd    string
	stdout io.Writer
	stderr io.Writer
	closed bool
	mx     sync.Mutex
}

// New создаёт новый CLI executor
func New(cfg Config, stdout, stderr io.Writer) *Executor {
	var tmpStdout io.Writer
	if stdout == nil {
		tmpStdout = os.Stdout
	} else {
		tmpStdout = stdout
	}
	var tmpStderr io.Writer
	if stderr == nil {
		tmpStderr = os.Stderr
	} else {
		tmpStderr = stderr
	}
	return &Executor{
		logger: slog.Default().WithGroup("executor/cli"),
		cmd:    cfg.Command,
		stdout: tmpStdout,
		stderr: tmpStderr,
		closed: false,
	}
}

// Start проверяет наличие команды в системе
func (e *Executor) Start() error {
	e.logger.Info("checking command availability", "command", e.cmd)

	if _, err := exec.LookPath(e.cmd); err != nil {
		e.logger.Error("command not found", "command", e.cmd, "error", err)
		return errors.Wrapf(err, "command %q not found", e.cmd)
	}

	e.logger.Info("command found", "command", e.cmd)
	return nil
}

// Execute выполняет команду с заданным контекстом и аргументами
func (e *Executor) Execute(ctx context.Context, args ...string) error {
	// 1. КРИТИЧЕСКАЯ СЕКЦИЯ (только проверка и регистрация)
	e.mx.Lock()
	if e.closed {
		e.mx.Unlock()
		e.logger.Error("attempting to execute command on closed executor", "command", e.cmd)
		return errors.New("executor is closed")
	}
	// Создание команды
	//nolint:gosec // G204: Subprocess launched with a potential tainted input or cmd arguments - e.cmd is validated in Start(), args are user-controlled but not shell-expanded
	cmd := exec.CommandContext(ctx, e.cmd, args...)
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr
	e.mx.Unlock()

	// 2. ВЫПОЛНЕНИЕ (параллельно, без блокировки мьютекса)
	e.logger.Info("executing command", "command", e.cmd, "args", args)

	_, span := tracer.Start(ctx, "executor.Execute")
	defer span.End()

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime).Seconds()

	if err != nil {
		e.logger.Error("command execution error", "command", e.cmd, "args", args, "error", err, "duration_seconds", duration)
		recordError(span, err)
		recordExecution(e.cmd, "error", duration)
		return errors.Wrapf(err, "command failed: %q", e.cmd)
	}

	e.logger.Info("command executed successfully", "command", e.cmd, "args", args, "duration_seconds", duration)
	span.SetStatus(codes.Ok, "")
	recordExecution(e.cmd, "success", duration)
	return nil
}

// Close закрывает executor
func (e *Executor) Close() error {
	e.mx.Lock()
	if e.closed {
		e.mx.Unlock()
		return errors.New("executor is already closed")
	}
	e.closed = true
	e.mx.Unlock()

	return nil
}
