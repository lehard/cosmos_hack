// Панель решений: кнопки «действие — направление», обязательная причина
// отклонения, выбор разрешения на отклонение для ремонта и «как есть»,
// «почему вы можете / не можете» и «Запросить решение» (FR-52, FR-53, FR-146).
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { at } from '@/entities/item/__tests__/fixtures'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { concessions, confirmedCard, ncCard } from '@/entities/nonconformity/__tests__/fixtures'
import DecisionPanelView from '../ui/DecisionPanelView.vue'

const ALL = new Set([
  'nonconformity.nonconformity.confirm',
  'nonconformity.signal.reject',
  'nonconformity.recheck.request',
  'nonconformity.item.isolate',
  'nonconformity.disposition.set',
])
const mountPanel = (props = {}) =>
  mount(DecisionPanelView, { props: { card: ncCard(), allowed: ALL, ...props }, global: { plugins: [createPinia(), i18n] } })

describe('панель решений контролёра', () => {
  it('кнопки названы действием и направлением — только допустимые по состоянию', () => {
    const labels = mountPanel().findAll('[data-group="signal"] button').map((b) => b.text())
    expect(labels).toEqual([
      'Подтвердить несоответствие',
      'Отклонить сигнал — изделие продолжает маршрут',
      'Назначить доп. проверку — изделие ждёт в изоляции',
      'Изолировать — до решения по изделию',
    ])
  })

  it('отклонение без причины невозможно (FR-52)', async () => {
    const w = mountPanel()
    await w.find('[data-action="reject_signal"]').trigger('click')
    expect(w.find('[data-testid="decision-form"]').text()).toContain('Почему сигнал отклонён (обязательно)')
    expect(w.find('[data-testid="problems"]').text()).toContain('Укажите, почему сигнал отклонён')
    expect(w.find('[data-testid="sign"]').attributes('disabled')).toBeDefined()
    await w.find('[data-testid="reason"] textarea').setValue('Блик на кромке, на повторном снимке признаков нет')
    expect(w.find('[data-testid="problems"]').exists()).toBe(false)
    await w.find('[data-testid="sign"]').trigger('click')
    expect(w.emitted('sign')?.[0]?.[0]).toMatchObject({ action: 'reject_signal', reason: 'Блик на кромке, на повторном снимке признаков нет' })
  })

  it('исходный сигнал сохраняется — решение новой записью', async () => {
    const w = mountPanel()
    await w.find('[data-action="reject_signal"]').trigger('click')
    expect(w.text()).toContain('Исходный сигнал сохраняется в истории без изменений')
  })

  it('нет полномочий — «Запросить решение» вместо «Подписать» (FR-146)', async () => {
    const w = mountPanel({ allowed: new Set(['nonconformity.signal.reject']) })
    await w.find('[data-action="confirm_nc"]').trigger('click')
    expect(w.find('[data-testid="sign"]').exists()).toBe(false)
    expect(w.find('[data-testid="why"]').text()).toBe('Почему вы не можете это сделать')
    await w.find('[data-testid="request-decision"]').trigger('click')
    expect(w.emitted('request-decision')?.[0]).toEqual(['nonconformity.nonconformity.confirm'])
    await w.find('[data-testid="why"]').trigger('click')
    expect(w.emitted('explain')?.[0]).toEqual(['nonconformity.nonconformity.confirm'])
  })

  it('права ещё не пришли — подписать нельзя', async () => {
    const w = mountPanel({ allowed: null })
    await w.find('[data-action="isolate"]').trigger('click')
    await w.find('[data-testid="reason"] textarea').setValue('Признак дефекта')
    expect(w.find('[data-testid="sign"]').attributes('disabled')).toBeDefined()
  })

  it('решение по несоответствию: переделка / ремонт / как есть / списать / вернуть', () => {
    const w = mountPanel({ card: confirmedCard() })
    expect(w.findAll('[data-group="disposition"] .option-title').map((b) => b.text())).toEqual([
      'Переделка',
      'Ремонт',
      'Принять как есть',
      'Списать',
      'Вернуть поставщику',
    ])
    expect(w.text()).toContain('Решение по изделию не ждёт установления причины')
    // Пилюля — коротко; целиком, с условием — в подсказке.
    expect(w.find('[data-disposition="rework"]').attributes('title')).toBe('Переделка — вернуть на операцию · без разрешения')
    // Этап 2 — карточки вариантов: смысл, условия, итог; первичных «подтвердить / отклонить» нет.
    const asIs = w.find('[data-disposition="use_as_is"]')
    expect(asIs.find('.option-terms').text()).toBe('требуется разрешение на отклонение')
    expect(w.find('[data-disposition="rework"] .option-terms').text()).toBe('без разрешения')
    // Шаг 1 свёрнут в итог, шаг 2 — «требуется ваше решение», шаг 3 — «ещё не начато».
    const states = w.findAll('[data-testid="stage-state"]').map((x) => x.text())
    expect(states).toEqual(['завершено: несоответствие подтверждено', 'требуется ваше решение', 'ещё не начато'])
    expect(w.find('[data-action="confirm_nc"]').exists()).toBe(false)
    expect(w.find('[data-stage="disposition"]').attributes('data-state')).toBe('current')
  })

  it('выбранный вариант отмечен; смысл и итог — после выбора', async () => {
    const w = mountPanel({ card: confirmedCard() })
    expect(w.find('[data-testid="chosen-title"]').exists()).toBe(false)
    await w.find('[data-disposition="scrap"]').trigger('click')
    expect(w.find('[data-disposition="scrap"]').attributes('aria-pressed')).toBe('true')
    expect(w.find('[data-testid="chosen-title"]').text()).toBe('Списать — оформить акт о браке')
    expect(w.find('[data-testid="chosen-outcome"]').text()).toContain('акт о браке')
    expect(w.find('[data-disposition="rework"]').attributes('aria-pressed')).toBe('false')
  })

  it('действия сервера: недоступный вариант — с причиной; у выбранного — деловые последствия, технические по раскрытию (п. 15)', async () => {
    const card = confirmedCard()
    card.to_decide.actions = [
      { operation: 'nonconformity.disposition.set', disposition: 'rework', label: 'Переделка', allowed: true, why_available: 'Лимит доработок зоны не исчерпан: 0 из 3', consequences: ['Изделие вернётся на сварку', 'Мастеру участка — задача переделки'], technical_consequences: ['1С: перевод в брак (переделка)'] },
      { operation: 'nonconformity.disposition.set', disposition: 'return_to_supplier', label: 'Вернуть', allowed: false, why_available: 'Недоступно: дефект возник на сварке', consequences: [], technical_consequences: [] },
    ]
    const w = mountPanel({ card })
    const ret = w.find('[data-disposition="return_to_supplier"]')
    expect(ret.attributes('disabled')).toBeDefined()
    expect(ret.find('[data-testid="option-why"]').text()).toBe('Недоступно: дефект возник на сварке')
    await w.find('[data-disposition="rework"]').trigger('click')
    expect(w.find('[data-testid="why-available"]').text()).toContain('0 из 3')
    expect(w.findAll('[data-testid="consequences"] li').map((x) => x.text())).toEqual(['Изделие вернётся на сварку', 'Мастеру участка — задача переделки'])
    expect(w.find('[data-testid="technical-consequences"]').exists()).toBe(false)
    await w.find('[data-testid="toggle-technical"]').trigger('click')
    expect(w.find('[data-testid="technical-consequences"]').text()).toContain('1С: перевод в брак')
  })

  it('после решения по изделию — «передано на исполнение: кому, что, состояние» (стык с мастером)', () => {
    const card = confirmedCard()
    card.status = 'disposition_set'
    card.handoff = { decision_event_id: 'd-9', role_id: 'site_foreman', role_label: 'мастеру участка', task_title: 'Переделка на СВ-017-1', status: 'waiting', status_label: 'ожидает исполнения', since: at('10:00') }
    const w = mountPanel({ card, receipt: { command_id: 'c', event_ids: ['e'], replayed: false, seq: 1300 } })
    expect(w.findAll('[data-testid="stage-state"]').map((x) => x.text())).toEqual([
      'завершено: несоответствие подтверждено',
      'решение принято',
      'передано мастеру участка: Переделка на СВ-017-1 · ожидает исполнения',
    ])
    expect(w.find('[data-testid="receipt-handoff"]').text()).toBe('Передано на исполнение мастеру участка: Переделка на СВ-017-1 · ожидает исполнения')
  })

  it('«как есть» без действующего разрешения не подписывается — оформить новое (FR-53, FR-54)', async () => {
    const w = mountPanel({ card: confirmedCard(), concessions: concessions().filter((c) => c.status !== 'active') })
    await w.find('[data-disposition="use_as_is"]').trigger('click')
    await w.find('[data-testid="reason"] textarea').setValue('Пора 0,3 мм')
    expect(w.find('[data-testid="no-concession"]').text()).toBe('Действующего разрешения нет — оформить новое')
    expect(w.find('[data-testid="problems"]').text()).toContain('Выберите действующее разрешение на отклонение')
    expect(w.find('[data-testid="sign"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="concession-result"]').text()).toBe('Итог: «годно по разрешению на отклонение» — это не «годно»')
    await w.find('[data-testid="create-concession"]').trigger('click')
    expect(w.emitted('create-concession')).toHaveLength(1)
  })

  it('действующее разрешение можно выбрать', async () => {
    const w = mountPanel({ card: confirmedCard(), concessions: concessions() })
    await w.find('[data-disposition="use_as_is"]').trigger('click')
    expect(w.find('[data-testid="concession"]').exists()).toBe(true)
    expect(w.find('[data-testid="no-concession"]').exists()).toBe(false)
  })

  it('«вернуть поставщику» — только для необработанных изделий партии', async () => {
    const w = mountPanel({ card: confirmedCard() })
    await w.find('[data-disposition="return_to_supplier"]').trigger('click')
    expect(w.find('[data-testid="return-only-unprocessed"]').text()).toBe('Вернуть поставщику можно только необработанные изделия партии')
  })

  it('воспроизведение — действия выключены', () => {
    const w = mountPanel({ canAct: false })
    expect(w.find('[data-testid="replay-note"]').text()).toBe('Действия недоступны в режиме воспроизведения')
    for (const b of w.findAll('[data-group="signal"] button')) expect(b.attributes('disabled')).toBeDefined()
  })

  it('квитанция: номер записи журнала и критического действия', () => {
    const w = mountPanel({ receipt: { command_id: 'c', event_ids: ['e'], replayed: false, seq: 1270, ca_ref: 'CA-312' } })
    expect(w.find('[data-testid="receipt"]').text()).toContain('Решение записано')
    expect(w.find('[data-testid="receipt-ref"]').text()).toBe('Запись журнала № 1270 · CA-312')
  })
})
