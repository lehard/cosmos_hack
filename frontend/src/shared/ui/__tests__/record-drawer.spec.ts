// Правое окно записи (Д-70, UI-7, UI-8): заголовок, вкладки, тело и нижняя
// панель действий; закрытие крестиком и Esc. Окно выводится в body.
import { h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import RecordDrawer from '../RecordDrawer.vue'

const $ = (sel: string) => document.querySelector<HTMLElement>(sel)
afterEach(() => {
  document.body.innerHTML = ''
})

function mountDrawer(props: Record<string, unknown> = {}) {
  return mount(RecordDrawer, {
    attachTo: document.body,
    props: {
      show: true,
      kindLabel: 'Несоответствие',
      number: 'НС-0142',
      subtitle: 'Изделие FL-0042',
      tabs: [
        { id: 'card', label: 'Карточка' },
        { id: 'passport', label: 'Паспорт изделия' },
      ],
      tab: 'card',
      ...props,
    },
    slots: {
      status: () => h('span', { 'data-testid': 'st' }, 'Черновик'),
      default: () => h('p', { 'data-testid': 'content' }, 'тело'),
      actions: () => h('button', { 'data-testid': 'act' }, 'Подтвердить'),
    },
  })
}

describe('окно записи', () => {
  it('заголовок: тип, номер, статус, пояснение; вкладки; тело; кнопки — в нижней панели', async () => {
    const w = mountDrawer()
    await flushPromises()
    const head = $('[data-testid="record-drawer-head"]')!
    expect(head.textContent).toContain('Несоответствие')
    expect(head.textContent).toContain('НС-0142')
    expect(head.querySelector('[data-testid="st"]')).not.toBeNull()
    expect(head.textContent).toContain('Изделие FL-0042')
    const tabs = [...document.querySelectorAll<HTMLElement>('[role="tab"]')]
    expect(tabs.map((x) => x.textContent?.trim())).toEqual(['Карточка', 'Паспорт изделия'])
    expect(tabs[0]!.getAttribute('aria-selected')).toBe('true')
    expect($('[data-testid="record-drawer-body"] [data-testid="content"]')).not.toBeNull()
    // Кнопки действий — не в прокручиваемом теле, а в нижней панели.
    expect($('[data-testid="record-drawer-actions"] [data-testid="act"]')).not.toBeNull()
    expect($('[data-testid="record-drawer-body"] [data-testid="act"]')).toBeNull()

    tabs[1]!.click()
    expect(w.emitted('update:tab')?.[0]).toEqual(['passport'])
    w.unmount()
  })

  it('одна вкладка — полоски вкладок нет; без действий — нижней панели нет', async () => {
    const w = mount(RecordDrawer, { attachTo: document.body, props: { show: true, kindLabel: 'Изделие', number: 'FL-0042', tabs: [{ id: 'p', label: 'Паспорт' }] } })
    await flushPromises()
    expect($('[role="tablist"]')).toBeNull()
    expect($('[data-testid="record-drawer-actions"]')).toBeNull()
    w.unmount()
  })

  it('закрытие: крестик и Esc сообщают close', async () => {
    const w = mountDrawer()
    await flushPromises()
    $('.n-base-close')!.click()
    expect(w.emitted('close')).toHaveLength(1)
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', code: 'Escape', bubbles: true }))
    expect(w.emitted('close')).toHaveLength(2)
    w.unmount()
  })
})
