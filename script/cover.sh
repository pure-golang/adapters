#!/bin/bash
set -e  # выход при любой ошибке
set -o pipefail  # выход при ошибке в любой команде пайплайна

mkdir -p .coverage
go test -covermode=atomic -coverprofile=.coverage/.out -coverpkg=./... ./...
COVER_EXCLUDE="(mocks|_easyjson\.go)"
grep -vE "$COVER_EXCLUDE" .coverage/.out > .coverage/.tmp
rm .coverage/.out
go tool cover -func=.coverage/.tmp -o=.coverage/.txt
go tool cover -html=.coverage/.tmp -o=.coverage/.html
