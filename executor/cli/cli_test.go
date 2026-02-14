package cli

// func TestNew(t *testing.T) {
// 	t.Parallel()

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)
// 	require.NotNil(t, executor)

// 	t.Cleanup(func() {
// 		executor.Close()
// 	})
// }

// func TestExecutor_Start(t *testing.T) {
// 	t.Parallel()

// 	tests := []struct {
// 		name        string
// 		command     string
// 		wantErr     bool
// 		errContains string
// 	}{
// 		{
// 			name:    "success",
// 			command: "echo",
// 			wantErr: false,
// 		},
// 		{
// 			name:        "command_not_found",
// 			command:     "nonexistent_command_12345",
// 			wantErr:     true,
// 			errContains: `"nonexistent_command_12345" not found`,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()

// 			cfg := Config{
// 				Command: tt.command,
// 			}

// 			executor := New(cfg, nil, nil)
// 			t.Cleanup(func() {
// 				executor.Close()
// 			})

// 			err := executor.Start()
// 			if tt.wantErr {
// 				require.Error(t, err)
// 				assert.Contains(t, err.Error(), tt.errContains)
// 			} else {
// 				require.NoError(t, err)
// 			}
// 		})
// 	}
// }

// func TestExecutor_Run(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)
// 	t.Cleanup(func() {
// 		executor.Close()
// 	})

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Тест с командой echo
// 	err := executor.Execute(ctx, "hello", "world")

// 	require.NoError(t, err)
// }

// func TestExecutor_Close(t *testing.T) {
// 	t.Parallel()

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)

// 	// Первое закрытие
// 	err := executor.Close()
// 	require.NoError(t, err)

// 	// Второе закрытие (должно вернуть ошибку)
// 	err = executor.Close()
// 	require.Error(t, err)
// 	assert.Contains(t, err.Error(), "executor is already closed")
// }

// func TestExecutor_Run_Closed(t *testing.T) {
// 	t.Parallel()

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)

// 	// Закрываем executor
// 	err := executor.Close()
// 	require.NoError(t, err)

// 	// Пытаемся выполнить команду
// 	ctx := context.Background()
// 	err = executor.Execute(ctx, "test")

// 	require.Error(t, err)
// 	assert.Contains(t, err.Error(), "executor is closed")
// }

// func TestExecutor_Run_WithError(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "sh",
// 	}

// 	executor := New(cfg, nil, nil)
// 	t.Cleanup(func() {
// 		executor.Close()
// 	})

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Тест с командой, которая завершается с ошибкой
// 	err := executor.Execute(ctx, "-c", "exit 1")

// 	require.Error(t, err)
// 	assert.Contains(t, err.Error(), "command failed")
// }

// func TestExecutor_Run_WithWriter(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	var stdout, stderr bytes.Buffer

// 	cfg := Config{
// 		Command: "sh",
// 	}

// 	executor := New(cfg, &stdout, &stderr)
// 	t.Cleanup(func() {
// 		executor.Close()
// 	})

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Тест с командой, которая пишет в stdout и stderr
// 	err := executor.Execute(ctx, "-c", "echo 'stdout message'; echo 'stderr message' >&2")

// 	require.NoError(t, err)
// 	assert.Contains(t, stdout.String(), "stdout message")
// 	assert.Contains(t, stderr.String(), "stderr message")
// }

// // TestExecutor_Run_WithTimeout проверяет обработку таймаута выполнения команды
// func TestExecutor_Run_WithTimeout(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "sleep",
// 	}

// 	executor := New(cfg, nil, nil)
// 	t.Cleanup(func() {
// 		executor.Close()
// 	})

// 	// Контекст с коротким таймаутом
// 	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
// 	defer cancel()

// 	// Команда sleep 5 должна быть прервана по таймауту
// 	err := executor.Execute(ctx, "5")

// 	require.Error(t, err)
// 	assert.Contains(t, err.Error(), "killed")
// }

// // TestExecutor_Run_WithCancelledContext проверяет обработку отмены контекста
// func TestExecutor_Run_WithCancelledContext(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "sleep",
// 	}

// 	executor := New(cfg, nil, nil)
// 	t.Cleanup(func() {
// 		executor.Close()
// 	})

// 	ctx, cancel := context.WithCancel(context.Background())

// 	// Запускаем команду в горутине
// 	errChan := make(chan error, 1)
// 	go func() {
// 		errChan <- executor.Execute(ctx, "10")
// 	}()

// 	// Отменяем контекст через 50мс
// 	time.Sleep(50 * time.Millisecond)
// 	cancel()

// 	// Получаем ошибку
// 	err := <-errChan
// 	require.Error(t, err)
// 	assert.Contains(t, err.Error(), "killed")
// }

// // TestExecutor_ConcurrentExecution проверяет конкурентное выполнение команд
// func TestExecutor_ConcurrentExecution(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)
// 	t.Cleanup(func() {
// 		executor.Close()
// 	})

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Запускаем несколько команд параллельно
// 	numGoroutines := 10
// 	results := make(chan error, numGoroutines)

// 	for i := 0; i < numGoroutines; i++ {
// 		go func() {
// 			results <- executor.Execute(ctx, "test")
// 		}()
// 	}

// 	// Проверяем, что все команды выполнились успешно
// 	for i := 0; i < numGoroutines; i++ {
// 		err := <-results
// 		require.NoError(t, err)
// 	}
// }

// // TestExecutor_ConcurrentExecutionWithDifferentArgs проверяет конкурентное выполнение с разными аргументами
// func TestExecutor_ConcurrentExecutionWithDifferentArgs(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)
// 	t.Cleanup(func() {
// 		executor.Close()
// 	})

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Запускаем несколько команд с разными аргументами
// 	numGoroutines := 5
// 	results := make(chan error, numGoroutines)

// 	for i := 0; i < numGoroutines; i++ {
// 		go func(idx int) {
// 			results <- executor.Execute(ctx, "test", "arg"+string(rune('0'+idx)))
// 		}(i)
// 	}

// 	// Проверяем, что все команды выполнились успешно
// 	for i := 0; i < numGoroutines; i++ {
// 		err := <-results
// 		require.NoError(t, err)
// 	}
// }

// // TestExecutor_RaceCondition проверяет отсутствие race condition при конкурентном доступе
// func TestExecutor_RaceCondition(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)
// 	t.Cleanup(func() {
// 		executor.Close()
// 	})

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Запускаем много горутин
// 	numGoroutines := 100
// 	done := make(chan struct{}, numGoroutines)

// 	for i := 0; i < numGoroutines; i++ {
// 		go func() {
// 			_ = executor.Execute(ctx, "test")
// 			done <- struct{}{}
// 		}()
// 	}

// 	// Ждём завершения всех горутин
// 	for i := 0; i < numGoroutines; i++ {
// 		<-done
// 	}
// }

// // TestExecutor_ConcurrentExecutionDifferentCommands проверяет конкурентное выполнение разных команд
// func TestExecutor_ConcurrentExecutionDifferentCommands(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg1 := Config{Command: "echo"}
// 	cfg2 := Config{Command: "printf"}

// 	exec1 := New(cfg1, nil, nil)
// 	defer exec1.Close()

// 	exec2 := New(cfg2, nil, nil)
// 	defer exec2.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	errChan := make(chan error, 2)

// 	go func() {
// 		errChan <- exec1.Execute(ctx, "hello")
// 	}()

// 	go func() {
// 		errChan <- exec2.Execute(ctx, "world")
// 	}()

// 	// Проверяем, что обе команды выполнились успешно
// 	for i := 0; i < 2; i++ {
// 		err := <-errChan
// 		require.NoError(t, err)
// 	}
// }

// // TestExecutor_LongRunningCommand проверяет выполнение длительной команды
// func TestExecutor_LongRunningCommand(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "sleep",
// 	}

// 	executor := New(cfg, nil, nil)
// 	defer executor.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Команда sleep 1 должна выполниться успешно
// 	start := time.Now()
// 	err := executor.Execute(ctx, "1")
// 	duration := time.Since(start)

// 	require.NoError(t, err)
// 	assert.GreaterOrEqual(t, duration, 1*time.Second)
// 	assert.Less(t, duration, 2*time.Second)
// }

// // TestExecutor_ExecuteWithMultipleArgs проверяет выполнение команды с множеством аргументов
// func TestExecutor_ExecuteWithMultipleArgs(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "printf",
// 	}

// 	executor := New(cfg, nil, nil)
// 	defer executor.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Команда с множеством аргументов
// 	err := executor.Execute(ctx, "%s", "%s", "%s", "hello", "world", "42")

// 	require.NoError(t, err)
// }

// // TestExecutor_ExecuteWithStderrWriter проверяет выполнение команды с кастомным Stderr
// func TestExecutor_ExecuteWithStderrWriter(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	var stderrBuf bytes.Buffer

// 	cfg := Config{
// 		Command: "sh",
// 	}

// 	executor := New(cfg, nil, &stderrBuf)
// 	defer executor.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Команда, которая пишет в stderr
// 	err := executor.Execute(ctx, "-c", "echo 'stderr message' >&2")

// 	require.NoError(t, err)
// 	assert.Contains(t, stderrBuf.String(), "stderr message")
// }

// // TestExecutor_ExecuteWithComplexCommand проверяет выполнение сложной команды через sh
// func TestExecutor_ExecuteWithComplexCommand(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "sh",
// 	}

// 	executor := New(cfg, nil, nil)
// 	defer executor.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Сложная команда с конвейером
// 	err := executor.Execute(ctx, "-c", "echo 'test' | grep test")

// 	require.NoError(t, err)
// }

// // TestExecutor_StartAndExecute проверяет последовательный вызов Start и Execute
// func TestExecutor_StartAndExecute(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)
// 	defer executor.Close()

// 	// Сначала проверяем наличие команды
// 	err := executor.Start()
// 	require.NoError(t, err)

// 	// Затем выполняем команду
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)
// 	err = executor.Execute(ctx, "test")
// 	require.NoError(t, err)
// }

// // TestExecutor_MultipleExecutors проверяет работу нескольких экземпляров executor
// func TestExecutor_MultipleExecutors(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg1 := Config{Command: "echo"}
// 	cfg2 := Config{Command: "printf"}

// 	exec1 := New(cfg1, nil, nil)
// 	exec2 := New(cfg2, nil, nil)
// 	defer exec1.Close()
// 	defer exec2.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Проверяем оба executor
// 	err1 := exec1.Start()
// 	err2 := exec2.Start()
// 	require.NoError(t, err1)
// 	require.NoError(t, err2)

// 	// Выполняем команды
// 	err1 = exec1.Execute(ctx, "from exec1")
// 	err2 = exec2.Execute(ctx, "from exec2")
// 	require.NoError(t, err1)
// 	require.NoError(t, err2)
// }

// // TestExecutor_ExecuteWithVeryLongArgs проверяет выполнение команды с очень длинными аргументами
// func TestExecutor_ExecuteWithVeryLongArgs(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)
// 	defer executor.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Создаём очень длинную строку
// 	longStr := ""
// 	for i := 0; i < 1000; i++ {
// 		longStr += "x"
// 	}

// 	err := executor.Execute(ctx, longStr)
// 	require.NoError(t, err)
// }

// // TestExecutor_CloseWhileExecuting проверяет закрытие executor во время выполнения
// func TestExecutor_CloseWhileExecuting(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "sleep",
// 	}

// 	executor := New(cfg, nil, nil)

// 	ctx := context.Background()

// 	// Запускаем длительную команду в горутине
// 	errChan := make(chan error, 1)
// 	go func() {
// 		errChan <- executor.Execute(ctx, "5")
// 	}()

// 	// Ждём немного и закрываем executor
// 	time.Sleep(100 * time.Millisecond)
// 	err := executor.Close()
// 	require.NoError(t, err)

// 	// Получаем результат выполнения (может быть ошибка контекста)
// 	<-errChan
// }

// // TestExecutor_ExecuteWithEmptyArgs проверяет выполнение команды без аргументов
// func TestExecutor_ExecuteWithEmptyArgs(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "echo",
// 	}

// 	executor := New(cfg, nil, nil)
// 	defer executor.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Выполняем команду без аргументов
// 	err := executor.Execute(ctx)
// 	require.NoError(t, err)
// }

// // TestExecutor_ExecuteWithSpecialChars проверяет выполнение команды с спецсимволами
// func TestExecutor_ExecuteWithSpecialChars(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test in short mode")
// 	}

// 	cfg := Config{
// 		Command: "sh",
// 	}

// 	executor := New(cfg, nil, nil)
// 	defer executor.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	t.Cleanup(cancel)

// 	// Команда с спецсимволами
// 	err := executor.Execute(ctx, "-c", "echo '$HOME && $PATH'")

// 	require.NoError(t, err)
// }
