/**
 * «Как машина пришла к выводу» (Ф1 сценария показа SHOW-IS2; кейс §1.4, §2.2,
 * §4.5, §5.4; FR-48, FR-50, FR-98; AD-29) — дорожка решения одной цепочкой:
 * точка и зона → качество наблюдения против порога → ответ анализатора и
 * уверенность → правило карты реакций (id, редакция) → уровень доверия
 * анализатора → что система сделала сама / что оставлено человеку.
 *
 * Только чистые функции над ответами API: наблюдение (`vision.observation.read`),
 * паспорт анализатора (`vision.passport.read`), карточка НС (`NCCard`: сигнал,
 * анализ системы, сдерживание) и правило карты реакций (`quality.reaction_map.read`).
 * Чего источник не передал — поле пустое, экран говорит «не передано», а не
 * додумывает (кейс §5.4).
 */
import type { AnalyzerPassport, NCCard, ObservationAccount, ReactionRule } from '@/shared/api/generated/model'

/** Шкала 0…10 000 б. п. с порогом. */
export interface TraceScale {
  /** Значение, б. п. */
  value: number
  /** Порог, б. п.; нет — источник порога не передал. */
  threshold?: number
  /** Откуда порог: карта контроля, паспорт анализатора (контроль дрейфа) или правило карты реакций. */
  thresholdFrom?: 'recipe' | 'passport' | 'rule'
}

/** Шаг анализатора: ступень, версия, уверенность. */
export interface TraceStage {
  name: string
  version: string
  confidenceBp?: number
  output?: string
}

/** Действие для текста: ключ `decisionTrace.acts.‹key›` и параметры. */
export interface TraceAct {
  key: string
  params?: Record<string, string | number>
}

/** Дорожка решения. */
export interface DecisionTrace {
  /** Где смотрели: точка контроля, зона изделия, время наблюдения (сам кадр — в карточке НС). */
  frame: { point?: string; zone?: string; at?: string }
  /** Качество наблюдения против порога; null — источник не передал. */
  quality: TraceScale | null
  /** Ответ анализатора: исход, вид дефекта, тяжесть, уверенность (≠ вероятность брака). */
  answer: { outcome?: string; defect?: string; severity?: string; confidence: TraceScale | null; stages: TraceStage[]; analyzerVersion?: string }
  /** Правило карты реакций; нет id — сервер не передал, какое правило сработало. */
  rule: { id?: string; rev?: string; title?: string; trigger?: string; automationMode?: number; reasons: string[] }
  /** Уровень доверия анализатора 0–4 на момент наблюдения и паспорт допуска. */
  trust: { level?: number; passportId?: string; title?: string; statusThen?: string; provenance?: string }
  /** Что система сделала сама. */
  system: TraceAct[]
  /** Что оставлено человеку. */
  human: TraceAct[]
}

/**
 * Всегда только человек — при любом уровне доверия (contracts/analyzer-trust-levels.yaml,
 * `forbidden_always`; FR-98, AD-27): приёмка на закрывающей точке, решение по изделию,
 * снятие блока, сужение области риска, выпуск. Ключи текстов — `decisionTrace.human.*`.
 */
export const FORBIDDEN_ALWAYS: readonly string[] = ['presentationResolved', 'dispositionSet', 'containmentReleased', 'scopeNarrowed', 'releaseRecorded']

/** Исход наблюдения анализатора (контракт наблюдения) → ключ текста. */
export const OUTCOME_TEXT: Record<string, string> = {
  defect_indicated: 'inspection.outcome.defectFound',
  no_defect_indicated: 'inspection.outcome.noDefectFound',
  unable_to_assess: 'inspection.outcome.unableToAssess',
}

/** Качество ниже порога — «признаков нет» не значит «годно» (кейс §4.5). */
export const qualityBelow = (q: TraceScale | null | undefined): boolean => !!q && q.threshold !== undefined && q.value < q.threshold

/** Уверенность ниже порога правила. */
export const confidenceBelow = (c: TraceScale | null | undefined): boolean => !!c && c.threshold !== undefined && c.value < c.threshold

/** Доля шкалы 0…100 для полосы. */
export const scalePercent = (bp: number | undefined): number => (bp === undefined ? 0 : Math.max(0, Math.min(100, bp / 100)))

/**
 * Порог качества: карта контроля (если сервер его передаст), иначе порог
 * контроля дрейфа из паспорта анализатора («кадр не как при допуске»).
 */
function qualityScale(value: number | undefined, recipeMin: number | undefined, passport: AnalyzerPassport | null | undefined): TraceScale | null {
  if (value === undefined) return null
  if (recipeMin !== undefined && recipeMin > 0) return { value, threshold: recipeMin, thresholdFrom: 'recipe' }
  const m = passport?.monitor?.quality_min_bp
  if (m !== undefined && m > 0) return { value, threshold: m, thresholdFrom: 'passport' }
  return { value }
}

/** Порог уверенности правила карты реакций. */
function confidenceScale(value: number | undefined, rule: ReactionRule | null | undefined): TraceScale | null {
  if (value === undefined) return null
  return rule?.threshold_bp !== undefined ? { value, threshold: rule.threshold_bp, thresholdFrom: 'rule' } : { value }
}

/**
 * Что система могла сделать сама по уровню доверия (contracts/analyzer-trust-levels.yaml,
 * уровни накопительные) — когда сервер не сообщил, что именно сделано.
 */
function actsByLevel(level: number | undefined, outcome: string | undefined, lowQuality: boolean, suspicious: boolean): TraceAct[] {
  const acts: TraceAct[] = [{ key: 'recorded' }]
  if (level === undefined) return acts
  if (lowQuality || outcome === 'unable_to_assess') {
    acts.push({ key: 'unableToAssess' })
    return acts
  }
  if (suspicious && level >= 2) acts.push({ key: 'suspicious' })
  else if (outcome === 'defect_indicated' && level >= 1) acts.push({ key: 'recommend' })
  if (outcome === 'no_defect_indicated' && level < 4) acts.push({ key: 'noAutoPass' })
  return acts
}

/** Кто решает дальше: контекст плюс всегдашние решения человека. */
function humanActs(extra: TraceAct[]): TraceAct[] {
  return [...extra, ...FORBIDDEN_ALWAYS.map((key) => ({ key }))]
}

/**
 * Дорожка по наблюдению анализатора (паспорт изделия): `vision.observation.read`
 * и паспорт допуска на момент наблюдения.
 * @param recipeMinBp — порог качества карты контроля, если известен
 */
export function traceFromObservation(
  obs: ObservationAccount,
  passport?: AnalyzerPassport | null,
  opts: { zone?: string; recipeMinBp?: number } = {},
): DecisionTrace {
  const quality = qualityScale(obs.quality_bp, opts.recipeMinBp, passport)
  const low = qualityBelow(quality)
  const extra: TraceAct[] = []
  if (obs.outcome === 'defect_indicated') extra.push({ key: 'confirmSignal' })
  if (low || obs.outcome === 'unable_to_assess' || obs.suspicious) extra.push({ key: 'recheck' })
  return {
    frame: { point: obs.point, zone: opts.zone, at: obs.occurred_at },
    quality,
    answer: {
      outcome: obs.outcome,
      confidence: confidenceScale(obs.confidence_bp, null),
      stages: (obs.stages ?? []).map((s) => ({ name: s.name, version: s.version, confidenceBp: s.confidence_bp, output: s.output })),
      analyzerVersion: obs.versions.analyzer_version,
    },
    rule: { reasons: obs.reasons },
    trust: { level: obs.level_then, passportId: obs.passport_id, title: passport?.title, statusThen: obs.status_then, provenance: passport?.provenance },
    system: actsByLevel(obs.level_then, obs.outcome, low, obs.suspicious),
    human: humanActs(extra),
  }
}

/**
 * Дорожка по карточке НС: исходный сигнал анализатора, вывод системы (правило
 * и редакция карты реакций), сдерживание, поставленное правилом, — и то же
 * наблюдение (`vision.observation.read`), если сигнал — результат контроля.
 */
export function traceFromNc(
  card: Pick<NCCard, 'evidence' | 'system_analysis' | 'containment' | 'origin' | 'status' | 'rule_rev'>,
  opts: { observation?: ObservationAccount | null; passport?: AnalyzerPassport | null; rule?: ReactionRule | null; recipeMinBp?: number } = {},
): DecisionTrace {
  const sig = card.evidence.signals[0]
  const obs = opts.observation ?? null
  const last = card.system_analysis.versions.at(-1)
  const quality = qualityScale(sig?.observation_quality_bp ?? obs?.quality_bp, opts.recipeMinBp, opts.passport)
  // Правило «качество ниже карты» (R-02 и подобные) — ниже порога, даже если числа порога нет.
  const low = qualityBelow(quality)
  const confidenceBp = sig?.analyzer_confidence_bp ?? obs?.confidence_bp
  const level = obs?.level_then ?? opts.passport?.trust_level

  // Что сделала система: сдерживание, поставленное правилом, и черновик НС по сигналу.
  const system: TraceAct[] = [{ key: 'recorded' }]
  for (const c of card.containment ?? []) {
    if (c.by === 'rule' && !c.basis_gone) system.push({ key: 'containment', params: { level: c.level, rule: c.rule_id ?? last?.rule_id ?? '' } })
  }
  if (card.origin === 'signal') system.push({ key: 'draftNc' })
  if (last?.outcome === 'manual_review' && !system.some((a) => a.key === 'containment')) system.push({ key: 'manualReview' })
  if (low || obs?.outcome === 'unable_to_assess') system.push({ key: 'unableToAssess' })

  const extra: TraceAct[] = []
  if (card.status === 'draft') extra.push({ key: 'confirmSignal' })
  if (low || last?.outcome === 'manual_review') extra.push({ key: 'recheck' })

  return {
    frame: { point: obs?.point, zone: sig?.zone_label ?? sig?.zone_id, at: sig?.record.occurred_at ?? obs?.occurred_at },
    quality,
    answer: {
      outcome: obs?.outcome,
      defect: sig?.defect_type_label ?? sig?.defect_type_code,
      severity: sig?.severity,
      confidence: confidenceScale(confidenceBp, opts.rule),
      stages: sig?.stages.length
        ? sig.stages.map((s) => ({ name: s.stage, version: s.version, confidenceBp: s.confidence_bp, output: s.output_note }))
        : (obs?.stages ?? []).map((s) => ({ name: s.name, version: s.version, confidenceBp: s.confidence_bp, output: s.output })),
      analyzerVersion: sig?.versions?.analyzer_version ?? obs?.versions.analyzer_version,
    },
    rule: {
      id: last?.rule_id,
      rev: last?.rule_rev ?? card.rule_rev,
      title: opts.rule?.title,
      trigger: opts.rule?.trigger,
      automationMode: last?.automation_mode,
      reasons: [...card.system_analysis.why, ...(obs?.reasons ?? [])],
    },
    trust: { level, passportId: obs?.passport_id ?? opts.passport?.passport_id, title: opts.passport?.title, statusThen: obs?.status_then, provenance: opts.passport?.provenance },
    system,
    human: humanActs(extra),
  }
}

/** Событие наблюдения анализатора, из которого родился сигнал НС (если это результат контроля). */
export function observationEventOf(card: Pick<NCCard, 'evidence'>): string | null {
  const r = card.evidence.signals[0]?.record
  return r && r.event_type === 'inspection.result.recorded' ? r.event_id : null
}
