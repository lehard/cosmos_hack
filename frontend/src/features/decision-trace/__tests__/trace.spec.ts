// «Как машина пришла к выводу» (Ф1 SHOW-IS2): дорожка решения по наблюдению и по
// карточке НС — качество против порога, ответ и уверенность, правило карты
// реакций, уровень доверия, «сделала сама / решает человек».
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { AnalyzerPassport, NCCard, ObservationAccount, ReactionRule } from '@/shared/api/generated/model'
import { i18n } from '@/shared/i18n'
import { FORBIDDEN_ALWAYS, qualityBelow, traceFromNc, traceFromObservation } from '../model/trace'
import DecisionTrace from '../ui/DecisionTrace.vue'

const obs = (over: Partial<ObservationAccount> = {}): ObservationAccount => ({
  event_id: 'E-OBS-3',
  item_id: 'Ф-003',
  occurred_at: '2026-09-21T08:30:00Z',
  point: 'КТ-3',
  outcome: 'defect_indicated',
  quality_bp: 9000,
  confidence_bp: 9300,
  level_then: 3,
  passport_id: 'AP-KT3-WELD-1',
  status_then: 'active',
  status_now: 'active',
  allowed_auto_actions: ['quality.signal.raised', 'decision.containment.applied'],
  suspicious: false,
  reasons: ['Паспорт AP-KT3-WELD-1 действовал: уровень 3'],
  stages: [{ name: 'detect', version: 'vqc-weld 2.3.1', confidence_bp: 9300 }],
  versions: { analyzer_version: 'vqc-weld 2.3.1', recipe_ref: 'kt3-weld@1' },
  ...over,
})
const passport = { passport_id: 'AP-KT3-WELD-1', title: 'Визуальный контроль сварного шва (КТ-3)', trust_level: 3, provenance: 'genesis', monitor: { window: 20, quality_min_bp: 6000, seen: 3, recent_quality_bp: [] } } as unknown as AnalyzerPassport
const rule: ReactionRule = { rule_id: 'R-07', title: 'Прожог или подрез шва — карантин и черновик карточки', trigger: 'признак W-BURNTHRU', outcome: 'isolate', automation_mode: 2, action_class: 'containment' as never, owner: 'technologist' }

const card = (over: Partial<NCCard> = {}) =>
  ({
    status: 'draft',
    origin: 'signal',
    rule_rev: 'flange-reactions@1',
    containment: [{ key: 'E-C1', by: 'rule', level: 'item_hold', reason: 'R-07', rule_id: 'R-07', basis_gone: false }],
    system_analysis: {
      alternatives: [],
      missing_information: [],
      why: ['Прожог в зоне У2, уверенность 0,93'],
      versions: [{ version: 1, event_id: 'E-A1', outcome: 'isolate', rule_id: 'R-07', rule_rev: 'flange-reactions@1', automation_mode: 2, causes: [], recorded_at: '2026-09-21T08:31:00Z', revised_due_to: null }],
    },
    evidence: {
      similar_count: 0,
      zone_history: [],
      signals: [
        {
          signal_id: 'SIG-3',
          basis_kind: 'inspection_result',
          defect_type_code: 'W-BURNTHRU',
          defect_type_known: true,
          defect_type_label: 'Прожог',
          severity: 'major',
          analyzer_confidence_bp: 9300,
          observation_quality_bp: 9000,
          evidence_refs: [],
          stages: [],
          zone_id: 'U2',
          zone_label: 'Шов У2',
          record: { event_id: 'E-OBS-3', event_type: 'inspection.result.recorded', kind: 'fact', occurred_at: '2026-09-21T08:30:00Z', summary: 'КТ-3' },
        },
      ],
    },
    ...over,
  }) as unknown as NCCard

describe('дорожка решения — правила', () => {
  it('Ф-003 (К2): прожог, правило R-07 редакции карты, уровень 3 — блок сама, подтверждает контролёр', () => {
    const tr = traceFromNc(card(), { observation: obs(), passport, rule })
    expect(tr.frame.zone).toBe('Шов У2')
    expect(tr.quality).toEqual({ value: 9000, threshold: 6000, thresholdFrom: 'passport' })
    expect(tr.answer).toMatchObject({ outcome: 'defect_indicated', defect: 'Прожог', severity: 'major', confidence: { value: 9300 } })
    expect(tr.rule).toMatchObject({ id: 'R-07', rev: 'flange-reactions@1', title: rule.title, automationMode: 2 })
    expect(tr.trust).toMatchObject({ level: 3, passportId: 'AP-KT3-WELD-1', provenance: 'genesis' })
    expect(tr.system.map((a) => a.key)).toEqual(['recorded', 'containment', 'draftNc'])
    expect(tr.system[1]!.params).toEqual({ level: 'item_hold', rule: 'R-07' })
    expect(tr.human.map((a) => a.key)).toEqual(['confirmSignal', ...FORBIDDEN_ALWAYS])
  })

  it('Ф-002 (К1): «признаков нет» при качестве 0,34 ниже порога — «оценка невозможна», доп. проверка — человек', () => {
    const tr = traceFromObservation(obs({ outcome: 'no_defect_indicated', quality_bp: 3400, confidence_bp: 8800 }), passport)
    expect(qualityBelow(tr.quality)).toBe(true)
    expect(tr.system.map((a) => a.key)).toEqual(['recorded', 'unableToAssess'])
    expect(tr.human[0]!.key).toBe('recheck')
    expect(tr.rule.id).toBeUndefined()
  })

  it('порог карты контроля важнее порога паспорта; без порога — просто значение', () => {
    expect(traceFromObservation(obs(), passport, { recipeMinBp: 5000 }).quality).toEqual({ value: 9000, threshold: 5000, thresholdFrom: 'recipe' })
    expect(traceFromObservation(obs(), null).quality).toEqual({ value: 9000 })
    expect(traceFromObservation(obs({ quality_bp: undefined }), null).quality).toBeNull()
  })

  it('уровень 0 — только запись; «признаков нет» без уровня 4 — «годно» сама не выдала', () => {
    expect(traceFromObservation(obs({ level_then: 0, outcome: 'defect_indicated' })).system.map((a) => a.key)).toEqual(['recorded'])
    expect(traceFromObservation(obs({ level_then: 3, outcome: 'no_defect_indicated' })).system.map((a) => a.key)).toEqual(['recorded', 'noAutoPass'])
  })
})

describe('дорожка решения — вид', () => {
  const mountTrace = (trace = traceFromNc(card(), { observation: obs(), passport, rule })) => mount(DecisionTrace, { props: { trace }, global: { plugins: [i18n] } })

  it('цепочка из пяти шагов, шкалы с порогом, правило с редакцией, уровень доверия, граница автоматизации', () => {
    const w = mountTrace()
    expect(w.findAll('.lane > .step')).toHaveLength(5)
    expect(w.find('[data-testid="trace-quality"]').text()).toBe('0,90')
    expect(w.find('[data-step="quality"]').text()).toContain('порог допуска анализатора 0,60')
    expect(w.find('[data-testid="trace-confidence"]').text()).toBe('0,93')
    expect(w.find('[data-step="answer"]').text()).toContain('Уверенность анализатора — не вероятность брака')
    expect(w.find('[data-testid="trace-rule"]').text()).toBe('R-07 — Прожог или подрез шва — карантин и черновик карточки')
    expect(w.find('[data-step="rule"]').text()).toContain('редакция flange-reactions@1 · режим автоматизации 2')
    expect(w.find('[data-testid="trace-trust"]').text()).toBe('Уровень 3 из 4 — дополнительно — блок и изоляция')
    expect(w.find('[data-testid="trace-genesis"]').exists()).toBe(true)
    expect(w.find('[data-testid="trace-system"]').text()).toContain('Поставила сдерживание «Блок изделия» по правилу R-07')
    expect(w.find('[data-testid="trace-system"]').text()).toContain('Завела черновик несоответствия')
    const human = w.find('[data-testid="trace-human"]').text()
    expect(human).toContain('Подтвердить или отклонить сигнал — контролёр')
    expect(human).toContain('Снятие блока')
  })

  it('качество ниже порога — отмечено; правило не передано — так и сказано', () => {
    const w = mountTrace(traceFromObservation(obs({ outcome: 'no_defect_indicated', quality_bp: 3400 }), passport))
    expect(w.find('[data-step="quality"]').attributes('data-below')).toBe('true')
    expect(w.find('[data-testid="trace-quality-low"]').text()).toContain('оценка невозможна')
    expect(w.find('[data-step="rule"]').text()).toContain('Номер сработавшего правила сервер не передал')
    expect(w.find('[data-testid="trace-answer"]').text()).toContain('Признаки не обнаружены')
  })
})
