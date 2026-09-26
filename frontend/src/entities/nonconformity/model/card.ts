/**
 * Правила показа карточки несоответствия и очереди (FR-51, FR-52, FR-55,
 * NFR-UI-4): состояние экрана, версии вывода системы, срок решения, порядок
 * очереди. Ничего не вычисляет за сервер: ранг риска, срок и статусы приходят
 * готовыми; здесь только порядок показа и выбор текста.
 */
import { qualityState } from '@/entities/item'
import type { WidgetDataState } from '@/shared/config/widget'
import type { NcCard, QueueEntry, SystemConclusion } from './types'

/**
 * Состояние данных карточки для рамки — по оси «Состояние качества» изделия
 * (как у паспорта): сигнал и подтверждённое несоответствие → «признак
 * дефекта»; «оценка невозможна» → своё состояние; блок и изоляция — не дефект.
 */
export const ncCardState = (card: Pick<NcCard, 'item_statuses'>): WidgetDataState => qualityState(card.item_statuses)

/** Версии вывода системы: текущая и прежние (FR-32: прежний вывод сохранён). */
export interface ConclusionVersions {
  current: SystemConclusion | null
  /** Прежние версии, от новых к старым. */
  previous: SystemConclusion[]
}

/**
 * Разложить версии вывода: последняя по `version` — текущая.
 * @param conclusions — версии в любом порядке
 */
export function conclusionVersions(conclusions: readonly SystemConclusion[]): ConclusionVersions {
  const sorted = [...conclusions].sort((a, b) => b.version - a.version)
  return { current: sorted[0] ?? null, previous: sorted.slice(1) }
}

/** Срок решения относительно «сейчас» (FR-55): осталось или просрочено, минуты. */
export interface Deadline {
  overdue: boolean
  /** Минут до срока или после него, целое неотрицательное. */
  minutes: number
}

/**
 * Срок решения. «Сейчас» — доменное время интерфейса (в воспроизведении —
 * момент воспроизведения), его передаёт вызывающий.
 * @param dueAt — срок, RFC 3339
 * @param now — «сейчас», мс
 */
export function deadlineOf(dueAt: string | null | undefined, now: number): Deadline | null {
  if (!dueAt) return null
  const diff = Date.parse(dueAt) - now
  if (Number.isNaN(diff)) return null
  return { overdue: diff < 0, minutes: Math.floor(Math.abs(diff) / 60_000) }
}

/** Изолировано в системе, физически не перемещено (FR-55) — показывается отдельно. */
export const notPhysicallyMoved = (card: Pick<NcCard, 'isolation'>): boolean => !!card.isolation && !card.isolation.physically_moved

// ─────────────────────────────── очередь ───────────────────────────────

/** Ключ сортировки очереди (срез стола `sort: [risk, deadline]`). */
export type QueueSortKey = 'risk' | 'deadline'

const SORT_KEYS = new Set<string>(['risk', 'deadline'])

/**
 * Порядок сортировки из среза стола; неизвестные ключи отбрасываются,
 * по умолчанию — риск, затем срок (PRD §3a).
 * @param value — `slice.sort`
 */
export function queueSortOf(value: unknown): QueueSortKey[] {
  const keys = Array.isArray(value) ? value.filter((v): v is QueueSortKey => typeof v === 'string' && SORT_KEYS.has(v)) : []
  return keys.length ? keys : ['risk', 'deadline']
}

const byRisk = (a: QueueEntry, b: QueueEntry) => b.risk_rank - a.risk_rank
/** Раньше срок — выше; без срока — в конце. */
const byDeadline = (a: QueueEntry, b: QueueEntry) => {
  if (a.due_at === b.due_at) return 0
  if (!a.due_at) return 1
  if (!b.due_at) return -1
  return Date.parse(a.due_at) - Date.parse(b.due_at)
}

/**
 * Очередь «Ждут моего решения» по риску и сроку (PRD §3a); при равенстве —
 * кто раньше поступил.
 * @param entries — строки очереди
 * @param order — порядок ключей
 */
export function sortQueue(entries: readonly QueueEntry[], order: readonly QueueSortKey[] = ['risk', 'deadline']): QueueEntry[] {
  const cmp = order.map((k) => (k === 'risk' ? byRisk : byDeadline))
  return [...entries].sort((a, b) => {
    for (const c of cmp) {
      const r = c(a, b)
      if (r) return r
    }
    return a.received_at.localeCompare(b.received_at) || a.entry_id.localeCompare(b.entry_id)
  })
}

/** Ключ текста вида строки очереди. */
export const QUEUE_KIND_TEXT: Record<QueueEntry['kind'], string> = {
  presentation_point: 'common.words.presentationPoint',
  signal_review: 'widgets.decisionQueue.kind.signalReview',
  isolated_item: 'widgets.decisionQueue.kind.isolatedItem',
}
