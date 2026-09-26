/**
 * Кнопки нижней панели окна версии (Д-70) по жизненному циклу версии
 * (FR-22, FR-23): черновик → на утверждении → действующая → выведена.
 * Показывается кнопка, если действие есть у статуса и разрешено правами
 * (x-ant-action); включена — если выполнены условия (изменения сохранены,
 * маршрут кворума закрыт). Решает сервер — здесь только подсказка экрана.
 */

/** Статус версии процесса (контракт `status`). */
export type VersionStatus = 'draft' | 'on_approval' | 'active' | 'retired'

/** Действие окна версии. */
export type VersionAction = 'newDraft' | 'edit' | 'saveDraft' | 'submit' | 'activate' | 'retire'

/** x-ant-action операции действия. */
export const ACTION_ID: Record<VersionAction, string> = {
  newDraft: 'process.version.draft',
  edit: 'process.version.draft',
  saveDraft: 'process.version.draft',
  submit: 'process.version.submit',
  activate: 'process.version.activate',
  retire: 'process.version.retire',
}

/** Какие действия есть у статуса (в порядке кнопок, главное — последним). */
export const ACTIONS_BY_STATUS: Record<VersionStatus, VersionAction[]> = {
  // Правка схемы — на весь экран (UI-34): «Править схему» открывает рабочее место, «Сохранить» — там.
  draft: ['edit', 'submit'],
  on_approval: ['activate'],
  active: ['newDraft', 'retire'],
  retired: ['newDraft'],
}

/** Условия экрана. */
export interface LifecycleState {
  /** В модельере есть несохранённые изменения. */
  dirty: boolean
  /** Маршрут листа утверждения закрыт (document.route.closed). */
  routeClosed: boolean
  /** Правка чужой (не черновик) версии уже начата — новый черновик в модельере. */
  editing: boolean
}

/** Кнопка окна. */
export interface ActionButtonState {
  action: VersionAction
  enabled: boolean
}

/**
 * Кнопки окна версии.
 * @param status — статус версии
 * @param can — разрешено ли действие (x-ant-action)
 * @param s — условия экрана
 */
export function actionsFor(status: string, can: (actionId: string) => boolean, s: LifecycleState): ActionButtonState[] {
  const list = ACTIONS_BY_STATUS[status as VersionStatus] ?? []
  // Правка начата из действующей или выведенной — вместо «Новый черновик» «Сохранить».
  const actions: VersionAction[] = s.editing && status !== 'draft' ? ['saveDraft'] : list
  return actions
    .filter((a) => can(ACTION_ID[a]))
    .map((a) => {
      switch (a) {
        case 'saveDraft':
          return { action: a, enabled: s.dirty }
        case 'submit':
          // Отправляется сохранённое: несохранённые правки — сначала сохранить.
          return { action: a, enabled: !s.dirty }
        case 'activate':
          return { action: a, enabled: s.routeClosed }
        default:
          return { action: a, enabled: true }
      }
    })
}

/** Следующая свободная метка версии: v1, v2… → v‹max+1›. */
export function nextLabel(labels: readonly string[]): string {
  let max = 0
  for (const l of labels) {
    const m = /^v(\d+)/i.exec(l.trim())
    if (m) max = Math.max(max, Number(m[1]))
  }
  return `v${max + 1}`
}
