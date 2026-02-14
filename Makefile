# Установка зависимостей проекта
.PHONY: all
all:
	go install github.com/go-task/task/v3/cmd/task@latest
	go install github.com/vektra/mockery/v2@v2.53.5
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	task mod
	task
