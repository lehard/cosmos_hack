#!/usr/bin/env bash
# Сборка пакета рабочего места (make token-agent, эпик 38): тесты ядра,
# пакет подписи WASM для расширения и агент токена под macOS (arm64 — основная
# цель, amd64) и Linux amd64. Запускается в контейнере Go из backend/.
#   build.sh ‹версия› ‹каталог пакета›
set -euo pipefail
VERSION="${1:-dev}"
OUT="${2:-/src/dist/glavny-token-agent}"
LD="-s -w -X ant/cmd/token-agent/internal/agent.Build=${VERSION}"
go test ./cmd/token-agent/...
mkdir -p "$OUT/extension" "$OUT/bin"
GOOS=js GOARCH=wasm go build -trimpath -ldflags="$LD" -o "$OUT/extension/signer.wasm" ./cmd/token-agent/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$OUT/extension/"
for t in darwin/arm64 darwin/amd64 linux/amd64; do
  GOOS="${t%/*}" GOARCH="${t#*/}" go build -trimpath -ldflags="$LD" -o "$OUT/bin/token-agent-${t%/*}-${t#*/}" ./cmd/token-agent
done
echo "собрано: $OUT"
