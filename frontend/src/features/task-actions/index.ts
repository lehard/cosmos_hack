/**
 * Фича «задачи цеха» (FR-55, FR-57, FR-137; эпик 13) — публичный вход (FSD):
 * список задач и запросов решения с отметкой «выполнено / принято / отклонено»
 * и подтверждение физического перемещения в изолятор — с задачи мастера и с
 * терминала исполнителя.
 */
export * from './model/actions'
export * from './model/isolation'
export { useIsolatorMove } from './model/use-isolator-move'
export { default as IsolatorMoveConfirm } from './ui/IsolatorMoveConfirm.vue'
export { default as TaskInbox } from './ui/TaskInbox.vue'
