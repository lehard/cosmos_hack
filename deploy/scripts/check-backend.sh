#!/usr/bin/env bash
# Проверки бэкенда для make check (AD-1, AD-4, AD-20; NFR-ARCH-1, NFR-DEV-3,
# NFR-DET-1). Выполняется внутри golang:1.27.1 из Makefile.
set -euo pipefail
cd /src/backend
bin=$(/src/deploy/scripts/go-tools.sh)

step() { printf '\n== %s\n' "$*"; }

step "правила слоёв актуальны (layers.json → .golangci.yml)"
(cd tools && "$bin/archgen" -layers archgen/layers.json -out ../.golangci.yml -check)

step "vendor согласован с go.mod"
go list -mod=vendor ./... >/dev/null

step "gofmt"
unformatted=$(gofmt -l $(find . -name '*.go' -not -path './vendor/*' -not -path './tools/*') tools || true)
if [[ -n "$unformatted" ]]; then
  echo "не отформатированы (gofmt -w):"; echo "$unformatted"; exit 1
fi

step "go vet"
go vet ./...
(cd tools && go vet ./...)

step "golangci-lint: depguard (направление зависимостей), forbidigo (часы и случайность в домене)"
"$bin/golangci-lint" run --max-issues-per-linter=0 --max-same-issues=0 ./...

step "detcheck: float и обход map в домене (AD-4)"
if [[ -n "$(find internal/domain -name '*.go' 2>/dev/null | head -1)" ]]; then
  go vet -vettool="$bin/detcheck" ./internal/domain/...
else
  echo "пакетов домена пока нет"
fi

step "emitcheck: модуль эмитит только свои типы записей (AD-40)"
go vet -vettool="$bin/emitcheck" ./internal/domain/... ./internal/application/...

step "go test"
# ANT_CHECK_FAST=1 — быстрый прогон при слиянии пачек (Д-18): тесты из кэша, без самопроверки линтеров.
if [[ "${ANT_CHECK_FAST-}" == 1 ]]; then go test ./...; else go test -count=1 ./...; fi

step "go build"
CGO_ENABLED=0 go build -o /dev/null ./cmd/...

if [[ "${ANT_CHECK_FAST-}" == 1 ]]; then
  echo "самопроверка линтеров пропущена (быстрый прогон)"
else
  /src/deploy/scripts/arch-selftest.sh "$bin"
fi
