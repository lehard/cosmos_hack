/**
 * Каталог показателей на экране: id показателя контракта → определение для
 * подписи (тексты `analytics.metrics.*` из ru.json процессной сессии) и раздел
 * раздельного учёта (кейс §2.4, §5.2; FR-86…FR-89).
 *
 * Название показателя приходит с сервера (`title`, по словарю продукта; Д-11 —
 * «Прохождение контроля с первого раза»). Здесь — только подсказка «что
 * считается», чтобы число не читалось шире своего смысла (NFR-UI-4).
 */
import type { MetricRow, MetricSliceDimension } from '@/shared/api/generated/model'

/** id показателя → ключ определения (подсказка у числа). */
const METRIC_HINT: Record<string, string> = {
  inspected_items: 'analytics.metrics.inspectedItems.hint',
  items_with_confirmed_nc: 'analytics.metrics.nonconformingItems.hint',
  confirmed_defects: 'analytics.metrics.defectCount.hint',
  defect_count: 'analytics.metrics.defectCount.hint',
  defects_by_type: 'analytics.metrics.defectsByType.hint',
  first_pass_yield: 'analytics.metrics.firstPassYield.hint',
  cause_established: 'analytics.metrics.causeEstablished.hint',
  incoming_defects: 'analytics.metrics.incomingDefectShare.hint',
  incoming_defect_share: 'analytics.metrics.incomingDefectShare.hint',
  unable_to_assess: 'analytics.unableToAssessBucket',
  rework_runs: 'analytics.metrics.reworkRuns.hint',
  unfinished_operations: 'analytics.metrics.unfinishedOperations.hint',
  comparable_welds: 'analytics.metrics.comparableWork.hint',
  confirmed_performer_errors: 'analytics.metrics.comparableWork.hint',
  equipment_downtime: 'analytics.metrics.downtime.hint',
  downtime: 'analytics.metrics.downtime.hint',
  lead_time: 'widgets.analytics.hints.leadTime',
  waiting_time: 'analytics.metrics.waiting.hint',
  operation_duration: 'analytics.metrics.operationDuration.hint',
  detection_delay: 'analytics.metrics.detectionDelay.hint',
  scrap_losses: 'analytics.metrics.scrapLosses.hint',
  recurrence_rate: 'analytics.metrics.recurrenceRate.hint',
  representations: 'analytics.metrics.representations.hint',
}

/** Ключ определения показателя или null — определения в словаре нет. */
export const metricHintKey = (metricId: string): string | null => METRIC_HINT[metricId] ?? null

/**
 * Раздельный учёт (FR-87, кейс §2.4): входной брак, проблемы оборудования, ошибки
 * исполнителей и гипотезы — четыре отдельные графы, не складываются.
 */
export type Account = 'incoming' | 'equipment' | 'people' | 'hypotheses'

/** Порядок граф раздельного учёта на экране. */
export const ACCOUNTS: readonly Account[] = ['incoming', 'equipment', 'people', 'hypotheses']

/**
 * Графа раздельного учёта строки показателя или null — строка не из раздельного
 * учёта. Оборудование и исполнители — по разделу контракта (`group`); входной
 * брак — по происхождению (срез `origin=incoming` или id показателя); гипотезы —
 * по id показателя. Отдельных разделов «входной брак» и «гипотезы» в контракте
 * v1 нет — см. отчёт эпика 15 (предложение расширить `MetricRow.group`).
 */
export function accountOf(row: Pick<MetricRow, 'metric_id' | 'group' | 'slices'>): Account | null {
  if (/incoming/.test(row.metric_id) || row.slices.some((s) => s.dimension === 'origin' && s.key === 'incoming')) return 'incoming'
  if (/hypothes/.test(row.metric_id)) return 'hypotheses'
  if (row.group === 'equipment') return 'equipment'
  if (row.group === 'people') return 'people'
  return null
}

/** Ключ названия графы раздельного учёта. */
export const ACCOUNT_TEXT: Record<Account, string> = {
  incoming: 'widgets.analytics.accounts.incoming',
  equipment: 'widgets.analytics.accounts.equipment',
  people: 'widgets.analytics.accounts.people',
  hypotheses: 'widgets.analytics.accounts.hypotheses',
}

/** Ключ названия измерения среза. */
export const DIMENSION_TEXT: Record<MetricSliceDimension, string> = {
  location: 'widgets.analytics.dimensions.location',
  step: 'analytics.slices.operation',
  equipment: 'analytics.slices.equipment',
  performer: 'analytics.slices.performer',
  shift: 'analytics.slices.shift',
  defect_type: 'analytics.slices.defectType',
  origin: 'widgets.analytics.dimensions.origin',
}
