#!/usr/bin/env bash
# Самопроверка линтеров: в копии backend создаются пробные нарушения, и каждое
# должно быть поймано. Если правило перестало срабатывать — make check красный.
# Выполняется внутри golang:1.27.1: arch-selftest.sh <каталог-инструментов>.
set -euo pipefail
bin="$1"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cp -a /src/backend "$work/backend"
cd "$work/backend"
rm -rf tools

printf '\n== самопроверка линтеров: пробные нарушения должны краснеть\n'

put() { # put <путь> <содержимое>
  mkdir -p "$(dirname "$1")"
  printf '%s\n' "$2" > "$1"
}

# Пакеты-мишени (если их ещё нет в дереве).
for p in internal/infrastructure/storage/item internal/infrastructure/transport/item internal/domain/engine internal/domain/item internal/domain/reference internal/application/item; do
  pkg=$(basename "$p")
  put "$p/zz_selftest_target.go" "package $pkg

// SelftestTarget — мишень для пробных импортов.
const SelftestTarget = 1"
done

# 1. Домен импортирует инфраструктуру (AD-1).
put internal/domain/item/zz_selftest_infra.go 'package item

import storage "ant/internal/infrastructure/storage/item"

var _ = storage.SelftestTarget'

# 2. Часы в домене (AD-4).
put internal/domain/item/zz_selftest_clock.go 'package item

import "time"

func selftestClock() time.Time { return time.Now() }'

# 3. Случайность в домене (AD-4).
put internal/domain/item/zz_selftest_rand.go 'package item

import "math/rand/v2"

func selftestRand() int { return rand.IntN(10) }'

# 4. Нарушение порядка импорта домена: reference раньше engine (AD-1).
put internal/domain/reference/zz_selftest_order.go 'package reference

import "ant/internal/domain/engine"

var _ = engine.SelftestTarget'

# 5. Зона storage импортирует зону transport (AD-1).
put internal/infrastructure/storage/item/zz_selftest_zone.go 'package item

import transport "ant/internal/infrastructure/transport/item"

var _ = transport.SelftestTarget'

# 6. application импортирует инфраструктуру (AD-1).
put internal/application/item/zz_selftest_app.go 'package item

import storage "ant/internal/infrastructure/storage/item"

var _ = storage.SelftestTarget'

# 7. float и обход map в домене (AD-4).
put internal/domain/item/zz_selftest_float.go 'package item

func selftestFloat(m map[string]int) (s float64) {
	for range m {
		s += 0.5
	}
	return s
}'

lint=$("$bin/golangci-lint" run ${SELFTEST_LINT_FLAGS-} --max-issues-per-linter=0 --max-same-issues=0 ./internal/... 2>&1 || true)
det=$(go vet -vettool="$bin/detcheck" ./internal/domain/... 2>&1 || true)

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
expect "$lint" 'zz_selftest_order\.go'  'depguard'  'reference импортирует engine (порядок AD-1)'
expect "$lint" 'zz_selftest_zone\.go'   'depguard'  'storage импортирует transport'
expect "$lint" 'zz_selftest_app\.go'    'depguard'  'application импортирует инфраструктуру'
expect "$det"  'zz_selftest_float\.go'  'float'     'float64 в домене'
expect "$det"  'zz_selftest_float\.go'  'map'       'range по map в домене'

if [[ $fail != 0 ]]; then
  echo; echo "--- вывод golangci-lint:"; echo "$lint"; echo "--- вывод detcheck:"; echo "$det"
  exit 1
fi
