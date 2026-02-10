package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Command: "echo",
	}

	executor := New(cfg)
	require.NotNil(t, executor)

	t.Cleanup(func() {
		executor.Close()
	})
}

func TestExecutor_Run(t *testing.T) {
	skipShort(t)
	t.Parallel()

	cfg := Config{
		Command: "echo",
	}

	executor := New(cfg)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx := context.Background()

	// Тест с командой echo
	output, err := executor.Execute(ctx, "hello", "world")

	require.NoError(t, err)
	assert.Contains(t, string(output), "hello world")
}

func TestExecutor_Close(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Command: "echo",
	}

	executor := New(cfg)

	// Первое закрытие
	err := executor.Close()
	require.NoError(t, err)

	// Второе закрытие (должно быть без ошибки)
	err = executor.Close()
	require.NoError(t, err)
}

func TestExecutor_Run_Closed(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Command: "echo",
	}

	executor := New(cfg)

	// Закрываем executor
	err := executor.Close()
	require.NoError(t, err)

	// Пытаемся выполнить команду
	ctx := context.Background()
	_, err = executor.Execute(ctx, "test")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "executor is closed")
}

func TestExecutor_Run_WithError(t *testing.T) {
	skipShort(t)
	t.Parallel()

	cfg := Config{
		Command: "sh",
	}

	executor := New(cfg)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx := context.Background()

	// Тест с командой, которая завершается с ошибкой и пишет в stderr
	// sh -c "echo 'error message' >&2; exit 1" - выводит в stderr и завершается с кодом 1
	output, err := executor.Execute(ctx, "-c", "echo 'error message' >&2; exit 1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error message")
	assert.Empty(t, output)
}
