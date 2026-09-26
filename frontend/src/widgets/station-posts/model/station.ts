/**
 * Участок мастера (PRD §3a «Мастер участка — Участок», FR-81, UJ-7): операции
 * цеха и посты — чистые функции над ответами сервера, без пересчёта чисел.
 *
 * Откуда что (AD-21: счётчики и показатели считает сервер):
 * - операции цеха, нормы времени и лимит доработок — схема процесса (AD-17);
 * - очередь, в работе, несоответствия, аномалии, ограничение линии —
 *   `process.live_map.read`;
 * - повторные выполнения, незавершённые операции, средняя длительность —
 *   `analytics.overview.read`, срез по шагу (`step:‹step_key›`);
 * - посты, назначенные, присутствие, текущая деталь — `access.workplace.list`;
 * - оборудование поста — `machinelogs.equipment.list` (`station_id` = пост) и
 *   пригодность по поверке — `reference.equipment.list`.
 *
 * «Почему растёт очередь» (UJ-7) — не вывод о причине, а факты участка рядом с
 * растущей очередью: кто из назначенных не на месте, какой пост без людей,
 * какое оборудование стоит, без данных или непригодно, сколько незавершённых
 * операций (NFR-UI-4).
 */
import type { NodeAnomaly, NodeCounters, Bottleneck, LiveMapData, ProcessStep } from '@/entities/live-map'
import type { EquipmentState, RefEquipment } from '@/entities/equipment'
import type { MetricRow, MetricValue } from '@/entities/metric'
import type { PostRow } from '@/entities/workplace'

/** Операция участка со счётчиками и показателями. */
export interface StationStep {
  step: ProcessStep
  counters: NodeCounters | null
  anomalies: NodeAnomaly[]
  bottleneck: Bottleneck | null
  /** Нет данных источника — оценка невозможна (не «норма»). */
  dataGap: boolean
  /** Повторные выполнения за период (`rework_runs`). */
  reworkRuns: MetricValue | null
  /** Незавершённые операции (`unfinished_operations`). */
  unfinished: MetricValue | null
  /** Средняя длительность операции (`operation_duration`). */
  meanDuration: MetricValue | null
  /** Очередь растёт: аномалия очереди или ожидания, или узел — ограничение линии. */
  growing: boolean
}

/** Пост участка с оборудованием. */
export interface StationPost {
  post: PostRow
  equipment: EquipmentState[]
}

/** Показатели из обзора аналитики, которые нужны участку. */
const METRICS = {
  reworkRuns: 'rework_runs',
  unfinished: 'unfinished_operations',
  meanDuration: 'operation_duration',
} as const

/** Значение показателя в срезе по шагу; нет среза — null (не передано ≠ ноль). */
export function stepMetric(rows: readonly MetricRow[] | null | undefined, metricId: string, stepKey: string): MetricValue | null {
  const row = rows?.find((r) => r.metric_id === metricId)
  if (!row || row.unknown) return null
  const slice = row.slices.find((s) => s.dimension === 'step' && (s.key === `step:${stepKey}` || s.key === stepKey))
  return slice?.value ?? null
}

/** Аномалии, при которых очередь растёт (FR-5). */
const GROWING: ReadonlySet<string> = new Set(['queue_above_norm', 'wait_above_norm'])

/**
 * Операции участка с данными карты и аналитики.
 * @param operations — операции цеха из схемы
 * @param map — живая карта (счётчики, аномалии, ограничение)
 * @param metrics — строки обзора аналитики
 */
export function buildSteps(operations: readonly ProcessStep[], map: LiveMapData | null | undefined, metrics: readonly MetricRow[] | null | undefined): StationStep[] {
  return operations.map((step) => {
    const k = step.stepKey
    const anomalies = map?.anomalies.filter((a) => a.step_key === k) ?? []
    const bottleneck = map?.bottleneck?.step_key === k ? map.bottleneck : null
    return {
      step,
      counters: map?.counters.find((c) => c.step_key === k) ?? null,
      anomalies,
      bottleneck,
      dataGap: map?.data_gaps.includes(k) ?? false,
      reworkRuns: stepMetric(metrics, METRICS.reworkRuns, k),
      unfinished: stepMetric(metrics, METRICS.unfinished, k),
      meanDuration: stepMetric(metrics, METRICS.meanDuration, k),
      growing: !!bottleneck || anomalies.some((a) => GROWING.has(a.kind)),
    }
  })
}

/** Посты участка с оборудованием поста (`station_id` = пост). */
export function buildPosts(posts: readonly PostRow[], equipment: readonly EquipmentState[] | null | undefined): StationPost[] {
  return posts.map((post) => ({ post, equipment: (equipment ?? []).filter((e) => e.station_id === post.workplace_id) }))
}

/** Факт участка рядом с растущей очередью. */
export type QueueFact =
  | { kind: 'presence'; post: string; person: string; presence: PostRow['presence'] }
  | { kind: 'not_assigned'; post: string }
  | { kind: 'equipment_state'; equipment: string; execution: EquipmentState['execution']; condition: EquipmentState['condition'] }
  | { kind: 'equipment_no_data'; equipment: string }
  | { kind: 'equipment_unusable'; equipment: string; reason: string }
  | { kind: 'unfinished'; value: MetricValue }
  | { kind: 'data_gap' }

/** Присутствие, при котором назначенного нет у поста (FR-6, FR-84). */
const AWAY: ReadonlySet<PostRow['presence']> = new Set(['key_missing', 'owner_absent', 'absent'])
/** Оборудование не работает. */
const DOWN: ReadonlySet<EquipmentState['execution']> = new Set(['stopped', 'interrupted'])

/**
 * Факты участка для растущей очереди на шаге.
 * @param step — операция с растущей очередью
 * @param posts — посты участка
 * @param registry — справочник оборудования (пригодность по поверке)
 */
export function queueFacts(step: StationStep, posts: readonly StationPost[], registry: readonly RefEquipment[] | null | undefined): QueueFact[] {
  const facts: QueueFact[] = []
  for (const { post, equipment } of posts) {
    if (post.presence === 'not_assigned') facts.push({ kind: 'not_assigned', post: post.station })
    else if (AWAY.has(post.presence)) facts.push({ kind: 'presence', post: post.station, person: post.assigned?.display ?? '—', presence: post.presence })
    for (const e of equipment) {
      if (e.condition === 'unknown' || e.execution === 'unknown') facts.push({ kind: 'equipment_no_data', equipment: e.title })
      else if (DOWN.has(e.execution) || e.condition === 'fault') facts.push({ kind: 'equipment_state', equipment: e.title, execution: e.execution, condition: e.condition })
      const reg = registry?.find((r) => r.equipment_id === e.equipment_id)
      if (reg && !reg.usable) facts.push({ kind: 'equipment_unusable', equipment: e.title, reason: reg.unusable_reason ?? '' })
    }
  }
  if (step.unfinished && step.unfinished.value > 0) facts.push({ kind: 'unfinished', value: step.unfinished })
  if (step.dataGap) facts.push({ kind: 'data_gap' })
  return facts
}

/** Что пришло и ждёт приёмки: задача процесса «Принять в цех» с изделием. */
export interface IncomingTask {
  taskId: string
  itemId: string
  /** Заголовок задачи («Принять в цех Ф-001»). */
  title: string
}

/** Изделие на входе участка (очередь первой операции). */
export interface EntryItem {
  item_id: string
  label: string
}

