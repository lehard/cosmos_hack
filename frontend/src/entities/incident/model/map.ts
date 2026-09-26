/**
 * Ответы операций analysis (сгенерированные типы contracts/openapi.yaml) →
 * входные данные экранов разбора. Отсутствующие необязательные поля в ответе не
 * приходят — здесь они становятся явным null (соглашение «Неизвестность»),
 * смысл полей не меняется (NFR-UI-4).
 */
import type {
  Circumstances,
  CommonFactors,
  Hypotheses,
  JournalRecordRef as ApiRecordRef,
  NcGroup as ApiNcGroup,
  RiskScope,
} from '@/shared/api/generated/model'
import type {
  CircumstancesModel,
  CommonFactorsModel,
  HypothesesModel,
  JournalRecordRef,
  MissingInformation,
  NcGroup,
  RiskScopeModel,
} from './types'

const ref = (r: ApiRecordRef): JournalRecordRef => ({
  event_id: r.event_id,
  event_type: r.event_type,
  variant: r.variant ?? null,
  occurred_at: r.occurred_at,
  params: r.params ?? {},
})

/** Разбор обстоятельств (`analysis.circumstances.read`). */
export const toCircumstancesModel = (a: Circumstances): CircumstancesModel => ({
  nc_id: a.nc_id,
  operation: a.operation ?? null,
  window: a.window ?? null,
  records: a.records.map((r) => ({
    ...ref(r),
    lane: r.lane,
    ended_at: r.ended_at ?? null,
    journal_seq: r.journal_seq ?? null,
    evidence_refs: r.evidence_refs ?? [],
    related_event_ids: r.related_event_ids ?? [],
    source_kind: r.source_kind ?? null,
  })),
  missing_information: a.missing_information as MissingInformation[],
  conclusion_is_categorical: a.conclusion_is_categorical,
})

/** Общие факторы (`analysis.common_factors.read`). */
export const toCommonFactorsModel = (a: CommonFactors): CommonFactorsModel => ({
  group_key: a.group_key,
  group_label: a.group_label,
  nc_count: a.nc_count,
  rows: a.rows.map((r) => ({ factor: r.factor, value: r.value ?? null, matches: r.matches, distinct_values: r.distinct_values })),
})

/** Гипотезы и похожие случаи (`analysis.hypothesis.list`). */
export const toHypothesesModel = (a: Hypotheses): HypothesesModel => ({
  nc_id: a.nc_id,
  version: a.version,
  hypotheses: a.hypotheses.map((h) => ({
    hypothesis_id: h.hypothesis_id,
    category: h.category,
    branch: h.branch ?? null,
    statement: h.statement ?? null,
    status: h.status,
    confidence_bp: h.confidence_bp ?? null,
    supporting: h.supporting.map(ref),
    contradicting: h.contradicting.map(ref),
    measurement_hint: h.measurement_hint ?? null,
  })),
  missing_information: a.missing_information as MissingInformation[],
  conclusion_is_categorical: a.conclusion_is_categorical,
  similar_cases: a.similar_cases.map((c) => ({
    nc_id: c.nc_id,
    number: c.number,
    cause_category: c.cause_category ?? null,
    cause_confirmed: c.cause_confirmed,
    measure: c.measure ?? null,
    result: c.result ?? null,
  })),
})

/** Область риска (`analysis.risk_scope.read`). */
export const toRiskScopeModel = (a: RiskScope): RiskScopeModel => ({
  incident_id: a.incident_id,
  incident_label: a.incident_label,
  common_factor: a.common_factor ?? null,
  window: a.window ?? null,
  last_known_good: a.last_known_good ?? null,
  versions: a.versions.map((v) => ({
    scope_version: v.scope_version,
    change: v.change,
    size: v.size,
    recorded_at: v.recorded_at,
    author: v.author ?? null,
    reason: v.reason ? { code: v.reason.code ?? null, text: v.reason.text } : null,
    evidence_event_ids: v.evidence_event_ids,
    breakdown: v.breakdown,
  })),
  items: a.items,
  shipped_to_partners: a.shipped_to_partners ?? 0,
})

/**
 * Группа несоответствий (`analysis.group.list`). `nc_ids` — предложенное
 * совместимое поле группы (несоответствия группы для разбора обстоятельств);
 * пока сервер его не отдаёт — пустой список.
 */
export const toNcGroup = (a: ApiNcGroup): NcGroup => ({
  group_key: a.group_key,
  defect_type: a.defect_type,
  operation: a.operation,
  equipment: a.equipment,
  nc_count: a.nc_count,
  investigation: a.investigation,
  last_found_at: a.last_found_at,
  nc_ids: (a as ApiNcGroup & { nc_ids?: string[] }).nc_ids ?? [],
})
