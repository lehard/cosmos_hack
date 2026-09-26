#!/usr/bin/env bash
# Установка агента токена «Главный» на macOS (или Linux): бинарник в каталог
# пользователя и манифест Native Messaging для Chrome, Chromium и Яндекс
# Браузера. Расширение ставится отдельно (README, шаг 2): chrome://extensions →
# режим разработчика → «Загрузить распакованное» → каталог extension/.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
EXT_ID="aagmahobgjnlhbbkpppkfefjhocfpjce"
case "$(uname -s)" in
  Darwin) DEST="$HOME/Library/Application Support/Glavny/token-agent/bin" ;;
  *) DEST="$HOME/.local/share/glavny/token-agent/bin" ;;
esac
BIN="$HERE/bin/token-agent-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"
if [ ! -x "$BIN" ]; then
  echo "Нет сборки агента для $(uname -s) $(uname -m): $BIN" >&2
  echo "Ключ в браузере работает и без агента — достаточно расширения." >&2
  exit 1
fi
mkdir -p "$DEST"
cp "$BIN" "$DEST/token-agent"
chmod 755 "$DEST/token-agent"
if [ "$(uname -s)" = Darwin ]; then
  # Снять карантин загрузки (бинарник подписан ad-hoc компоновщиком Go).
  xattr -d com.apple.quarantine "$DEST/token-agent" 2>/dev/null || true
  xattr -dr com.apple.quarantine "$HERE/extension" 2>/dev/null || true
fi
"$DEST/token-agent" install -extension-id "$EXT_ID"
echo
echo "Агент токена: $DEST/token-agent"
echo "Положить ключ на «токен» (физический ключ, класс hardware_token):"
echo "  \"$DEST/token-agent\" import -key ‹файл›-ta@1.key.json [-token /Volumes/‹флешка›]"
echo "Расширение: chrome://extensions → Режим разработчика → Загрузить распакованное → $HERE/extension"
