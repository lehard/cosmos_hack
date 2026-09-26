#!/usr/bin/env bash
# make load — нагрузочный прогон и две проверки «1 = N» (эпик 35, FR-107, AD-6, SM-3).
#
# Два раздельных прогона одного сценария с одним seed на чистой БД, профиль load
# (64 партиции), свой проект compose ant-load и свои порты (не мешает демо-стенду):
#   1) один обработчик: ant (api, стадия, проектор, …) + ant-worker ×1;
#   2) N обработчиков:  ant + ant-worker ×N; посреди прогона одна копия воркера
#      останавливается и через OUTAGE секунд поднимается снова — недоступность
#      и восстановление: партиции уходят живым копиям по истечении аренды и
#      возвращаются после подъёма (OUTAGE=0 — без этого).
# После каждого прогона роль load (в том же образе) ждёт догонки журнала и
# пишет отчёт: задержки, потери, повторы, полнота, rebuild_hash на 1 и N
# обработчиках над тем же журналом и state_hash итогового состояния.
# Проверки: rebuild_hash_1 = rebuild_hash_n в каждом прогоне; state_hash и
# табло прогона 1 = прогона 2. Отчёты — scenarios/load/out/.
#
# Параметры: N (воркеров, 4), SCENARIO (MS-1), SEED (из определения),
# OUTAGE (20 с), ANT_HTTP_PORT (8484), ANT_STANDS_PORT (8494), ANT_IMAGE (ant:local),
# KEEP=1 — не останавливать стенд после второго прогона.
set -euo pipefail
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
N=${N:-4}
SCENARIO=${SCENARIO:-MS-1}
SEED=${SEED:-0}
OUTAGE=${OUTAGE:-20}
OUT="$ROOT/scenarios/load/out"
export ANT_HTTP_PORT=${ANT_HTTP_PORT:-8484} ANT_STANDS_PORT=${ANT_STANDS_PORT:-8494}
export ANT_LOAD_SCENARIO=$SCENARIO ANT_LOAD_SEED=$SEED ANT_LOAD_WORKERS=$N
DOCKER="$ROOT/deploy/scripts/docker.sh"
compose() {
  "$DOCKER" compose -p ant-load -f "$ROOT/deploy/compose/compose.yaml" -f "$ROOT/scenarios/load/compose.load.yaml" "$@"
}
mkdir -p "$OUT"

# run ‹метка› ‹воркеров› ‹сбой: 0|1› — один прогон на чистой БД, отчёт → $OUT/‹метка›.json
run() {
  local label=$1 workers=$2 outage=$3
  echo "== прогон $label: $SCENARIO, воркеров $workers"
  compose down -v --remove-orphans >/dev/null 2>&1 || true
  "$DOCKER" --lock compose -p ant-load -f "$ROOT/deploy/compose/compose.yaml" -f "$ROOT/scenarios/load/compose.load.yaml" \
    up -d --wait --scale ant-worker="$workers"
  local started=$SECONDS
  if [[ $outage == 1 && $OUTAGE -gt 0 ]]; then
    # Недоступность и восстановление: копия воркера №1 падает посреди прогона.
    (
      sleep 30
      echo "-- сбой: останавливаю ant-worker-1 на ${OUTAGE} с"
      "$DOCKER" stop -t 1 ant-load-ant-worker-1 >/dev/null
      sleep "$OUTAGE"
      "$DOCKER" start ant-load-ant-worker-1 >/dev/null
      echo "-- восстановление: ant-worker-1 снова в работе"
    ) &
  fi
  ANT_LOAD_LABEL="$label" compose run --rm -T load | tee "$OUT/$label.log" | grep -v '^LOAD-REPORT ' || true
  wait
  grep '^LOAD-REPORT ' "$OUT/$label.log" | sed 's/^LOAD-REPORT //' > "$OUT/$label.json"
  test -s "$OUT/$label.json" || { echo "прогон $label: отчёта нет (см. $OUT/$label.log)"; exit 1; }
  echo "   прогон $label: $((SECONDS - started)) с"
}

field() { sed -n "s/.*\"$2\": *\"\\{0,1\\}\\([^\",}]*\\).*/\\1/p" "$OUT/$1.json" | head -1; }

run workers-1 1 0
run "workers-$N" "$N" 1
if [[ ${KEEP:-0} != 1 ]]; then
  compose down -v --remove-orphans >/dev/null
fi

ok=1
for l in workers-1 "workers-$N"; do
  if [[ $(field "$l" rebuild_equal) != true ]]; then
    echo "НЕ СОВПАЛ rebuild_hash на 1 и $N обработчиках в прогоне $l"; ok=0
  fi
done
for f in state_hash board_hash board_actual_hash; do
  a=$(field workers-1 "$f"); b=$(field "workers-$N" "$f")
  printf '%-18s 1: %s\n%-18s %s: %s\n' "$f" "$a" "" "$N" "$b"
  [[ -n $a && $a == "$b" ]] || { echo "НЕ СОВПАЛ $f раздельных прогонов"; ok=0; }
done
echo "отчёты: $OUT/workers-1.json $OUT/workers-$N.json"
[[ $ok == 1 ]] && echo "make load: хеши совпадают на 1 и $N обработчиках" || exit 1
