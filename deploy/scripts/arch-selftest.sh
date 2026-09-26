#!/usr/bin/env bash
# Самопроверка линтеров: в копии backend создаются пробные нарушения, и каждое
# должно быть поймано. Если правило перестало срабатывать — make check красный.
# Выполняется внутри golang:1.27.1: arch-selftest.sh <каталог-инструментов>.
set -euo pipefail
bin="$1"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cp -a /src/backend "$work/backend"
mkdir -p "$work/contracts/events" && cp /src/contracts/events/catalog.yaml "$work/contracts/events/"
cd "$work/backend"
rm -rf tools

printf '\n== самопроверка линтеров: пробные нарушения должны краснеть\n'

put() { # put <путь> <содержимое>
  mkdir -p "$(dirname "$1")"
  printf '%s\n' "$2" > "$1"
}

# Пробные пакеты-листья zzprobe (их нет в дереве): настоящие модули уже связаны
# импортами, и пробный импорт в них дал бы цикл вместо нарушения правила.
for p in internal/infrastructure/storage/zzprobe internal/infrastructure/transport/zzprobe; do
  put "$p/zz_selftest_target.go" "package zzprobe

// SelftestTarget — мишень для пробных импортов.
const SelftestTarget = 1"
done
put internal/domain/zzlate/zz_selftest_target.go 'package zzlate

// SelftestTarget — мишень для пробных импортов.
const SelftestTarget = 1'

# 1. Домен импортирует инфраструктуру (AD-1).
put internal/domain/zzprobe/zz_selftest_infra.go 'package zzprobe

import storage "ant/internal/infrastructure/storage/zzprobe"

var _ = storage.SelftestTarget'

# 2. Часы в домене (AD-4).
put internal/domain/zzprobe/zz_selftest_clock.go 'package zzprobe

import "time"

func selftestClock() time.Time { return time.Now() }'

# 3. Случайность в домене (AD-4).
put internal/domain/zzprobe/zz_selftest_rand.go 'package zzprobe

import "math/rand/v2"

func selftestRand() int { return rand.IntN(10) }'

# 4. Нарушение порядка импорта домена: kernel (первый) импортирует доменный модуль (AD-1).
put internal/domain/kernel/zz_selftest_order.go 'package kernel

import "ant/internal/domain/zzlate"

var _ = zzlate.SelftestTarget'

# 5. Зона storage импортирует зону transport (AD-1).
put internal/infrastructure/storage/zzprobe/zz_selftest_zone.go 'package zzprobe

import transport "ant/internal/infrastructure/transport/zzprobe"

var _ = transport.SelftestTarget'

# 6. application импортирует инфраструктуру (AD-1).
put internal/application/zzprobe/zz_selftest_app.go 'package zzprobe

import storage "ant/internal/infrastructure/storage/zzprobe"

var _ = storage.SelftestTarget'

# 7. float и обход map в домене (AD-4).
put internal/domain/zzprobe/zz_selftest_float.go 'package zzprobe

func selftestFloat(m map[string]int) (s float64) {
	for range m {
		s += 0.5
	}
	return s
}'

# 8. Модуль эмитит чужой тип записи (AD-40): ops строит реакцию quality.
put internal/domain/ops/zz_selftest_emit.go 'package ops

import (
	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

func selftestEmit() (kernel.Reaction, error) {
	return kernel.NewReaction("ops", catalog.QualitySignalRaised, kernel.Slot{}, nil)
}'

lint=$("$bin/golangci-lint" run ${SELFTEST_LINT_FLAGS-} --max-issues-per-linter=0 --max-same-issues=0 ./internal/... 2>&1 || true)
det=$(go vet -vettool="$bin/detcheck" ./internal/domain/zzprobe/ 2>&1 || true)
emit=$(go vet -vettool="$bin/emitcheck" ./internal/domain/ops/ 2>&1 || true)

fail=0
expect() { # expect <вывод> <файл> <линтер> <описание>
  if grep -E "$2.*$3" <<<"$1" >/dev/null; then
    echo "  ок: $4"
  else
    echo "  НЕ ПОЙМАНО: $4 ($2, $3)"; fail=1
  fi
}
expect "$lint" 'zz_selftest_infra\.go'  'depguard'  'домен импортирует инфраструктуру'
expect "$lint" 'zz_selftest_clock\.go'  'forbidigo' 'time.Now() в домене'
expect "$lint" 'zz_selftest_rand\.go'   'depguard|forbidigo' 'math/rand в домене'
expect "$lint" 'zz_selftest_order\.go'  'depguard'  'kernel импортирует доменный модуль позже себя (порядок AD-1)'
expect "$lint" 'zz_selftest_zone\.go'   'depguard'  'storage импортирует transport'
expect "$lint" 'zz_selftest_app\.go'    'depguard'  'application импортирует инфраструктуру'
expect "$det"  'zz_selftest_float\.go'  'float'     'float64 в домене'
expect "$det"  'zz_selftest_float\.go'  'map'       'range по map в домене'
expect "$emit" 'zz_selftest_emit\.go'   'чужой тип' 'модуль эмитит чужой тип записи'

if [[ $fail != 0 ]]; then
  echo; echo "--- вывод golangci-lint:"; echo "$lint"; echo "--- вывод detcheck:"; echo "$det"; echo "--- вывод emitcheck:"; echo "$emit"
  exit 1
fi
