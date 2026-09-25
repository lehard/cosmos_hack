# internal — контракты собственных процессов (AD-46)

| Файл | Кто ⇄ кто | Что |
|---|---|---|
| `token-agent/nm-request.v1.json`, `nm-response.v1.json` | браузерное расширение ⇄ агент токена (Native Messaging) | привет, состояние, подпись и пачка подписей, сменный рапорт, локальный журнал (AD-14) |
| `token-agent/shift-report.v1.json` | агент токена → ant | сменный рапорт уровня 3: корень Меркла RFC 6962 и счётчики по типам (AD-12) |
| `token-agent/level1-actions.yaml` | вкомпилирован в агент | перечень действий уровня подписи 1; остальное агент на уровне 1 отклоняет сам (AD-13) |
| `keeper.openapi.yaml` | ant, verifier ⇄ keeper (mTLS) | головы и звенья, контрольные точки, отчёты верификатора (AD-8, AD-46) |
| `verifier/report.v1.json` | verifier → keeper | подписанный отчёт верификатора (AD-9) |
| `demo-signer.openapi.yaml` | simulation → demo-signer | выполнить шаг сценария от демо-персоны (AD-26, AD-33) |
| `stands/telemetry.v1.json` | stand оборудования → edge-агент | отсчёты и события MTConnect (FR-149) |

Пачки edge-агента идут обычной операцией приёма из `contracts/openapi.yaml`. Все сообщения здесь — закрытые схемы (AD-10); Go- и TS-типы генерирует тот же `make generate` (эпик 02).
