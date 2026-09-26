#!/usr/bin/env bash
# Собирает закреплённые инструменты из backend/tools (директивы tool, AD-20)
# в кэш и печатает каталог с бинарниками. Выполняется внутри golang:1.27.1.
# Каталог зависит от хеша tools/go.mod, go.sum и исходников своих
# анализаторов — пересборка только при их изменении.
#   go-tools.sh [пакет-или-путь ...]   по умолчанию — линтеры для make check
set -euo pipefail
cd /src/backend/tools
TOOLBIN="${TOOLBIN:-/cache/bin}"
key=$(cat go.mod go.sum */*.go | sha256sum | cut -c1-16)
bin="$TOOLBIN/$key"
pkgs=("$@")
if [[ ${#pkgs[@]} == 0 ]]; then
  pkgs=(github.com/golangci/golangci-lint/v2/cmd/golangci-lint ./detcheck ./archgen)
fi
need=()
for p in "${pkgs[@]}"; do
  name=$(basename "$p")
  [[ $name =~ ^v[0-9]+$ ]] && name=$(basename "$(dirname "$p")")
  [[ -x "$bin/$name" ]] || need+=("$p")
done
if [[ ${#need[@]} -gt 0 ]]; then
  mkdir -p "$bin"
  echo "go-tools: сборка ${need[*]}" >&2
  go build -o "$bin/" "${need[@]}"
  # Старые наборы инструментов (другие версии) старше 3 дней — удалить.
  find "$TOOLBIN" -mindepth 1 -maxdepth 1 -type d ! -name "$key" -mtime +3 -exec rm -rf {} + 2>/dev/null || true
fi
echo "$bin"
