// Package cli реализует [executor.Executor] для выполнения внешних CLI команд.
//
// Поддерживает:
//   - выполнение локальных команд
//   - выполнение команд через SSH (с паролем или ключом)
//   - OpenTelemetry tracing
//   - Prometheus метрики
//
// Использование (локальная команда):
//
//	cfg := cli.Config{Command: "ffmpeg"}
//	exec := cli.New(cfg, os.Stdout, os.Stderr)
//	if err := exec.Start(); err != nil {
//	    log.Fatal(err)
//	}
//	err := exec.Execute(ctx, "-i", "input.mp4", "-c:v", "libx264", "output.mp4")
//	defer exec.Close()
//
// Использование (SSH):
//
//	cfg := cli.Config{
//	    Command: "docker",
//	    SSH: cli.SSHConfig{
//	        Host:     "remote.example.com",
//	        User:     "deploy",
//	        KeyPath:  "/home/user/.ssh/id_rsa",
//	    },
//	}
//	exec := cli.New(cfg, os.Stdout, os.Stderr)
//
// Конфигурация через переменные окружения:
//
//	EXECUTOR_COMMAND — путь к исполняемому файлу
//	EXECUTOR_SSH_HOST — хост для SSH подключения
//	EXECUTOR_SSH_USER — пользователь SSH
//	EXECUTOR_SSH_PORT — порт SSH (default: 22)
//	EXECUTOR_SSH_KEY_PATH — путь к SSH ключу
//	EXECUTOR_SSH_PASSWORD — пароль SSH (требует sshpass)
//
// Особенности:
//   - Start() проверяет наличие команды в системе
//   - SSH с паролем требует установленный sshpass
//   - Потокобезопасный Close()
//   - Автоматическое логирование выполнения
package cli
