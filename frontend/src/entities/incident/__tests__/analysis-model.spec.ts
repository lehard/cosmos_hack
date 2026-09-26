// Правила экранов разбора (FR-58, FR-61, FR-135, FR-153): подписи записей,
// общая шкала, этапы «до / во время / после», входной дефект, общие факторы,
// гипотезы, версии области риска, фокус разбора.
import { createPinia, setActivePinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import {
  circumstancesState,
  describeRecord,
  factorsState,
  hypothesesState,
  isCommonToAll,
  isIncomingDefect,
  linkedIds,
  makeTimeScale,
  phaseSummary,
  scopeIssues,
  scopeReduction,
  scopeState,
  sortFactors,
  sortGroups,
  toCircumstancesModel,
  toRiskScopeModel,
  sortHypotheses,
  stackLabels,
  useAnalysisFocusStore,
} from '@/entities/incident'
import { i18n } from '@/shared/i18n'
import { incomingCircumstances, ncGroups, weldCircumstances, weldFactors, weldHypotheses, weldScope } from './fixtures'

const ids = (rs: { event_id: string }[]) => rs.map((r) => r.event_id)

describe('подпись записи журнала', () => {
  it('тип + уточнение → ключ текста и смысловой тон', () => {
    const r = describeRecord({ event_id: 'x', event_type: 'equipment.deviation.detected', variant: 'out_of_setpoint', occurred_at: '2026-09-23T08:18:00.000Z', params: { parameter: 'ток', value: '212 А', setpoint: '180 А' } })
    expect(r.key).toBe('timeline.equipment.parameterOutOfRange')
    expect(r.tone).toBe('deviation')
    expect(i18n.global.t(r.key!, r.params)).toBe('Вне уставки: ток 212 А при уставке 180 А')
  })

  it('исход контроля различает норму, находку и «оценка невозможна»', () => {
    const base = { event_id: 'x', event_type: 'inspection.result.recorded', occurred_at: '2026-09-23T08:00:00.000Z' }
    expect(describeRecord({ ...base, variant: 'no_defect_indicated' }).tone).toBe('normal')
    expect(describeRecord({ ...base, variant: 'defect_indicated' }).tone).toBe('finding')
    expect(describeRecord({ ...base, variant: 'unable_to_assess' }).tone).toBe('unable')
  })

  it('неизвестный тип не подменяется похожим', () => {
    const r = describeRecord({ event_id: 'x', event_type: 'zz.unknown.type', variant: 'v', occurred_at: '2026-09-23T08:00:00.000Z' })
    expect(r.key).toBeNull()
    expect(r.tone).toBe('unknown')
    expect(r.eventType).toBe('zz.unknown.type/v')
  })
})

describe('общая шкала времени', () => {
  it('все моменты внутри шкалы, порядок сохраняется, деления внутри', () => {
    const s = makeTimeScale(['2026-09-23T07:55:00.000Z', '2026-09-23T08:52:00.000Z', '2026-09-23T08:10:00.000Z'])
    const a = s.pos('2026-09-23T07:55:00.000Z')
    const b = s.pos('2026-09-23T08:10:00.000Z')
    const c = s.pos('2026-09-23T08:52:00.000Z')
    expect(a).toBeGreaterThan(0)
    expect(c).toBeLessThan(100)
    expect(a).toBeLessThan(b)
    expect(b).toBeLessThan(c)
    expect(s.ticks.length).toBeGreaterThan(1)
    expect(s.ticks.length).toBeLessThanOrEqual(8)
    for (const t of s.ticks) expect(t >= s.start && t <= s.end).toBe(true)
  })

  it('одна точка — шкала не вырождается', () => {
    const s = makeTimeScale(['2026-09-23T08:00:00.000Z'])
    expect(s.end - s.start).toBeGreaterThan(0)
    expect(s.pos('2026-09-23T08:00:00.000Z')).toBeCloseTo(50, 5)
  })

  it('подписи соседних отметок раскладываются по ярусам', () => {
    expect(stackLabels([0, 5, 50, 52], [20, 20, 20, 20])).toEqual([0, 1, 0, 1])
  })
})

describe('разбор обстоятельств', () => {
  it('до сварки чисто → во время две нештатности → после признак дефекта (FR-149)', () => {
    const s = phaseSummary(weldCircumstances())
    expect(ids(s.before)).toEqual(['e-kt2-ok'])
    expect(ids(s.during)).toEqual(['e-current', 'e-override'])
    expect(ids(s.after)).toEqual(['e-kt3-defect'])
  })

  it('входной дефект: находка до операции — не связывается с исполнителем (FR-58)', () => {
    expect(isIncomingDefect(weldCircumstances())).toBe(false)
    expect(isIncomingDefect(incomingCircumstances())).toBe(true)
  })

  it('выбор записи подсвечивает её связанные и ссылающиеся на неё', () => {
    const m = weldCircumstances()
    expect([...linkedIds(m.records, 'e-kt3-defect')].sort()).toEqual(['e-current', 'e-kt3-defect', 'e-override'])
    expect([...linkedIds(m.records, 'e-override')].sort()).toEqual(['e-kt3-defect', 'e-manual', 'e-override'])
    expect(linkedIds(m.records, null).size).toBe(0)
  })

  it('четыре состояния экрана', () => {
    expect(circumstancesState(weldCircumstances())).toBe('defect_indication')
    expect(circumstancesState({ ...weldCircumstances(), window: null })).toBe('unable_to_assess')
    const bad = weldCircumstances()
    bad.records[0]!.occurred_at = 'вчера'
    expect(circumstancesState(bad)).toBe('input_error')
    const clean = weldCircumstances()
    clean.records = clean.records.filter((r) => r.event_id !== 'e-kt3-defect')
    expect(circumstancesState(clean)).toBe('normal')
  })
})

describe('общие факторы (FR-135)', () => {
  it('фактор, общий для всей группы, — первым', () => {
    const rows = sortFactors(weldFactors().rows)
    expect(rows.map((r) => r.factor)).toEqual(['machine', 'program', 'fixture', 'material_batch', 'performer', 'tool'])
    expect(isCommonToAll(rows[0]!, 3)).toBe(true)
    expect(isCommonToAll(rows[2]!, 3)).toBe(false)
  })

  it('состояния: совпадений больше N — ошибка входа; ни одного известного — оценка невозможна', () => {
    expect(factorsState(weldFactors())).toBe('normal')
    const bad = weldFactors()
    bad.rows[0]!.matches = 5
    expect(factorsState(bad)).toBe('input_error')
    const blind = weldFactors()
    blind.rows = blind.rows.map((r) => ({ ...r, value: null }))
    expect(factorsState(blind)).toBe('unable_to_assess')
  })
})

describe('гипотезы (FR-59)', () => {
  it('открытые по уверенности вывода, отклонённые — в конце', () => {
    expect(sortHypotheses(weldHypotheses().hypotheses).map((h) => h.hypothesis_id)).toEqual(['h-equipment', 'h-performer', 'h-incoming'])
  })

  it('при недостатке сведений — «оценка невозможна», без категоричного вывода', () => {
    expect(hypothesesState(weldHypotheses())).toBe('unable_to_assess')
    expect(hypothesesState({ ...weldHypotheses(), conclusion_is_categorical: true })).toBe('normal')
    const bad = weldHypotheses()
    bad.hypotheses[0]!.confidence_bp = 12_000
    expect(hypothesesState(bad)).toBe('input_error')
  })
})

describe('область риска (FR-61)', () => {
  it('34 → 13 → 6: сокращение и основания на месте', () => {
    const m = weldScope()
    expect(m.versions.map((v) => v.size)).toEqual([34, 13, 6])
    expect(scopeIssues(m)).toEqual([])
    const r = scopeReduction(m)!
    expect([r.from, r.to]).toEqual([34, 6])
    expect(r.ratio).toBeCloseTo(28 / 34, 5)
  })

  it('ни одно изделие не выходит из области без основания', () => {
    const m = weldScope()
    m.versions[2]!.reason = null
    m.versions[2]!.evidence_event_ids = []
    expect(scopeIssues(m)).toEqual([{ kind: 'basis_missing', version: 3 }])
    expect(scopeState(m)).toBe('input_error')
  })

  it('разбивка должна сходиться с размером; «сужение» не увеличивает область', () => {
    const m = weldScope()
    m.versions[1]!.breakdown.shipped = 5
    m.versions[2]!.size = 40
    m.versions[2]!.breakdown.in_production = 39
    expect(scopeIssues(m).map((x) => x.kind)).toEqual(['breakdown_mismatch', 'narrowed_grew'])
  })

  it('подтверждено у изделия — «признак дефекта»; неизвестно последнее нормальное — «оценка невозможна»', () => {
    expect(scopeState(weldScope())).toBe('defect_indication')
    const m = weldScope()
    m.items = m.items.filter((i) => i.known !== 'confirmed')
    expect(scopeState(m)).toBe('normal')
    expect(scopeState({ ...m, last_known_good: null })).toBe('unable_to_assess')
  })
})

describe('группы и фокус разбора', () => {
  it('группы: больше несоответствий — выше, при равенстве — свежее', () => {
    expect(sortGroups(ncGroups()).map((g) => g.equipment)).toEqual(['ИС-3', 'ЧПУ-7', 'ИС-2'])
  })

  it('выбор группы сбрасывает выбор внутри прежней; вход из фактора запоминается', () => {
    setActivePinia(createPinia())
    const f = useAnalysisFocusStore()
    f.eventId = 'e-1'
    f.enterFromFactor(weldFactors().rows[5]!, 'narrow_scope')
    expect(f.factor?.intent).toBe('narrow_scope')
    f.selectGroup('g-2')
    expect([f.groupKey, f.eventId, f.factor]).toEqual(['g-2', null, null])
  })
})

describe('ответы API → данные экранов', () => {
  it('отсутствующие необязательные поля — явный null', () => {
    const m = toCircumstancesModel({
      nc_id: 'NC-1',
      basis_seq: 5,
      conclusion_is_categorical: true,
      missing_information: [],
      records: [{ event_id: 'e', lane: 'item', event_type: 'inspection.result.recorded', occurred_at: '2026-09-23T08:00:00.000Z' }],
    })
    expect(m.operation).toBeNull()
    expect(m.window).toBeNull()
    expect(m.records[0]).toMatchObject({ variant: null, ended_at: null, journal_seq: null, evidence_refs: [], related_event_ids: [], source_kind: null })
    const s = toRiskScopeModel({ incident_id: 'I', incident_label: 'И', basis_seq: 1, items: [], versions: [] })
    expect([s.common_factor, s.window, s.last_known_good]).toEqual([null, null, null])
  })
})
