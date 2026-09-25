#!/usr/bin/env bash
# Перечень Go-зависимостей с лицензиями (make licenses; PRD §6.3, NFR-SEC-2).
# Выполняется внутри golang:1.27.1.
set -euo pipefail
bin=$(/src/deploy/scripts/go-tools.sh github.com/google/go-licenses/v2)
cd /src/backend
out=/src/docs/licenses/go.csv
echo "package,license_url,license" > "$out.tmp"
{
  GOFLAGS="-mod=mod -buildvcs=false" "$bin/go-licenses" report --ignore ant ./... 2>/dev/null
  # Заимствованный код исходниками (third_party): go-licenses видит его, только
  # когда пакет импортирован продуктом, поэтому строка — всегда.
  echo "go.stargrave.org/gogost/v7,third_party/gogost/COPYING,GPL-3.0"
} | sort -u >> "$out.tmp"
mv "$out.tmp" "$out"
echo "go: $(($(wc -l < "$out") - 1)) пакетов → docs/licenses/go.csv"
