/**
 * Правила показа карточки несоответствия и очереди (FR-51, FR-52, FR-55,
 * NFR-UI-4): состояние экрана, версии вывода системы, срок решения. Ничего не
 * вычисляет за сервер: ранг риска, порядок очереди, срок и статусы приходят
 * готовыми; здесь только выбор текста и раскладка.
 */
import { qualityState } from '@/entities/item'
import type { WidgetDataState } from '@/shared/config/widget'
import type { DecisionQueueRow, NCCard, NCConclusionVersion, NCRecordRef } from './types'

/** Ключ строки очереди: вид и объект. */
export const rowKey = (row: Pick<DecisionQueueRow, 'kind' | 'object_id'>): string => `${row.kind}:${row.object_id}`

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

// ─────────────────────────── хронология доказательств ───────────────────────────

/** Тон записи в хронологии: норма, обстоятельство для внимания, признак дефекта, нейтрально. */
export type RecordTone = 'ok' | 'warn' | 'danger' | 'neutral'

/** Типы записей-обстоятельств: отклонение режима и ручное вмешательство (не вина, а факт). */
const WARN_TYPES = new Set(['equipment.deviation.detected', 'operator.override.performed'])

/**
 * Тон записи по её смыслу, без вычислений за сервер: итог контроля
 * (`params.outcome`) и вид записи-обстоятельства.
 * @param r — запись карточки
 */
export function recordTone(r: Pick<NCRecordRef, 'event_type' | 'params'>): RecordTone {
  const outcome = r.params?.outcome
  if (outcome === 'defect_indicated') return 'danger'
  if (outcome === 'unable_to_assess') return 'warn'
  if (outcome === 'no_defect_indicated') return 'ok'
  if (WARN_TYPES.has(r.event_type)) return 'warn'
  return 'neutral'
}

/** Фаза хронологии относительно операции. */
export type EvidencePhase = 'before' | 'during' | 'after'

/** Фаза хронологии с записями по времени. */
export interface EvidencePhaseBlock {
  phase: EvidencePhase
  records: NCRecordRef[]
}

const byTime = (a: NCRecordRef, b: NCRecordRef) => a.occurred_at.localeCompare(b.occurred_at) || (a.seq ?? 0) - (b.seq ?? 0)

/**
 * «Как было дело»: до операции → во время → после, записи в каждой фазе по
 * времени (сервер отдаёт их как есть, ручные отметки — без номера записи).
 * @param card — карточка
 */
export function evidencePhases(card: Pick<NCCard, 'happened'>): EvidencePhaseBlock[] {
  const h = card.happened
  return (['before', 'during', 'after'] as const).map((phase) => ({ phase, records: [...h[phase]].sort(byTime) }))
}

/**
 * Записи истории зоны, которых нет в хронологии «до / во время / после», —
 * их показываем отдельно, остальное не дублируем.
 * @param card — карточка
 */
export function extraZoneHistory(card: Pick<NCCard, 'happened' | 'evidence'>): NCRecordRef[] {
  const shown = new Set([...card.happened.before, ...card.happened.during, ...card.happened.after].map((r) => r.event_id))
  return card.evidence.zone_history.filter((r) => !shown.has(r.event_id)).sort(byTime)
}
