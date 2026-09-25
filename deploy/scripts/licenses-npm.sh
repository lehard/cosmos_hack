#!/usr/bin/env bash
# Перечень npm-зависимостей интерфейса (только попадающие в сборку) с лицензиями
# (make licenses; PRD §6.3, NFR-SEC-2). Выполняется внутри node:24.21.0-slim.
set -euo pipefail
cd /src/frontend
[[ -d node_modules ]] || npm ci --prefer-offline --no-audit --no-fund
npx --no-install license-checker-rseidelsohn --production --csv --excludePrivatePackages \
  --out /src/docs/licenses/npm.csv >/dev/null
echo "npm: $(($(wc -l < /src/docs/licenses/npm.csv) - 1)) пакетов → docs/licenses/npm.csv"
