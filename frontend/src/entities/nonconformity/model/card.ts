/**
 * Правила показа карточки несоответствия и очереди (FR-51, FR-52, FR-55,
 * NFR-UI-4): состояние экрана, версии вывода системы, срок решения. Ничего не
 * вычисляет за сервер: ранг риска, порядок очереди, срок и статусы приходят
 * готовыми; здесь только выбор текста и раскладка.
 */
import { qualityState } from '@/entities/item'
import type { WidgetDataState } from '@/shared/config/widget'
import type { NCCard, NCConclusionVersion } from './types'

/**
 * Состояние данных карточки для рамки — по оси «Состояние качества» изделия
 * (как у паспорта): сигнал и подтверждённое несоответствие → «признак
 * дефекта»; «оценка невозможна» → своё состояние; блок и изоляция — не дефект.
 */
export const ncCardState = (card: Pick<NCCard, 'axes'>): WidgetDataState => qualityState(card.axes)

/** Версии вывода системы: текущая и прежние (FR-32: прежний вывод сохранён). */
export interface ConclusionVersions {
  current: NCConclusionVersion | null
  /** Прежние версии, от новых к старым. */
  previous: NCConclusionVersion[]
}

/**
 * Разложить версии вывода: последняя по `version` — текущая.
 * @param versions — версии в любом порядке
 */
export function conclusionVersions(versions: readonly NCConclusionVersion[]): ConclusionVersions {
  const sorted = [...versions].sort((a, b) => b.version - a.version)
  return { current: sorted[0] ?? null, previous: sorted.slice(1) }
}

/**
 * Решения людей, принятые до текущей версии вывода системы (FR-32: «решение
 * принято до новых данных — пересмотрите»): время решения раньше записи
 * пересмотренного вывода.
 * @param card — карточка
 */
export function decisionsBeforeRevision(card: Pick<NCCard, 'human_decisions' | 'system_analysis'>): Set<string> {
  const { current } = conclusionVersions(card.system_analysis.versions)
  if (!current || current.revised_due_to == null) return new Set()
  return new Set(card.human_decisions.filter((d) => d.occurred_at < current.recorded_at).map((d) => d.event_id))
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

/** Порядок очереди, который считает сервер (`sort`). */
export type QueueSort = 'risk' | 'deadline'

/**
 * Порядок очереди из среза стола `sort: [risk, deadline]`: первый известный
 * ключ; по умолчанию — риск (PRD §3a). Сортирует сервер.
 * @param value — `slice.sort`
 */
export function queueSortOf(value: unknown): QueueSort {
  const list = Array.isArray(value) ? value : [value]
  const first = list.find((v): v is QueueSort => v === 'risk' || v === 'deadline')
  return first ?? 'risk'
}
