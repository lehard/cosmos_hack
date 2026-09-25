#!/usr/bin/env bash
# Перечень Go-зависимостей с лицензиями (make licenses; PRD §6.3, NFR-SEC-2).
# Выполняется внутри golang:1.27.1.
set -euo pipefail
bin=$(/src/deploy/scripts/go-tools.sh github.com/google/go-licenses/v2)
cd /src/backend
out=/src/docs/licenses/go.csv
{
  echo "package,license_url,license"
  GOFLAGS="-mod=mod -buildvcs=false" "$bin/go-licenses" report ./... 2>/dev/null | sort
} > "$out"
echo "go: $(($(wc -l < "$out") - 1)) пакетов → docs/licenses/go.csv"
