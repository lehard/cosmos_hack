#!/usr/bin/env bash
# Обёртка над docker для Makefile.
#  - Если текущий процесс не входит в группу docker, команда выполняется через
#    `sg docker -c …` (так работают агенты на машине сборки).
#  - С флагом --lock команда берёт блокировку ANT_BUILD_LOCK (flock): тяжёлые
#    сборки, pull и compose up на машине с ~2 ГБ свободной памяти идут по одной.
#    Пустой ANT_BUILD_LOCK — без блокировки.
set -euo pipefail

lock=0
if [[ "${1-}" == "--lock" ]]; then
  lock=1
  shift
fi

if docker info >/dev/null 2>&1; then
  cmd=(docker "$@")
elif command -v sg >/dev/null 2>&1; then
  cmd=(sg docker -c "$(printf '%q ' docker "$@")")
else
  echo "docker недоступен: нет прав на сокет и нет sg" >&2
  exit 1
fi

if [[ $lock == 1 && -n "${ANT_BUILD_LOCK-}" ]] && command -v flock >/dev/null 2>&1; then
  mkdir -p "$(dirname "$ANT_BUILD_LOCK")"
  exec flock "$ANT_BUILD_LOCK" "${cmd[@]}"
fi
exec "${cmd[@]}"
