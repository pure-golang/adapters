#!/bin/bash
set -e  # выход при любой ошибке
set -o pipefail  # выход при ошибке в любой команде пайплайна

mkdir -p .coverage

# не отключать COVERAGE_EXCLUDE на этом этапе
GOEXPERIMENT=synctest go test -covermode=atomic -coverprofile=.coverage/.out -coverpkg=./... ./...

COVERAGE_EXCLUDE="(/mocks/|_easyjson\.go|/graph/|/pb/)"
grep -vE "$COVERAGE_EXCLUDE" .coverage/.out > .coverage/.txt
rm .coverage/.out
go tool cover -html=.coverage/.txt -o=.coverage/.html
COVERAGE=$(go tool cover -func=.coverage/.txt | tail -1 | awk '{print $NF}')
echo ""
echo "📊 Общее покрытие кода: $COVERAGE"
echo ""
echo "🎯 Для применения в VSCode:"
echo "1. Нажмите Ctrl+Shift+P (Cmd+Shift+P на Mac)"
echo "2. Введите 'Go: Apply Cover Profile'"
echo "3. Укажите путь: $(pwd)/.coverage/.txt"
echo ""
