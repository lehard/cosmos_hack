// Правила паспорта изделия: состояние экрана, пометка источника, проверка
// подписи, генеалогия, зоны (FR-42, FR-45, FR-46, FR-68, FR-140, NFR-UI-4).
import { describe, expect, it } from 'vitest'
import {
  SIGNATURE_CHECK,
  entryText,
  identificationDoubtful,
  isHeld,
  qualityState,
  sortEntries,
  sourceMark,
  splitGenealogy,
  zoneStatus,
} from '@/entities/item'
import { flangeEntries, flangeGenealogy } from './fixtures'

describe('состояние экрана паспорта — только по оси качества (NFR-UI-4)', () => {
  it('сигнал и подтверждённое несоответствие — «признак дефекта»', () => {
    expect(qualityState({ quality: 'signal' })).toBe('defect_indication')
    expect(qualityState({ quality: 'nonconforming' })).toBe('defect_indication')
  })

  it('«оценка невозможна» — своё состояние, не «норма» и не «годно»', () => {
    expect(qualityState({ quality: 'unable_to_assess' })).toBe('unable_to_assess')
  })

  it('«заблокировано» ≠ «признано дефектным»: блок не делает паспорт дефектным', () => {
    expect(isHeld({ containment: 'item_hold' })).toBe(true)
    expect(qualityState({ quality: 'conforming' })).toBe('normal')
    expect(qualityState({ quality: 'not_inspected' })).toBe('normal')
  })
})

describe('пометка источника факта (FR-140)', () => {
  it('ручная отметка — ручной ввод, а не данные станка', () => {
    const m = sourceMark({ kind: 'fact', source_kind: 'manual_entry', reliability: 'medium' })
    expect(m).toMatchObject({ code: 'manual_entry', key: 'timeline.sourceKind.manual', manual: true, reliabilityKey: 'widgets.passport.reliability.medium' })
    expect(sourceMark({ kind: 'fact', source_kind: 'machine' })).toMatchObject({ code: 'machine', manual: false })
  })

  it('вывод системы, решение человека и факт без источника различимы', () => {
    expect(sourceMark({ kind: 'reaction' }).code).toBe('system')
    expect(sourceMark({ kind: 'decision' }).code).toBe('person')
    // Не додумываем: без вида источника — «неизвестен», а не «станок» (FR-123).
    expect(sourceMark({ kind: 'fact' })).toMatchObject({ code: 'unknown', key: 'widgets.passport.source.unknown' })
  })
})

describe('проверка подписи (FR-68)', () => {
  it('зелёная пометка — только у действительной подписи', () => {
    expect(SIGNATURE_CHECK.valid.tone).toBe('success')
    for (const c of ['rejected', 'not_verifiable', 'unchecked'] as const) expect(SIGNATURE_CHECK[c].tone).not.toBe('success')
  })
})

describe('записи и генеалогия', () => {
  it('текст записи — от сервера; нет — название типа из каталога; нет и там — UNKNOWN', () => {
    expect(entryText({ summary: 'Сварка начата', event_type: 'operation.run.started' })).toBe('Сварка начата')
    expect(entryText({ summary: '', event_type: 'decision.signal.rejected' })).toBe('Сигнал отклонён')
    expect(entryText({ summary: '', event_type: 'x.y.z' })).toBe('UNKNOWN(x.y.z)')
  })

  it('записи — по времени возникновения, при равенстве — по seq', () => {
    const ids = sortEntries(flangeEntries()).map((e) => e.event_id)
    expect(ids.slice(0, 3)).toEqual(['e-reg', 'e-weld-start', 'e-weld-end'])
    expect(ids.indexOf('e-weld-end')).toBeLessThan(ids.indexOf('e-fix'))
  })

  it('генеалогия: куда входит, из чего состоит, партии, выписки (FR-45)', () => {
    const g = splitGenealogy(flangeGenealogy())
    expect(g.up.map((n) => n.ref)).toEqual(['ENT:ASM-9'])
    expect(g.down.map((n) => n.ref)).toEqual(['ENT:RING-3'])
    expect(g.lots.map((n) => n.ref)).toEqual(['LOT-ST-19'])
    expect(g.extracts.map((n) => n.ref)).toEqual(['PX-44'])
  })

  it('зона: «проверена» не утверждается, если сервер этого не сообщил (FR-46)', () => {
    expect(zoneStatus({ closed: false })).toBeNull()
    expect(zoneStatus({ closed: true })).toBe('closed')
    expect(zoneStatus({ closed: true, open_intervention: 'INT-5' })).toBe('stale_after_intervention')
  })

  it('идентификация под сомнением — неоднозначно или не опознано (AD-16)', () => {
    expect(identificationDoubtful('ambiguous')).toBe(true)
    expect(identificationDoubtful('unidentified')).toBe(true)
    expect(identificationDoubtful('probable')).toBe(false)
  })
})
