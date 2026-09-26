/**
 * Центр управления руководителя: четыре состояния предприятия (производство,
 * качество, поток, решения) и одна лента «Требует моего решения» — просроченные
 * решения, эскалации, неисполненная изоляция, сроки точек предъявления,
 * целостность, меры без подтверждённого эффекта. Отклонения узлов от норм — не
 * решение руководителя, а знание: они — в состоянии «Поток» и на карте.
 * Всё — из ответов сервера (живая карта, внимание, тревоги, инциденты).
 */
import type { AlertEntry, AttentionEntry, IncidentSummary, LiveMap } from '@/shared/api/generated/model'
import type { DrillRef } from '@/shared/model/drill'

export type Tone = 'normal' | 'attention' | 'danger'

export interface StateTile {
  id: 'production' | 'quality' | 'flow' | 'decisions'
  tone: Tone
  /** Главное число и подпись — ключи текста подставляет вид. */
  data: Record<string, string | number>
}

export interface DecisionRow {
  id: string
  tone: 'danger' | 'attention' | 'critical'
  kind: AttentionEntry['kind'] | AlertEntry['kind']
  entry: AttentionEntry | AlertEntry
  at?: string
  ref?: DrillRef
}

const ALERT_TONE: Partial<Record<AlertEntry['kind'], DecisionRow['tone']>> = {
  overdue_isolation: 'danger',
  not_moved_to_isolator: 'danger',
  escalation: 'danger',
  gate_overdue: 'attention',
  integrity_violation: 'critical',
}

/** Лента решений: сначала критичное и опасное, потом требующее внимания. */
export function decisionRows(attention: readonly AttentionEntry[], alerts: readonly AlertEntry[]): DecisionRow[] {
  const rows: DecisionRow[] = [
    ...alerts
      .filter((a) => ALERT_TONE[a.kind])
      .map((a) => ({ id: `al-${a.alert_id}`, tone: ALERT_TONE[a.kind]!, kind: a.kind, entry: a, at: a.at, ref: a.ref })),
    ...attention.map((e) => ({
      id: `at-${e.entry_id}`,
      tone: (e.kind === 'overdue_decision' ? 'danger' : 'attention') as DecisionRow['tone'],
      kind: e.kind,
      entry: e,
      ref: e.ref,
    })),
  ]
  const rank = { critical: 0, danger: 1, attention: 2 } as const
  return rows.sort((a, b) => rank[a.tone] - rank[b.tone])
}

/** Четыре состояния предприятия. */
export function stateTiles(map: LiveMap | null, incident: IncidentSummary | null, decisions: readonly DecisionRow[]): StateTile[] {
  const sum = (f: (c: LiveMap['counters'][number]) => number | undefined) => (map?.counters ?? []).reduce((a, c) => a + (f(c) ?? 0), 0)
  const delayed = (map?.anomalies ?? []).filter((a) => a.kind === 'queue_above_norm' || a.kind === 'wait_above_norm').length
  const ncs = sum((c) => c.nonconformities)
  const overdue = decisions.filter((d) => d.kind === 'overdue_decision' || d.kind === 'escalation' || d.kind === 'overdue_isolation').length
  return [
    { id: 'production', tone: delayed ? 'attention' : 'normal', data: { inProgress: sum((c) => c.in_progress), queue: sum((c) => c.queue), delayed } },
    {
      id: 'quality',
      tone: incident ? 'danger' : ncs ? 'attention' : 'normal',
      data: { scope: incident?.size ?? 0, confirmed: incident?.counts?.confirmed ?? 0, ncs, incident: incident ? 1 : 0 },
    },
    {
      id: 'flow',
      tone: map?.bottleneck ? 'attention' : 'normal',
      data: { bottleneck: map?.bottleneck ? map.bottleneck.step_name || map.bottleneck.step_key : '', wait: map?.bottleneck?.wait ?? '', anomalies: map?.anomalies.length ?? 0 },
    },
    { id: 'decisions', tone: overdue ? 'danger' : decisions.length ? 'attention' : 'normal', data: { n: decisions.length, overdue } },
  ]
}
