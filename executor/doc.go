// Package executor определяет интерфейсы для выполнения внешних команд.
//
// Пакет предоставляет абстракцию для запуска внешних CLI утилит.
// Реализации находятся в дочерних пакетах:
//   - [executor/cli] — выполнение локальных и SSH команд
//
// Использование:
//
//	var exec executor.Executor = cli.New(cfg, stdout, stderr)
//	err := exec.Start()
//	err = exec.Execute(ctx, "-i", "input.mp4", "output.mp4")
//	defer exec.Close()
//
// Интерфейсы:
//   - [Provider] — запуск и остановка исполнителя
//   - [Executor] — выполнение команд с контекстом
package executor
