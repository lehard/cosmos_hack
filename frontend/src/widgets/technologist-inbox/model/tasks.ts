/**
 * Задачи рабочего стола технолога — из того, что уже отдаёт сервер: открытые
 * расследования (стадия, «что дальше»), повторяющиеся проблемы без мер, меры
 * (просрочены, не помогли, пора оценить), версии процесса (черновик, на
 * утверждении). Каждая задача ведёт в свой раздел стола. Названия — из данных
 * сервера; нет названия — код, выдумывать не будем.
 */
import type { NcGroup } from '@/entities/incident'
import type { CorrectiveActionList, IncidentSummary, ProcessSummary, ProcessVersionSummary } from '@/shared/api/generated/model'

/** Раздел стола, куда ведёт задача. */
export type InboxTab = 'investigation' | 'actions' | 'process'

export interface InboxTask {
  id: string
  kind: 'investigation' | 'recurring' | 'actions' | 'version'
  /** Тон: danger — сломано/просрочено, attention — ждёт решения, info — к сведению. */
  tone: 'danger' | 'attention' | 'info'
  tab: InboxTab
  query?: Record<string, string>
  /** Данные для текста — вид задачи решает, какие ключи подставить. */
  data: Record<string, string | number>
}

export interface InboxInput {
  incidents: readonly IncidentSummary[]
  groups: readonly NcGroup[]
  actions: CorrectiveActionList | null
  versions: readonly { process: ProcessSummary; version: ProcessVersionSummary }[]
}

export function buildTasks(input: InboxInput): InboxTask[] {
  const out: InboxTask[] = []
  for (const i of input.incidents.filter((x) => x.status === 'open')) {
    out.push({
      id: `inc-${i.incident_id}`,
      kind: 'investigation',
      tone: (i.counts?.confirmed ?? 0) > 0 ? 'danger' : 'attention',
      tab: 'investigation',
      query: { incident: i.incident_id },
      data: {
        label: i.label,
        factor: i.common_factor?.label || i.common_factor?.value || '',
        stage: i.stage ?? '',
        size: i.size,
        initial: i.initial_size,
        next: i.next_step ?? '',
      },
    })
  }
  for (const r of input.actions?.recurring ?? []) {
    if (r.with_action) continue
    const g = input.groups.find((x) => x.defect_type === r.defect_type && x.operation === r.step_key)
    out.push({
      id: `rec-${r.defect_type}-${r.step_key}`,
      kind: 'recurring',
      tone: 'danger',
      tab: 'investigation',
      query: g?.incident_id ? { incident: g.incident_id } : undefined,
      data: { defect: g?.defect_type_label || r.defect_type, step: g?.operation_label || r.step_key, count: r.count },
    })
  }
  const s = input.actions?.summary
  if (s && (s.overdue || s.ineffective || s.evaluation_due)) {
    out.push({
      id: 'actions',
      kind: 'actions',
      tone: s.overdue || s.ineffective ? 'danger' : 'attention',
      tab: 'actions',
      data: { overdue: s.overdue, ineffective: s.ineffective, evaluation: s.evaluation_due },
    })
  }
  for (const { process, version } of input.versions) {
    if (version.status !== 'draft' && version.status !== 'on_approval') continue
    out.push({
      id: `ver-${version.version_id}`,
      kind: 'version',
      tone: version.status === 'on_approval' ? 'attention' : 'info',
      tab: 'process',
      data: { process: process.name, label: version.label, status: version.status },
    })
  }
  const rank = { danger: 0, attention: 1, info: 2 } as const
  return out.sort((a, b) => rank[a.tone] - rank[b.tone])
}
