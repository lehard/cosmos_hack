/**
 * «Надёжность данных» — первый раздел стола администратора (разбор роли
 * администратора): можно ли сейчас доверять данным и внешним системам, на
 * которых люди принимают решения. Три контура — источники, интеграции, VisionQC —
 * из уже приходящих ответов (ingest.source.list, ops.integration.list,
 * vision.analyzer.list, vision.escape.list); ничего не досчитывается за сервер,
 * только сводка и «что требует внимания» со ссылкой в свой раздел.
 */
import type { AdaptationEscape, AnalyzerSummary, IntegrationEntry, SourceView } from '@/shared/api/generated/model'

export type Tone = 'ok' | 'warn' | 'danger'

/** Что требует внимания — строка со ссылкой в раздел стола (и запись справа, если есть). */
export interface Attention {
  key: string
  area: 'sources' | 'integrations' | 'vision'
  tone: Exclude<Tone, 'ok'>
  /** Ключ текста и параметры. */
  text: string
  params: Record<string, string | number>
  tab: string
  open?: string
}

export interface AreaSummary {
  area: Attention['area']
  ok: number
  total: number
  tone: Tone
}

const worst = (list: readonly { tone: Tone }[]): Tone => (list.some((x) => x.tone === 'danger') ? 'danger' : list.some((x) => x.tone === 'warn') ? 'warn' : 'ok')

/** Источник: подозрение на потерю, неизвестный ключ, пропуски последовательности, карантин. */
export function sourceIssues(sources: readonly SourceView[]): Attention[] {
  const out: Attention[] = []
  for (const s of sources) {
    if (s.state === 'disabled') continue
    if (s.state === 'unknown_key') out.push({ key: `s:${s.source_id}:key`, area: 'sources', tone: 'danger', text: 'dataReliability.issue.unknownKey', params: { source: s.source_id }, tab: 'sources' })
    else if (s.state === 'loss_suspected' || s.gap_count > 0)
      out.push({ key: `s:${s.source_id}:gap`, area: 'sources', tone: 'danger', text: 'dataReliability.issue.gaps', params: { source: s.source_id, n: s.gap_count }, tab: 'sources' })
    if (s.quarantined > 0) out.push({ key: `s:${s.source_id}:q`, area: 'sources', tone: 'warn', text: 'dataReliability.issue.quarantined', params: { source: s.source_id, n: s.quarantined }, tab: 'quarantine' })
  }
  return out
}

/** Интеграция (включённая): недоступна или деградировала по последней проверке, очередь или карантин исходящих. */
export function integrationIssues(items: readonly IntegrationEntry[], name: (system: string) => string): Attention[] {
  const out: Attention[] = []
  for (const i of items) {
    if (i.state === 'disabled') continue
    const open = `integration:${i.system}`
    const r = i.last_check?.result
    if (r === 'unreachable' || r === 'degraded')
      out.push({ key: `i:${i.system}:check`, area: 'integrations', tone: r === 'unreachable' ? 'danger' : 'warn', text: `dataReliability.issue.${r}`, params: { system: name(i.system), n: i.queued ?? 0 }, tab: 'integrations', open })
    else if ((i.queued ?? 0) > 0) out.push({ key: `i:${i.system}:queue`, area: 'integrations', tone: 'warn', text: 'dataReliability.issue.queued', params: { system: name(i.system), n: i.queued ?? 0 }, tab: 'integrations', open })
    if ((i.quarantined ?? 0) > 0) out.push({ key: `i:${i.system}:q`, area: 'integrations', tone: 'warn', text: 'dataReliability.issue.outQuarantine', params: { system: name(i.system), n: i.quarantined ?? 0 }, tab: 'integrations', open })
  }
  return out
}

/** VisionQC: приостановленный паспорт; пропуски брака с изделиями на перепроверку. */
export function visionIssues(analyzers: readonly AnalyzerSummary[], escapes: readonly AdaptationEscape[]): Attention[] {
  const out: Attention[] = []
  for (const a of analyzers)
    if (a.status === 'suspended')
      out.push({ key: `v:${a.analyzer_id}`, area: 'vision', tone: 'danger', text: 'dataReliability.issue.suspended', params: { analyzer: a.title }, tab: 'vision-adaptation', ...(a.passport_id ? { open: `passport:${a.passport_id}` } : {}) })
  const recheck = new Set(escapes.flatMap((e) => e.recheck.map((r) => r.item_id)))
  if (recheck.size) out.push({ key: 'v:recheck', area: 'vision', tone: 'warn', text: 'dataReliability.issue.recheck', params: { n: recheck.size }, tab: 'vision-adaptation' })
  return out
}

/** Сводка контура: сколько в норме из скольких и худший тон. */
export function areaSummary(area: Attention['area'], total: number, issues: readonly Attention[]): AreaSummary {
  const own = issues.filter((x) => x.area === area)
  const bad = new Set(own.map((x) => x.key.split(':').slice(0, 2).join(':'))).size
  return { area, ok: Math.max(0, total - bad), total, tone: worst(own) }
}

/** Общий статус: есть опасное — «работа с ограничениями», только предупреждения — «есть замечания», иначе — доверять можно. */
export const overallTone = (issues: readonly Attention[]): Tone => worst(issues)
