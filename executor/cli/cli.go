package cli

import (
	"context"
	"fmt"
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
	ssh    SSHConfig
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
		logger: slog.Default().With("executor/cli", cfg.Command),
		cmd:    cfg.Command,
		stdout: tmpStdout,
		stderr: tmpStderr,
		closed: false,
		ssh:    cfg.SSH,
	}
}

// Start проверяет наличие команды в системе
func (e *Executor) Start() error {
	if e.ssh.Host != "" {
		e.logger.Info("checking ssh availability", "host", e.ssh.Host)

		if _, err := exec.LookPath("ssh"); err != nil {
			e.logger.Error("ssh not found", "error", err)
			return errors.Wrap(err, "ssh not found")
		}

		if e.ssh.Password != "" {
			if _, err := exec.LookPath("sshpass"); err != nil {
				e.logger.Error("sshpass not found (required for password auth)", "error", err)
				return errors.Wrap(err, "sshpass not found (required for password authentication)")
			}
		}

		e.logger.Info("ssh found", "host", e.ssh.Host)
		return nil
	}

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
	// КРИТИЧЕСКАЯ СЕКЦИЯ (только проверка и регистрация)
	e.mx.Lock()
	if e.closed {
		e.mx.Unlock()
		e.logger.Error("attempting to execute command on closed executor", "command", e.cmd)
		return errors.New("executor is closed")
	}
	// Создание команды
	var tmpCmd string
	var tmpArgs []string
	if e.ssh.Host != "" {
		tmpCmd, tmpArgs = e.buildSSHCommand(args...)
	} else {
		tmpCmd, tmpArgs = e.cmd, args
	}
	execCmd := exec.CommandContext(ctx, tmpCmd, tmpArgs...)
	execCmd.Stdout = e.stdout
	execCmd.Stderr = e.stderr
	e.mx.Unlock()

	e.logger.Info("executing command", "command", e.cmd, "args", args)

	_, span := tracer.Start(ctx, "executor.Execute")
	defer span.End()

	startTime := time.Now()
	err := execCmd.Run()
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

// buildSSHCommand формирует команду и аргументы для выполнения через SSH
func (e *Executor) buildSSHCommand(args ...string) (string, []string) {
	var sshArgs []string

	baseCmd := "ssh"

	if e.ssh.Password != "" {
		baseCmd = "sshpass"
		sshArgs = append(sshArgs, "-p", e.ssh.Password, "ssh")
	}

	if e.ssh.User != "" {
		sshArgs = append(sshArgs, "-l", e.ssh.User)
	}

	if e.ssh.KeyPath != "" {
		sshArgs = append(sshArgs, "-i", e.ssh.KeyPath)
	}

	if e.ssh.Port != 0 && e.ssh.Port != 22 {
		sshArgs = append(sshArgs, "-p", fmt.Sprintf("%d", e.ssh.Port))
	}

	sshArgs = append(sshArgs, e.ssh.Host, e.cmd)
	sshArgs = append(sshArgs, args...)

	return baseCmd, sshArgs
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
