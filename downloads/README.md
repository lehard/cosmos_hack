# Готовые пакеты рабочего места

Собрано `make token-agent` (с демо-ключами), чтобы проверяющему не собирать самому.

| Файл | Что внутри |
|---|---|
| `glavny-token-agent-all.tar.gz` | расширение браузера с пакетом подписи WASM (одно на все ОС: Chrome, Chromium, Яндекс Браузер); агент токена под macOS arm64 и amd64, Linux amd64 и arm64, Windows amd64 (`bin/`); скрипт установки `install.sh` (macOS и Linux); демо-ключи всех персон (`demo-keys/`) |
| `glavny-demo-keys-all.tar.gz` | только демо-ключи всех персон (`*-ta@1.key.json` — ГОСТ, `*-ta-pq@1.key.json` — ML-DSA) |

Демо-ключи выводятся из публичного демо-зерна репозитория (решение Д-82): одинаковы на любой машине и после сброса стенда. Это не секрет и только для демо; в профиле `prod` такой вывод запрещён.

## Как поставить

1. Распаковать `glavny-token-agent-all.tar.gz`.
2. Браузер → `chrome://extensions` → «Режим разработчика» → «Загрузить распакованное расширение» → каталог `glavny-token-agent/extension`.
3. В расширении «Управление ключом» → «Файлы ключей» → выбрать файлы нужной персоны из `demo-keys/` (оба: `‹персона›-ta@1.key.json` и `‹персона›-ta-pq@1.key.json`) → PIN → «Загрузить и зашифровать». Разрешить адрес стенда кнопкой «Разрешить» и перезагрузить вкладку.

Ключ в браузере работает и без агента токена — для показа достаточно расширения. Агент нужен для ключа на «токене» (физический ключ): на macOS и Linux — `./install.sh`; на Windows бинарник `bin/token-agent-windows-amd64.exe` есть, но скрипта установки нет — регистрацию Native Messaging делать вручную (`token-agent-windows-amd64.exe install -extension-id aagmahobgjnlhbbkpppkfefjhocfpjce`, не проверялось).

Подробно — [docs/guides/sign_extension.md](../docs/guides/sign_extension.md).
