/**
 * Ссылка «провалиться в детали» (FR-7): вид сущности контракта SSE и id.
 * Одна форма для всех виджетов — узел карты, изделие, несоответствие, тревога,
 * показатель; куда вести, решает features/drill-down. Узел живой карты —
 * `live_map` + step_key.
 */
import type { EntityKind } from '@/shared/api/generated/model'

export interface DrillRef {
  entity: EntityKind
  id: string
}
