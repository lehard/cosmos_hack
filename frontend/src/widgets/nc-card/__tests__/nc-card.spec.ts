// Карточка несоответствия — досье для решения контролёра (FR-51, FR-32, FR-50,
// PRD §3a, UI-24): «что случилось» → «почему это проблема» (требование КД) →
// «как было дело» (до → операция → во время → после) → «что предлагает
// система» → решения людей и итоговый статус → подробности вторым слоем.
// Исходный сигнал, анализ системы, решения людей и итоговый статус раздельно.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { i18n } from '@/shared/i18n'
import { WIDGET_FRAME_CONTEXT } from '@/shared/ui'
import { at } from '@/entities/item/__tests__/fixtures'
import { ncCard } from '@/entities/nonconformity/__tests__/fixtures'
import type { NCCard } from '@/entities/nonconformity'
import NcCardView from '../ui/NcCardView.vue'

// Дорожка решения (Ф1) ходит за наблюдением и паспортом — здесь заглушка; сама дорожка проверена в features/decision-trace.
vi.mock('@/features/decision-trace', async () => {
  const { defineComponent, h } = await import('vue')
  return { NcDecisionTrace: defineComponent({ props: { card: { type: Object, required: true } }, setup: () => () => h('div', { 'data-testid': 'nc-decision-trace' }) }) }
})

const NOW = Date.parse(at('11:23'))
/** Пробелы в единицах — неразрывные (common.units). */
const norm = (x: string) => x.replace(/\s/g, ' ')
const global = (provide = {}) => ({ plugins: [createPinia(), i18n], stubs: { EvidenceMaterial: true }, provide })
const mountCard = (over: Partial<NCCard> = {}, props = {}) => mount(NcCardView, { props: { card: ncCard(over), now: NOW, ...props }, global: global() })
const mountWith = (card: NCCard) => mount(NcCardView, { props: { card, now: NOW }, global: global() })
const inWindow = () => mount(NcCardView, { props: { card: ncCard(), now: NOW }, global: global({ [WIDGET_FRAME_CONTEXT as symbol]: { hideTitle: true, plain: true } }) })

describe('карточка несоответствия', () => {
  it('разделы от смысла к технике: что случилось, требование, лента, предложение системы, решения, подробности', () => {
    const zones = mountCard().findAll('.zone').map((z) => z.attributes('data-zone'))
    expect(zones).toEqual(['what-happened', 'requirement', 'timeline', 'what-to-decide', 'decisions', 'details'])
    const w = mountCard()
    expect(w.find('[data-zone="requirement"] h3').text()).toBe('Почему это проблема')
    expect(w.find('[data-zone="timeline"] h3').text()).toBe('Как было дело')
    expect(w.find('[data-zone="what-to-decide"] h3').text()).toBe('Что предлагает система')
  })

  it('что случилось: признак крупно, основание и тяжесть, операция и время сигнала', () => {
    const hero = mountCard().find('[data-zone="what-happened"]')
    expect(hero.find('[data-testid="headline"]').text()).toBe('КТ-3, камера: обнаружен признак дефекта')
    expect(hero.find('.kicker').text()).toContain('значительный')
    expect(hero.find('.meta').text()).toContain('Сварка')
  })

  it('названия из справочников вместо кодов: вид дефекта, зона, оборудование; нет названия — прежний текст', async () => {
    const card = ncCard()
    const sig = card.evidence.signals[0]!
    sig.defect_type_label = 'Пора'
    sig.zone_label = 'Шов 1, участок 40–80 мм'
    card.happened.operation!.equipment_label = 'Сварочный источник ИС-3'
    const w = mountWith(card)
    expect(w.find('[data-testid="headline"]').text()).toBe('Пора')
    expect(w.find('[data-testid="zone"]').text()).toBe('Шов 1, участок 40–80 мм')
    expect(w.find('[data-testid="requirement"] .observed').text()).toContain('Пора · Шов 1, участок 40–80 мм')
    expect(w.find('[data-testid="operation"]').text()).toContain('Сварочный источник ИС-3')
    await w.find('[data-testid="toggle-details"]').trigger('click')
    expect(w.find('[data-testid="defect-type"]').text()).toBe('Пора (W-POR)')
    expect(w.find('[data-testid="zone-detail"]').text()).toBe('Шов 1, участок 40–80 мм (Z-weld-1)')
    // Неизвестный вид — название не подставляется.
    sig.defect_type_known = false
    expect(mountWith(card).find('[data-testid="headline"]').text()).toBe('КТ-3, камера: обнаружен признак дефекта')
  })

  it('два статуса: по изделию и системное расследование — из карточки, а не «нет данных»', () => {
    const two = mountCard().find('[data-testid="two-statuses"]').text()
    expect(two).toContain('По изделию: Черновик карточки')
    expect(two).toContain('Закрытие по изделию не закрывает расследование')
    expect(mountCard().find('[data-testid="investigation"]').text()).toBe('Системное расследование: Нет данных — неизвестно')
    expect(mountCard({ investigation_status: 'open' }).find('[data-testid="investigation"]').text()).toBe('Системное расследование: Идёт разбор')
  })

  it('уверенность анализатора рядом с качеством наблюдения; уверенность ≠ вероятность брака', () => {
    const facts = mountCard().find('[data-testid="signal-facts"]')
    const c = facts.find('[data-testid="confidence"]').text()
    expect(c).toContain('0,87')
    expect(c).toContain('Уверенность — не вероятность брака')
    expect(c).not.toContain('%')
    expect(facts.find('[data-testid="observation-quality"]').text()).toBe('0,91')
  })

  it('кадр контроля — в первом слое; нет кадра — так и написано', () => {
    expect(mountCard().find('[data-testid="materials"]').exists()).toBe(true)
    const card = ncCard()
    card.evidence.signals[0]!.evidence_refs = []
    expect(mountWith(card).find('[data-testid="no-material"]').text()).toContain('Кадр не передан источником')
  })

  it('требование КД рядом с наблюдением', () => {
    const req = mountCard().find('[data-testid="requirement"]')
    expect(req.text()).toContain('Требование КД')
    expect(req.text()).toContain('Поры в шве')
    expect(req.text()).toContain('не допускаются')
    expect(req.text()).toContain('ФЛ-100.02 п. 4.3 rev.C')
    expect(req.find('.observed').text()).toContain('обнаружен признак дефекта')
  })

  it('нет требования КД — вопрос технологу, а не брак', () => {
    const card = ncCard()
    delete card.evidence.requirement
    expect(mountWith(card).find('[data-testid="no-requirement"]').text()).toBe('Требования нет — это вопрос технологу, а не брак')
  })

  it('лента «как было дело»: до → операция → во время → после, отметки смысла, действия исполнителя — не вина', () => {
    const w = mountCard()
    const tl = w.find('[data-testid="evidence-timeline"]')
    const phases = tl.findAll('.phase').map((p) => p.text())
    expect(phases).toEqual(['До операции', 'Операция', 'Во время операции', 'После операции'])
    expect(tl.find('[data-id="e-kt2"]').attributes('data-tone')).toBe('ok')
    expect(tl.find('[data-id="e-current"]').attributes('data-tone')).toBe('warn')
    expect(tl.find('[data-id="e-override"]').attributes('data-tone')).toBe('warn')
    const op = w.find('[data-testid="operation"]').text()
    expect(op).toContain('ИС-3')
    expect(op).toContain('Горелка Г-2')
    expect(op).toContain('P-17 rev.4')
    expect(w.find('[data-testid="performer"]').text()).toBe('Исполнитель неизвестен')
    expect(tl.text()).toContain('Обстоятельство, а не вина')
    expect(w.find('[data-testid="similar-count"]').exists()).toBe(true)
  })

  it('полоса режима: уставка и наблюдение на одной шкале, выход за допуск — крупно; один раз на параметр', () => {
    const card = ncCard()
    const reading = { parameter: 'current_a', unit: 'A', scale: 0, setpoint_nominal: 180, setpoint_min: 170, setpoint_max: 190, observed_min: 205, observed_max: 212 }
    card.happened.during[0]!.reading = reading
    card.happened.during.push({ ...card.happened.during[0]!, event_id: 'e-cycle', event_type: 'equipment.cycle.summarized', occurred_at: at('08:40'), reading })
    const w = mountWith(card)
    const bars = w.findAll('[data-testid="regime-bar"]')
    expect(bars).toHaveLength(1)
    expect(w.find('[data-id="e-cycle"] [data-testid="regime-bar"]').exists()).toBe(true)
    expect(norm(bars[0]!.find('[data-testid="regime-verdict"]').text())).toBe('Ток сварки: макс. 212 А — на 22 А выше допуска')
    expect(norm(bars[0]!.text())).toContain('уставка 170–190 А')
    expect(norm(bars[0]!.text())).toContain('наблюдалось 205–212 А')
    expect(bars[0]!.findAll('.observed[data-out]')).toHaveLength(1)
  })

  it('режим в пределах уставки — так и написано, без красного', () => {
    const card = ncCard()
    card.happened.during[0]!.reading = { parameter: 'voltage_x', unit: 'V', scale: 1, setpoint_min: 200, setpoint_max: 240, observed_min: 210, observed_max: 230 }
    const bar = mountWith(card).find('[data-testid="regime-bar"]')
    expect(bar.find('[data-testid="regime-verdict"]').text()).toBe('в пределах уставки')
    expect(norm(bar.text())).toContain('уставка 20–24 В')
    expect(bar.findAll('.observed[data-out]')).toHaveLength(0)
  })

  it('история коротко: до 3 главных событий в фазе, признаки и обстоятельства — первыми; вся история по кнопке', async () => {
    const card = ncCard()
    const base = card.happened.before[0]!
    card.happened.before = [
      { ...base, event_id: 'b1', occurred_at: at('07:10'), params: {} },
      { ...base, event_id: 'b2', occurred_at: at('07:20'), params: {} },
      { ...base, event_id: 'b3', occurred_at: at('07:30'), params: { outcome: 'no_defect_indicated' } },
      { ...base, event_id: 'b4', occurred_at: at('07:40'), params: { outcome: 'unable_to_assess' } },
    ]
    const w = mountWith(card)
    expect(w.findAll('[data-id^="b"]').map((e) => e.attributes('data-id'))).toEqual(['b1', 'b3', 'b4'])
    await w.find('[data-testid="toggle-history"]').trigger('click')
    expect(w.findAll('[data-id^="b"]')).toHaveLength(4)
  })

  it('что предлагает система: предложение, основания, альтернативы, нехватка сведений; правило — во втором слое', () => {
    const why = mountCard().find('[data-testid="why-system"]')
    expect(why.text()).toContain('Это предложение системы, а не решение')
    expect(why.find('[data-testid="conclusion-current"]').text()).toContain('Изолировать до решения')
    expect(why.text()).toContain('строка 12 карты реакций')
    expect(why.find('[data-testid="alternatives"]').text()).toContain('Блик на кромке шва')
    expect(why.find('[data-testid="missing"]').text()).toContain('Неизвестен инструмент')
    expect(why.text()).not.toContain('RM-weld-7')
  })

  it('подробности для проверки свёрнуты: исходный сигнал целиком, ступени и версии, правило и режим', async () => {
    const w = mountCard()
    expect(w.find('[data-testid="details"]').exists()).toBe(false)
    await w.find('[data-testid="toggle-details"]').trigger('click')
    const det = w.find('[data-testid="details"]')
    expect(det.text()).toContain('Исходный сигнал')
    expect(det.find('[data-testid="source-signal"]').text()).toContain('Где дефект: шов 1, 40 мм, уверенность 0,93, версия loc-2.1')
    expect(det.text()).toContain('vqc-3.2')
    expect(det.text()).toContain('Это сигнал, а не несоответствие')
    expect(det.find('[data-testid="rule"]').text()).toContain('Правило: RM-weld-7')
    expect(det.find('[data-testid="rule"]').text()).toContain('Сама по утверждённому правилу')
  })

  it('неизвестный вид дефекта — с пометкой, не подменяется', async () => {
    const card = ncCard()
    card.evidence.signals[0]!.defect_type_known = false
    const w = mountWith(card)
    await w.find('[data-testid="toggle-details"]').trigger('click')
    expect(w.find('[data-testid="defect-type"]').text()).toBe('Неизвестный вид дефекта: W-POR')
  })

  it('две версии вывода: текущая с причиной пересмотра, прежняя сохранена (FR-32)', async () => {
    const w = mountCard()
    expect(w.find('[data-testid="revised"]').text()).toBe('Пересмотрен из-за события e-late-log')
    expect(w.find('[data-testid="conclusion-previous"]').exists()).toBe(false)
    await w.find('[data-testid="toggle-previous"]').trigger('click')
    const prev = w.find('[data-testid="conclusion-previous"]')
    expect(prev.text()).toContain('Ручной осмотр')
    expect(prev.text()).toContain('Прежний вывод сохранён в журнале')
  })

  it('решение человека до пересмотра вывода помечено «пересмотрите»', () => {
    const d = mountCard().find('[data-testid="human-decisions"]')
    expect(d.text()).toContain('Назначена доп. проверка: рентген')
    expect(d.find('[data-testid="record-mark"]').text()).toBe('Решение принято до новых данных — пересмотрите')
  })

  it('строка записи: вид записи не повторяется, номер журнала — только в подробностях (хвост UI-18)', () => {
    const d = mountCard().find('[data-testid="human-decisions"]').text()
    expect(d.match(/Решение человека/g)?.length ?? 0).toBeLessThanOrEqual(1)
    expect(d).not.toContain('Запись журнала')
  })

  it('в первом слое — «изделие сейчас» одной строкой; пять осей — по кнопке «Технические статусы»', async () => {
    const w = mountCard()
    expect(w.find('[data-testid="item-now"]').text()).toContain('Изделие сейчас')
    expect(w.find('[data-testid="final-status"]').exists()).toBe(false)
    await w.find('[data-testid="toggle-axes"]').trigger('click')
    expect(w.findAll('[data-testid="final-status"] .status-tag')).toHaveLength(5)
  })

  it('срок решения: обратный отсчёт и просрочка (FR-55)', () => {
    expect(norm(mountCard().find('[data-testid="deadline"]').text())).toBe('Осталось 37 мин')
    const late = mountCard({}, { now: Date.parse(at('12:37')) })
    expect(norm(late.find('[data-testid="deadline"]').text())).toBe('Просрочено на 37 мин')
    expect(late.find('[data-testid="deadline"]').attributes('data-overdue')).toBe('true')
  })

  it('решений людей нет — «нет решения ≠ нет данных»', () => {
    expect(mountCard({ human_decisions: [] }).find('[data-testid="human-decisions"]').text()).toBe('Нет решения ≠ нет данных')
  })

  it('в окне записи (Д-70): номер и изделие не повторяются, срок виден, кнопок решения нет', () => {
    const w = inWindow()
    expect(w.find('[data-testid="nc-card"]').attributes('data-in-window')).toBe('true')
    expect(w.find('.number').exists()).toBe(false)
    expect(w.find('[data-testid="open-item"]').exists()).toBe(false)
    expect(norm(w.find('[data-testid="deadline"]').text())).toBe('Осталось 37 мин')
    // Кнопки решения — только в нижней панели окна; здесь лишь раскрытие подробностей.
    expect(w.findAll('button').map((b) => b.attributes('data-testid')).sort()).toEqual(['toggle-axes', 'toggle-details', 'toggle-previous'])
  })
})
