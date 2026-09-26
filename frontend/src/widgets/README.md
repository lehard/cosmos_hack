Слой `widgets` (Feature-Sliced Design). Каждый виджет — папка `‹id›/` с `index.ts`
(публичный вход) и `ui/`; список — в `registry.ts`, на него ссылаются столы ролей
`normative/desks/‹роль›.yaml`. Виджет получает `WidgetProps` (срез и плотность со
стола) и рисует себя внутри `WidgetFrame` из `shared/ui`. Папки заготовлены эпиком 03
для всех виджетов эпиков 10–15; эпики наполняют их, не трогая `registry.ts` и
`WidgetHost.vue`.
