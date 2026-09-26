#!/usr/bin/env bash
# Проверки фронтенда для make check. Выполняется внутри node:24.21.0-slim.
#  - зависимости ставятся строго по package-lock.json (npm ci), только если
#    lock-файл изменился с прошлой установки;
#  - ESLint (запрет ручных HTTP-вызовов, AD-20) + самопроверка правила;
#  - оболочка (эпик 03): столы normative/desks ↔ реестр виджетов, тексты,
#    сгенерированный клиент совпадает с контрактом (+ самопроверка);
#  - vue-tsc; юнит-тесты (vitest); сборка vite.
set -euo pipefail
cd /src/frontend

step() { printf '\n== %s\n' "$*"; }

step "зависимости (npm ci по lock-файлу)"
stamp=node_modules/.ant-lock.sha256
want=$(sha256sum package-lock.json | cut -d' ' -f1)
if [[ ! -f $stamp || "$(cat "$stamp")" != "$want" ]]; then
  seed=()
  # офлайн-кэш из make npm-cache, если он наполнен
  [[ -n "$(ls -A .npm-cache 2>/dev/null | grep -v '^.gitkeep$')" ]] && seed=(--cache /src/frontend/.npm-cache)
  npm ci --prefer-offline --no-audit --no-fund "${seed[@]}"
  echo "$want" > "$stamp"
else
  echo "без изменений"
fi

step "ESLint"
npx --no-install eslint --max-warnings=0 .

step "самопроверка ESLint: ручной HTTP-вызов должен краснеть"
probe='export const x = () => fetch("/api/items"); export const y = () => window.fetch("/api/items");'
if out=$(printf '%s\n' "$probe" | npx --no-install eslint --stdin --stdin-filename src/pages/zz-selftest.ts 2>&1); then
  echo "  НЕ ПОЙМАНО: fetch в src/pages"; exit 1
fi
grep -q 'no-restricted-globals' <<<"$out" && grep -q 'no-restricted-properties' <<<"$out" \
  || { echo "  НЕ ПОЙМАНО полностью:"; echo "$out"; exit 1; }
echo "  ок: fetch и window.fetch запрещены вне src/shared/api/sse"
probe_import='import axios from "axios"; export default axios;'
if printf '%s\n' "$probe_import" | npx --no-install eslint --stdin --stdin-filename src/features/zz-selftest.ts >/dev/null 2>&1; then
  echo "  НЕ ПОЙМАНО: import axios"; exit 1
fi
echo "  ок: HTTP-библиотеки запрещены"

step "оболочка: столы ролей ↔ реестр виджетов, тексты, сгенерированный клиент не устарел"
npm run --silent check:shell

step "vue-tsc"
npm run --silent typecheck

step "юнит-тесты оболочки"
npm run --silent test

step "сборка vite"
npm run --silent build
