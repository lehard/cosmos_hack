#!/usr/bin/env bash
# Проверки контрактов ant (эпик 00; вызывается из `make check`). Запуск — из корня репозитория:
#   contracts/scripts/check.sh
# Что проверяется (AD-20, AD-40, AD-17, AD-46):
#   диалект JSON Schema и компиляция всех схем; каталог типов (схема и ровно один эмитент у каждого типа);
#   AsyncAPI 3.0 собран из каталога и валиден; коды ошибок; таблица уровней доверия анализатора;
#   тест-векторы crypto; контракты собственных процессов; примеры пяти случаев изменения контракта и v1 → v2;
#   затравка нормативного слоя; BPMN фланца по дескриптору urn:ant:bpmn-ext:1 (step_key у каждого узла);
#   openapi.yaml: x-ant-action, классы, эмитенты, соответствие политике (эпик 02) и самопроверка.
# Нужен Node ≥ 20 на хосте; иначе проверка идёт в контейнере node:24.21.0-slim (docker через `sg docker -c`).
# Блокировку ~/.cache/cosmo-build.lock скрипт НЕ берёт (её держит make check).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
SCRIPTS=contracts/scripts
CHECKS=(lint-schemas check-catalog check-errors check-tables asyncapi check-vectors check-internal check-examples check-normative check-bpmn check-bpmn-js check-openapi)

node_ok() {
  command -v node >/dev/null 2>&1 || return 1
  local major; major="$(node -p 'process.versions.node.split(".")[0]' 2>/dev/null || echo 0)"
  [ "${major}" -ge 20 ]
}

run_checks() {
  # Зависимости проверок закреплены в package-lock.json; ставятся один раз в contracts/scripts/node_modules.
  if [ ! -d "$SCRIPTS/node_modules" ] || [ "$SCRIPTS/package-lock.json" -nt "$SCRIPTS/node_modules/.package-lock.json" ]; then
    (cd "$SCRIPTS" && timeout 600 npm ci --no-audit --no-fund --loglevel=error)
  fi
  local failed=0
  for c in "${CHECKS[@]}"; do
    timeout 300 node "$SCRIPTS/$c.mjs" || failed=1
  done
  # Самопроверка проверки операций (эпик 02): «нет класса» и «чужой тип» краснеют.
  timeout 300 node "$SCRIPTS/check-openapi.mjs" --selftest || failed=1
  return "$failed"
}

if [ "${1:-}" = "--in-container" ] || node_ok; then
  run_checks
  echo "contracts: все проверки пройдены"
  exit 0
fi

# Нет подходящего Node — проверка в контейнере (без блокировки сборки).
DOCKER="docker"
if ! docker info >/dev/null 2>&1; then DOCKER="sg docker -c docker"; fi
mkdir -p "${HOME}/.cache/ant/npm"
CMD="docker run --rm -u $(id -u):$(id -g) -e HOME=/tmp -e npm_config_cache=/npm -v ${ROOT}:/repo -v ${HOME}/.cache/ant/npm:/npm -w /repo node:24.21.0-slim bash contracts/scripts/check.sh --in-container"
if [ "$DOCKER" = "docker" ]; then
  exec timeout 900 $CMD
else
  exec timeout 900 sg docker -c "$CMD"
fi
