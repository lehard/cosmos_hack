// Решения контролёра и по несоответствию: что допустимо, что обязательно до
// подписи, какое тело уходит на сервер (FR-52, FR-53, FR-54, FR-66, AD-7, AD-39).
import { describe, expect, it } from 'vitest'
import {
  SUMMARY_MAX,
} from '@/entities/document'
import {
  availableActions,
  buildDecisionRequest,
  conclusionVersions,
  deadlineOf,
  decisionSummary,
  decisionsBeforeRevision,
  draftProblems,
  ncCardState,
  queueSortOf,
  usableConcessions,
  uuidv7,
  type DecisionDraft,
} from '@/entities/nonconformity'
import { concessions, confirmedCard, ncCard } from './fixtures'

const meta = { command_id: '0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b', policy_seq: 40 }
const draft = (over: Partial<DecisionDraft>): DecisionDraft => ({ action: 'confirm_nc', reason: '', ...over })

describe('какие решения показать (FR-52)', () => {
  it('допустимые по состоянию — из to_decide, в порядке кнопок', () => {
    expect(availableActions(ncCard())).toEqual(['confirm_nc', 'reject_signal', 'request_recheck', 'isolate'])
    expect(availableActions(confirmedCard())).toEqual(['disposition'])
  })

  it('неизвестная операция не превращается в кнопку', () => {
    const card = ncCard({ to_decide: { concession_required: false, decisions: ['nonconformity.something.new'] } })
    expect(availableActions(card)).toEqual([])
  })
})

describe('что обязательно до подписи', () => {
  it('отклонение без причины невозможно (FR-52)', () => {
    expect(draftProblems(draft({ action: 'reject_signal' }), [])).toContain('errors.decision.rejectReasonRequired')
    expect(draftProblems(draft({ action: 'reject_signal', reason: '  ' }), [])).toContain('errors.decision.rejectReasonRequired')
    expect(draftProblems(draft({ action: 'reject_signal', reason: 'Блик на кромке' }), [])).toEqual([])
  })

  it('«как есть» без действующего разрешения не подписывается (FR-53)', () => {
    const base = draft({ action: 'disposition', disposition: 'use_as_is', reason: 'Пора 0,3 мм в пределах РО' })
    expect(draftProblems(base, concessions())).toContain('decisions.disposition.selectConcession')
    // Истёкшее разрешение выбрать нельзя.
    expect(draftProblems({ ...base, concession_id: 'CON-2' }, concessions())).toContain('decisions.disposition.selectConcession')
    expect(draftProblems({ ...base, concession_id: 'CON-1' }, concessions())).toEqual([])
  })

  it('разрешения: только действующие, того же вида и с остатком лимита (FR-54)', () => {
    expect(usableConcessions(concessions(), 'use_as_is').map((c) => c.concession_id)).toEqual(['CON-1'])
    expect(usableConcessions(concessions(), 'repair')).toEqual([])
    expect(usableConcessions(concessions(), 'rework')).toEqual([])
  })

  it('переделке разрешение не нужно; основание решения обязательно', () => {
    expect(draftProblems(draft({ action: 'disposition', disposition: 'rework' }), [])).toEqual(['widgets.decisions.reasonRequired'])
    expect(draftProblems(draft({ action: 'disposition', disposition: 'rework', reason: 'Переварить шов' }), [])).toEqual([])
  })

  it('доп. проверка — с методом; «принять» без данных точки предъявления — нельзя', () => {
    expect(draftProblems(draft({ action: 'request_recheck', reason: 'Уточнить' }), [])).toContain('widgets.decisions.methodRequired')
    expect(draftProblems(draft({ action: 'accept_and_pass' }), [])).toContain('widgets.decisions.presentationContextMissing')
  })
})

describe('тело команды — только поля контракта (AD-7, AD-39)', () => {
  it('отклонение сигнала: сигналы, причина, basis_seq из карточки, policy_seq сеанса', () => {
    const req = buildDecisionRequest(draft({ action: 'reject_signal', reason: ' Блик на кромке ' }), ncCard(), { ...meta, workplace_id: 'WP-QC-1' })
    expect(req.action).toBe('reject_signal')
    expect(req.item_id).toBe('ENT:FL-0042')
    expect(req.body).toEqual({
      basis_seq: 1260,
      command_id: meta.command_id,
      policy_seq: 40,
      workplace_id: 'WP-QC-1',
      reason: { text: 'Блик на кромке' },
      signal_ids: ['SIG-77'],
    })
  })

  it('подтверждение: тяжесть, вид дефекта и требование — из сигнала', () => {
    const req = buildDecisionRequest(draft({ action: 'confirm_nc', reason: 'Пора подтверждена' }), ncCard(), meta)
    expect(req.body).toMatchObject({ severity: 'major', defect_type_code: 'W-POR', requirement_ref: 'ФЛ-100.02 п. 4.3 rev.C', signal_ids: ['SIG-77'] })
  })

  it('«как есть» уходит с разрешением; подпись агента — конвертом DSSE', () => {
    const signature = { payload: 'e30=', payloadType: 'application/vnd.ant.event+json; v=1', signatures: [{ keyid: 'qc-03@1', sig: 'AAAA' }] }
    const req = buildDecisionRequest(draft({ action: 'disposition', disposition: 'use_as_is', concession_id: 'CON-1', reason: 'По РО-12/26' }), confirmedCard(), { ...meta, signature })
    expect(req.nc_id).toBe('NC-0142')
    expect(req.body).toMatchObject({ disposition: 'use_as_is', concession_id: 'CON-1', signature })
  })

  it('доп. проверка — метод и зоны сигнала', () => {
    const req = buildDecisionRequest(draft({ action: 'request_recheck', method: 'radiography', reason: 'Проверить рентгеном' }), ncCard(), meta)
    expect(req.body).toMatchObject({ method: 'radiography', zone_ids: ['Z-weld-1'] })
  })
})

describe('окно подписи уровня 2 (FR-66)', () => {
  it('сводка — от 3 до 7 полей', () => {
    const s = decisionSummary(draft({ action: 'disposition', disposition: 'use_as_is', concession_id: 'CON-1', reason: 'По РО' }), confirmedCard(), concessions())
    expect(s.length).toBeGreaterThanOrEqual(3)
    expect(s.length).toBeLessThanOrEqual(SUMMARY_MAX)
    expect(s.map((f) => f.labelKey)).toContain('decisions.concession.title')
  })
})

describe('карточка: версии вывода, срок, порядок очереди', () => {
  it('текущая версия вывода — последняя; прежняя сохранена (FR-32)', () => {
    const v = conclusionVersions(ncCard().system_analysis.versions)
    expect(v.current?.version).toBe(2)
    expect(v.current?.revised_due_to).toBe('e-late-log')
    expect(v.previous.map((x) => x.version)).toEqual([1])
  })

  it('решение, принятое до пересмотра вывода, помечается «пересмотрите»', () => {
    expect([...decisionsBeforeRevision(ncCard())]).toEqual(['d-1'])
  })

  it('срок: осталось или просрочено, в минутах', () => {
    expect(deadlineOf('2026-09-23T12:00:00.000Z', Date.parse('2026-09-23T11:23:00.000Z'))).toEqual({ overdue: false, minutes: 37 })
    expect(deadlineOf('2026-09-23T12:00:00.000Z', Date.parse('2026-09-23T12:37:30.000Z'))).toEqual({ overdue: true, minutes: 37 })
    expect(deadlineOf(undefined, 0)).toBeNull()
  })

  it('порядок очереди из среза стола; по умолчанию — риск', () => {
    expect(queueSortOf(['risk', 'deadline'])).toBe('risk')
    expect(queueSortOf(['deadline'])).toBe('deadline')
    expect(queueSortOf(undefined)).toBe('risk')
  })

  it('состояние карточки — по оси качества: блок ≠ дефект (NFR-UI-4)', () => {
    expect(ncCardState(ncCard())).toBe('defect_indication')
    expect(ncCardState(ncCard({ axes: { ...ncCard().axes, quality: 'unable_to_assess' } }))).toBe('unable_to_assess')
    expect(ncCardState(ncCard({ axes: { ...ncCard().axes, quality: 'conforming' } }))).toBe('normal')
  })
})

describe('идентификатор команды', () => {
  it('UUIDv7: версия 7, вариант RFC, время в начале', () => {
    const id = uuidv7(Date.parse('2026-09-23T08:00:00.000Z'))
    expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
    expect(parseInt(id.replace(/-/g, '').slice(0, 12), 16)).toBe(Date.parse('2026-09-23T08:00:00.000Z'))
    expect(uuidv7()).not.toBe(uuidv7())
  })
})
