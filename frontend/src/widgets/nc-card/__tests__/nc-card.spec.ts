// Карточка несоответствия в три зоны; исходный сигнал, анализ системы, решения
// людей и итоговый статус раздельно; «почему система это предлагает» и две
// версии вывода с причиной пересмотра (FR-51, FR-32, FR-50, PRD §3a).
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { WIDGET_FRAME_CONTEXT } from '@/shared/ui'
import { at } from '@/entities/item/__tests__/fixtures'
import { ncCard } from '@/entities/nonconformity/__tests__/fixtures'
import NcCardView from '../ui/NcCardView.vue'

const NOW = Date.parse(at('11:23'))
/** Пробелы в единицах — неразрывные (common.units). */
const norm = (x: string) => x.replace(/\s/g, ' ')
const mountCard = (over = {}, props = {}) => mount(NcCardView, { props: { card: ncCard(over), now: NOW, ...props }, global: { plugins: [createPinia(), i18n] } })

describe('карточка несоответствия', () => {
  it('три зоны: что произошло, доказательства, что решить', () => {
    const zones = mountCard().findAll('.zone')
    expect(zones.map((z) => z.attributes('data-zone'))).toEqual(['what-happened', 'evidence', 'what-to-decide'])
    expect(zones.map((z) => z.find('h3').text())).toEqual(['Что произошло', 'Доказательства', 'Что решить'])
  })

  it('четыре слоя раздельно: исходный сигнал, анализ системы, решения людей, итоговый статус', () => {
    const text = mountCard().text()
    for (const s of ['Исходный сигнал', 'Анализ системы', 'Решения людей', 'Итоговый статус']) expect(text).toContain(s)
  })

  it('два статуса: по изделию и системное расследование — закрытие одного не закрывает другое', () => {
    const two = mountCard().find('[data-testid="two-statuses"]').text()
    expect(two).toContain('По изделию: Черновик карточки')
    expect(two).toContain('Системное расследование')
    expect(two).toContain('Закрытие по изделию не закрывает расследование')
  })

  it('операция: станок, инструмент, программа; исполнитель неизвестен — так и написано', () => {
    const w = mountCard()
    const op = w.find('[data-testid="operation"]').text()
    expect(op).toContain('ИС-3')
    expect(op).toContain('Горелка Г-2')
    expect(op).toContain('P-17 rev.4')
    expect(w.find('[data-testid="performer"]').text()).toBe('Исполнитель неизвестен')
    expect(w.text()).toContain('Обстоятельство, а не вина')
  })

  it('исходный сигнал: уверенность ≠ вероятность брака, качество наблюдения отдельно, ступени и версии', () => {
    const s = mountCard().find('[data-testid="source-signal"]')
    expect(s.find('[data-testid="confidence"]').text()).toContain('0,87')
    expect(s.find('[data-testid="confidence"]').text()).toContain('Уверенность — не вероятность брака')
    expect(s.find('[data-testid="confidence"]').text()).not.toContain('%')
    expect(s.find('[data-testid="observation-quality"]').text()).toBe('0,91')
    expect(s.text()).toContain('Где дефект: шов 1, 40 мм, уверенность 0,93, версия loc-2.1')
    expect(s.text()).toContain('vqc-3.2')
    expect(s.text()).toContain('Это сигнал, а не несоответствие')
  })

  it('неизвестный вид дефекта — с пометкой, не подменяется', () => {
    const card = ncCard()
    card.evidence.signals[0]!.defect_type_known = false
    const w = mount(NcCardView, { props: { card, now: NOW }, global: { plugins: [createPinia(), i18n] } })
    expect(w.find('[data-testid="defect-type"]').text()).toBe('Неизвестный вид дефекта: W-POR')
  })

  it('нет требования КД — вопрос технологу, а не брак', () => {
    const card = ncCard()
    delete card.evidence.requirement
    const w = mount(NcCardView, { props: { card, now: NOW }, global: { plugins: [createPinia(), i18n] } })
    expect(w.find('[data-testid="no-requirement"]').text()).toBe('Требования нет — это вопрос технологу, а не брак')
  })

  it('«почему система это предлагает»: правило, режим, основания, альтернативы, нехватка сведений', () => {
    const why = mountCard().find('[data-testid="why-system"]')
    expect(why.text()).toContain('Это предложение системы, а не решение')
    expect(why.find('[data-testid="conclusion-current"]').text()).toContain('Изолировать до решения')
    expect(why.text()).toContain('Правило: RM-weld-7')
    expect(why.text()).toContain('Сама по утверждённому правилу')
    expect(why.text()).toContain('строка 12 карты реакций')
    expect(why.find('[data-testid="alternatives"]').text()).toContain('Блик на кромке шва')
    expect(why.find('[data-testid="missing"]').text()).toContain('Неизвестен инструмент')
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

  it('срок решения: обратный отсчёт и просрочка (FR-55)', () => {
    expect(norm(mountCard().find('[data-testid="deadline"]').text())).toBe('Осталось 37 мин')
    const late = mountCard({}, { now: Date.parse(at('12:37')) })
    expect(norm(late.find('[data-testid="deadline"]').text())).toBe('Просрочено на 37 мин')
    expect(late.find('[data-testid="deadline"]').attributes('data-overdue')).toBe('true')
  })

  it('решений людей нет — «нет решения ≠ нет данных»', () => {
    expect(mountCard({ human_decisions: [] }).find('[data-testid="human-decisions"]').text()).toBe('Нет решения ≠ нет данных')
  })

  it('в окне записи (Д-70): номер и изделие не повторяются, срок виден, зона решения — «основания», без кнопок решения', () => {
    const w = mount(NcCardView, {
      props: { card: ncCard(), now: NOW },
      global: { plugins: [createPinia(), i18n], provide: { [WIDGET_FRAME_CONTEXT as symbol]: { hideTitle: true, plain: true } } },
    })
    expect(w.find('[data-testid="nc-card"]').attributes('data-in-window')).toBe('true')
    expect(w.find('.number').exists()).toBe(false)
    expect(w.find('[data-testid="open-item"]').exists()).toBe(false)
    expect(norm(w.find('[data-testid="deadline"]').text())).toBe('Осталось 37 мин')
    expect(w.find('[data-zone="what-to-decide"] h3').text()).toBe('Основания для решения')
    expect(w.find('[data-zone="what-to-decide"]').text()).toContain('Почему система это предлагает')
    // Кнопки решения — только в нижней панели окна; здесь лишь «показать прежние версии».
    expect(w.findAll('[data-zone="what-to-decide"] button').map((b) => b.attributes('data-testid'))).toEqual(['toggle-previous'])
  })
})
