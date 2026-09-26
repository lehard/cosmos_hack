/**
 * Виджет «journal» — публичный вход (FSD): общий журнал с полосой целостности
 * (PRD §3a; эпик 14). Оболочка грузит его через widgets/registry.ts и передаёт
 * WidgetProps (shared/config/widget.ts); срез `entry_kind` — вид записи по умолчанию.
 */
export { default } from './ui/JournalWidget.vue'
