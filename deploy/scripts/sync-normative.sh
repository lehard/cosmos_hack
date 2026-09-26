#!/usr/bin/env bash
# Обновляет встроенные копии нормативного слоя (normative/…) во всех seed/
# модулей хранения и во входе мира заготовок: копирует только те файлы,
# которые в копии уже есть (состав копии задаёт модуль). После правки
# normative/ — запустить, затем make fixtures-update (перегенерация заготовок).
# Проверяют копии тесты TestSeedMatchesRepo каждого модуля.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"
n=0
while IFS= read -r copy; do
  src="normative/${copy#*/normative/}"
  if [[ -f "$src" ]] && ! cmp -s "$src" "$copy"; then
    cp "$src" "$copy"; echo "обновлено: $copy"; n=$((n+1))
  fi
done < <(find backend -path '*/normative/*' -type f \( -path '*/seed/normative/*' -o -path '*/world/input/normative/*' \))
echo "sync-normative: обновлено файлов — $n"
