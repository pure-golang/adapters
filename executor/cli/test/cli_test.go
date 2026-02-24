package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/executor/cli"
)

func TestExecutor_Run(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "echo",
	}

	executor := cli.New(cfg, nil, nil)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err := executor.Execute(ctx, "hello", "world")

	require.NoError(t, err)
}

func TestExecutor_Run_WithError(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "sh",
	}

	executor := cli.New(cfg, nil, nil)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err := executor.Execute(ctx, "-c", "exit 1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "command failed")
}

func TestExecutor_Run_WithWriter(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	var stdout, stderr bytes.Buffer

	cfg := cli.Config{
		Command: "sh",
	}

	executor := cli.New(cfg, &stdout, &stderr)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err := executor.Execute(ctx, "-c", "echo 'stdout message'; echo 'stderr message' >&2")

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "stdout message")
	assert.Contains(t, stderr.String(), "stderr message")
}

func TestExecutor_Run_WithTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "sleep",
	}

	executor := cli.New(cfg, nil, nil)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := executor.Execute(ctx, "5")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "killed")
}

func TestExecutor_Run_WithCancelledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "sleep",
	}

	executor := cli.New(cfg, nil, nil)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error, 1)
	go func() {
		errChan <- executor.Execute(ctx, "10")
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-errChan
	require.Error(t, err)
	assert.Contains(t, err.Error(), "killed")
}

func TestExecutor_ConcurrentExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "echo",
	}

	executor := cli.New(cfg, nil, nil)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	numGoroutines := 10
	results := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			results <- executor.Execute(ctx, "test")
		}()
	}

	for range numGoroutines {
		err := <-results
		require.NoError(t, err)
	}
}

func TestExecutor_ConcurrentExecutionWithDifferentArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "echo",
	}

	executor := cli.New(cfg, nil, nil)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	numGoroutines := 5
	results := make(chan error, numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			results <- executor.Execute(ctx, "test", "arg"+string(rune('0'+idx)))
		}(i)
	}

	for range numGoroutines {
		err := <-results
		require.NoError(t, err)
	}
}

func TestExecutor_RaceCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "echo",
	}

	executor := cli.New(cfg, nil, nil)
	t.Cleanup(func() {
		executor.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	numGoroutines := 100
	done := make(chan struct{}, numGoroutines)

	for range numGoroutines {
		go func() {
			_ = executor.Execute(ctx, "test")
			done <- struct{}{}
		}()
	}

	for range numGoroutines {
		<-done
	}
}

func TestExecutor_ConcurrentExecutionDifferentCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg1 := cli.Config{Command: "echo"}
	cfg2 := cli.Config{Command: "printf"}

	exec1 := cli.New(cfg1, nil, nil)
	defer exec1.Close()

	exec2 := cli.New(cfg2, nil, nil)
	defer exec2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	errChan := make(chan error, 2)

	go func() {
		errChan <- exec1.Execute(ctx, "hello")
	}()

	go func() {
		errChan <- exec2.Execute(ctx, "world")
	}()

	for range 2 {
		err := <-errChan
		require.NoError(t, err)
	}
}

func TestExecutor_LongRunningCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "sleep",
	}

	executor := cli.New(cfg, nil, nil)
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	start := time.Now()
	err := executor.Execute(ctx, "1")
	duration := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 1*time.Second)
	assert.Less(t, duration, 2*time.Second)
}

func TestExecutor_ExecuteWithMultipleArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "printf",
	}

	executor := cli.New(cfg, nil, nil)
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err := executor.Execute(ctx, "%s", "%s", "%s", "hello", "world", "42")

	require.NoError(t, err)
}

func TestExecutor_ExecuteWithStderrWriter(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	var stderrBuf bytes.Buffer

	cfg := cli.Config{
		Command: "sh",
	}

	executor := cli.New(cfg, nil, &stderrBuf)
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err := executor.Execute(ctx, "-c", "echo 'stderr message' >&2")

	require.NoError(t, err)
	assert.Contains(t, stderrBuf.String(), "stderr message")
}

func TestExecutor_ExecuteWithComplexCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "sh",
	}

	executor := cli.New(cfg, nil, nil)
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err := executor.Execute(ctx, "-c", "echo 'test' | grep test")

	require.NoError(t, err)
}

func TestExecutor_StartAndExecute(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "echo",
	}

	executor := cli.New(cfg, nil, nil)
	defer executor.Close()

	err := executor.Start()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	err = executor.Execute(ctx, "test")
	require.NoError(t, err)
}

func TestExecutor_MultipleExecutors(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg1 := cli.Config{Command: "echo"}
	cfg2 := cli.Config{Command: "printf"}

	exec1 := cli.New(cfg1, nil, nil)
	exec2 := cli.New(cfg2, nil, nil)
	defer exec1.Close()
	defer exec2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err1 := exec1.Start()
	err2 := exec2.Start()
	require.NoError(t, err1)
	require.NoError(t, err2)

	err1 = exec1.Execute(ctx, "from exec1")
	err2 = exec2.Execute(ctx, "from exec2")
	require.NoError(t, err1)
	require.NoError(t, err2)
}

func TestExecutor_ExecuteWithVeryLongArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "echo",
	}

	executor := cli.New(cfg, nil, nil)
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	var longStr strings.Builder
	for range 1000 {
		longStr.WriteString("x")
	}

	err := executor.Execute(ctx, longStr.String())
	require.NoError(t, err)
}

func TestExecutor_CloseWhileExecuting(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "sleep",
	}

	executor := cli.New(cfg, nil, nil)

	ctx := context.Background()

	errChan := make(chan error, 1)
	go func() {
		errChan <- executor.Execute(ctx, "5")
	}()

	time.Sleep(100 * time.Millisecond)
	err := executor.Close()
	require.NoError(t, err)

	<-errChan
}

func TestExecutor_ExecuteWithEmptyArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "echo",
	}

	executor := cli.New(cfg, nil, nil)
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err := executor.Execute(ctx)
	require.NoError(t, err)
}

func TestExecutor_ExecuteWithSpecialChars(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := cli.Config{
		Command: "sh",
	}

	executor := cli.New(cfg, nil, nil)
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	err := executor.Execute(ctx, "-c", "echo '$HOME && $PATH'")

	require.NoError(t, err)
}
