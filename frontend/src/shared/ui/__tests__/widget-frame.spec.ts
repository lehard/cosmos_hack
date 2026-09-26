// Рамка виджета (AD-21, NFR-UI-4): четыре состояния, момент, метка режима,
// выключение действий в воспроизведении.
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { h } from 'vue'
import { i18n } from '@/shared/i18n'
import { useMomentStore } from '@/shared/model/moment'
import WidgetFrame from '../WidgetFrame.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})

const mountFrame = (props: Record<string, unknown>) =>
  mount(WidgetFrame, {
    props: { titleKey: 'desks.passport', ...props },
    slots: { default: () => 'данные', actions: () => h('button', { class: 'act' }, 'Действие') },
    global: { plugins: [pinia, i18n] },
  })

describe('рамка виджета', () => {
  it.each([
    ['normal', 'Норма', false],
    ['defect_indication', 'Обнаружен признак дефекта', true],
    ['unable_to_assess', 'Оценка невозможна', true],
    ['input_error', 'Ошибка входных данных', true],
  ])('состояние %s', (state, label, shown) => {
    const w = mountFrame({ state })
    expect(w.attributes('data-state')).toBe(state)
    expect(w.text().includes(label)).toBe(shown)
  })

  it('ошибка запроса — «ошибка входа» с текстом по коду контракта', () => {
    const err = Object.assign(new Error(), { status: 501, info: { type: 'urn:ant:problem:api.not_implemented', title: 'Операция ещё не реализована', status: 501, code: 'api.not_implemented' } })
    const w = mountFrame({ error: err })
    expect(w.attributes('data-state')).toBe('input_error')
    expect(w.text()).toContain('Не удалось выполнить действие')
  })

  it('метка режима fixtures | live и момент «Сейчас»', () => {
    const w = mountFrame({ mode: 'fixtures' })
    expect(w.attributes('data-mode')).toBe('fixtures')
    expect(w.text()).toContain('Демо на заготовках')
    expect(w.text()).toContain('Сейчас')
  })

  it('в воспроизведении действия выключены', async () => {
    const w = mountFrame({})
    expect(w.find('fieldset.actions').attributes('disabled')).toBeUndefined()
    useMomentStore().travel('2026-09-23T11:05:00.000Z')
    await w.vm.$nextTick()
    expect(w.find('fieldset.actions').attributes('disabled')).toBeDefined()
    expect(w.text()).toContain('Действия недоступны в режиме воспроизведения')
    expect(w.text()).toContain('Как было')
  })
})
