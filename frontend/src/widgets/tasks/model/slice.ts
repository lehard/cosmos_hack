/**
 * Срез виджета «Задачи» (стол мастера, терминал исполнителя; FR-57, FR-137):
 * - `kinds` — какие виды показать: `task` (задачи и запросы решения), `alarm`
 *   (тревоги), `escalation` (просроченные решения с ценой задержки); пусто — все;
 * - `scope: own_workplace` — только задачи своего рабочего места (и без места);
 * - `run_id` — прогон сценария (AD-38).
 */
import type { TaskEntry } from '@/entities/task'
import type { DrillRef } from '@/shared/model/drill'

/** Раздел виджета. */
export type TaskSection = 'task' | 'alarm' | 'escalation'

const SECTIONS: readonly TaskSection[] = ['task', 'alarm', 'escalation']

/** Разделы по срезу стола; неизвестные значения пропускаются. */
export function sectionsOf(slice: Record<string, unknown>): TaskSection[] {
  const kinds = Array.isArray(slice.kinds) ? slice.kinds.filter((k): k is TaskSection => SECTIONS.includes(k as TaskSection)) : []
  return kinds.length ? kinds : [...SECTIONS]
}

/** Прогон сценария из среза. */
export const runOf = (slice: Record<string, unknown>): string | undefined => (typeof slice.run_id === 'string' && slice.run_id ? slice.run_id : undefined)

/**
 * Задачи своего рабочего места: с этим местом или без места (адресность по
 * роли и сотруднику решил сервер). Места в сеансе нет — все адресованные.
 */
export function ownWorkplaceTasks(tasks: readonly TaskEntry[], workplaceId: string | null | undefined): TaskEntry[] {
  if (!workplaceId) return [...tasks]
  // Своё место — задачи поста; задачи цеха или участка (не привязанные к другому посту) тоже свои.
  return tasks.filter((t) => !t.location_id || t.location_id === workplaceId || !t.location_id.startsWith('WP-'))
}

/** Открытые — сначала просроченные, затем по сроку; закрытые — отдельно, новые сверху. */
export function splitTasks(tasks: readonly TaskEntry[]): { open: TaskEntry[]; closed: TaskEntry[] } {
  const due = (t: TaskEntry) => (t.due_at ? Date.parse(t.due_at) : Number.MAX_SAFE_INTEGER)
  const open = tasks.filter((t) => t.state === 'open').sort((a, b) => Number(b.overdue) - Number(a.overdue) || due(a) - due(b))
  const closed = tasks.filter((t) => t.state !== 'open')
  return { open, closed }
}

/** Строка тревоги или «требует внимания», уже с текстом. */
export interface NoticeRow {
  id: string
  text: string
  /** Время (для тревог). */
  time?: string
  /** Тревожная (красная) или предупреждение. */
  severe: boolean
  ref: DrillRef | null
}
