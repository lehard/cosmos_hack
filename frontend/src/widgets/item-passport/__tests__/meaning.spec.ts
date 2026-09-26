// NFR-UI-4 «Экран не приписывает данным смысл, которого в них нет»: смысл
// статусов закреплён тестами на тех экранах, где статусы показываются.
// «Оценка невозможна» ≠ «годно»; «под подозрением» ≠ «брак»; «заблокировано»
// ≠ «признано дефектным»; «уверенность 0,87» ≠ «вероятность брака 87 %»;
// «годно по разрешению на отклонение» ≠ «годно»; «нет данных» ≠ «годно».
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { qualityState } from '@/entities/item'
import { SummaryTag } from '@/entities/item'
import { i18n } from '@/shared/i18n'
import { StatusTag } from '@/shared/ui'
import { flangePassport } from '@/entities/item/__tests__/fixtures'
import { ncCard } from '@/entities/nonconformity/__tests__/fixtures'
import NcCardView from '@/widgets/nc-card/ui/NcCardView.vue'
import PassportHeader from '../ui/PassportHeader.vue'

const g = () => ({ plugins: [createPinia(), i18n] })
const header = (status: Partial<ReturnType<typeof flangePassport>['status']>) =>
  mount(PassportHeader, { props: { passport: flangePassport({ status: { ...flangePassport().status, ...status } }) }, global: g() })
const tag = (axis: string, code: string) => mount(StatusTag, { props: { axis: axis as never, code }, global: g() }).text()

describe('смысл статусов на экране (NFR-UI-4)', () => {
  it('«оценка невозможна» — не «годно»: своё слово, своё состояние, пояснение', () => {
    expect(tag('quality', 'unable_to_assess')).toBe('Оценка невозможна')
    expect(tag('quality', 'unable_to_assess')).not.toContain('Годно')
    expect(qualityState({ quality: 'unable_to_assess' })).toBe('unable_to_assess')
    const w = header({ quality: 'unable_to_assess' })
    expect(w.find('[data-testid="unable-hint"]').text()).toContain('Это не «годно» и не брак')
  })

  it('«под подозрением» — не «брак»', () => {
    expect(tag('incident', 'suspect')).toBe('Под подозрением')
    const summary = mount(SummaryTag, { props: { code: 'suspect' }, global: g() }).text()
    expect(summary).toBe('Под подозрением')
    for (const s of [tag('incident', 'suspect'), summary]) {
      expect(s).not.toMatch(/брак|несоответствие/i)
    }
  })

  it('«заблокировано» — не «признано дефектным»: блок не меняет состояние качества', () => {
    const w = header({ quality: 'conforming', containment: 'item_hold', summary: 'hold' })
    expect(w.find('[data-testid="axes"]').text()).toContain('Блок изделия')
    expect(w.find('[data-testid="axes"]').text()).toContain('Годно')
    expect(w.find('[data-testid="hold-hint"]').text()).toContain('Это не значит «признано дефектным»; снятие блока не значит «годно»')
    expect(qualityState({ quality: 'conforming' })).toBe('normal')
    expect(mount(SummaryTag, { props: { code: 'hold' }, global: g() }).text()).toBe('Заблокировано')
  })

  it('изоляция — положение, а не решение и не брак', () => {
    const w = header({ position: 'isolated', quality: 'not_inspected', containment: 'none' })
    expect(w.find('[data-testid="isolation-hint"]').text()).toContain('Изоляция — положение изделия, ожидающего решения')
    expect(qualityState({ quality: 'not_inspected' })).toBe('normal')
  })

  it('«годно по разрешению на отклонение» — не «годно»', () => {
    expect(tag('quality', 'accepted_with_concession')).toBe('Годно по разрешению на отклонение')
    expect(tag('quality', 'accepted_with_concession')).not.toBe(tag('quality', 'conforming'))
    expect(header({ quality: 'accepted_with_concession' }).find('[data-testid="concession-hint"]').text()).toContain('это не «годно»')
  })

  it('«сигнал» ≠ «подтверждённое несоответствие»: разные слова одной оси', () => {
    expect(tag('quality', 'signal')).toBe('Сигнал о признаке дефекта')
    expect(tag('quality', 'nonconforming')).toBe('Несоответствие подтверждено')
  })

  it('уверенность 0,87 — не «вероятность брака 87 %»', () => {
    const w = mount(NcCardView, { props: { card: ncCard(), now: 0 }, global: g() })
    const c = w.find('[data-testid="confidence"]').text()
    expect(c).toContain('0,87')
    expect(c).toContain('Уверенность — не вероятность брака')
    expect(c).not.toMatch(/87\s*%|вероятность брака 87/)
  })

  it('нет уверенности — «нет данных — неизвестно», а не ноль и не «годно»', () => {
    const card = ncCard()
    delete card.evidence.signals[0]!.analyzer_confidence_bp
    const w = mount(NcCardView, { props: { card, now: 0 }, global: g() })
    expect(w.find('[data-testid="confidence"]').text()).toBe('Нет данных — неизвестно')
  })

  it('код вне словаря — UNKNOWN(код), а не похожий статус', () => {
    expect(tag('quality', 'almost_ok')).toBe('UNKNOWN(almost_ok)')
    expect(mount(SummaryTag, { props: { code: 'almost_ok' }, global: g() }).text()).toBe('UNKNOWN(almost_ok)')
  })
})
