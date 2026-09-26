// Левое меню разделов (Д-73): развёрнуто — значок и подпись; свёрнуто — только
// значки, подпись для чтения с экрана; выбранный пункт помечен; сворачивание —
// кнопкой внизу, скрыть меню совсем нельзя.
import { h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SideNav from '../SideNav.vue'

const Icon = { render: () => h('svg') }
const items = [
  { id: 'overview', label: 'Обзор', icon: Icon, hint: 'Alt+1' },
  { id: 'analytics', label: 'Аналитика качества по участкам и операциям', icon: Icon, hint: 'Alt+2' },
]
const mountNav = (collapsed: boolean) =>
  mount(SideNav, { props: { items, active: 'analytics', collapsed, label: 'Разделы стола', collapseLabel: 'Свернуть меню', expandLabel: 'Развернуть меню' } })

describe('левое меню', () => {
  it('развёрнуто: подписи с многоточием, выбранный пункт, выбор щелчком', async () => {
    const w = mountNav(false)
    const buttons = w.findAll('[data-item]')
    expect(buttons.map((b) => b.text())).toEqual(['Обзор', 'Аналитика качества по участкам и операциям'])
    expect(buttons[1]!.find('.text').classes()).toContain('ant-ellipsis')
    expect(buttons[1]!.attributes('aria-current')).toBe('page')
    expect(buttons[0]!.attributes('aria-current')).toBeUndefined()
    await buttons[0]!.trigger('click')
    expect(w.emitted('select')?.[0]).toEqual(['overview'])
    expect(w.find('[data-testid="side-nav-toggle"]').text()).toBe('Свернуть меню')
  })

  it('свёрнуто: только значки, подпись — для чтения с экрана; кнопка «развернуть»', async () => {
    const w = mountNav(true)
    expect(w.find('nav').attributes('data-collapsed')).toBe('true')
    expect(w.find('[data-item="overview"] .text').exists()).toBe(false)
    expect(w.find('[data-item="overview"]').attributes('aria-label')).toBe('Обзор')
    const toggle = w.find('[data-testid="side-nav-toggle"]')
    expect(toggle.attributes('title')).toBe('Развернуть меню')
    await toggle.trigger('click')
    expect(w.emitted('update:collapsed')?.[0]).toEqual([false])
  })
})
