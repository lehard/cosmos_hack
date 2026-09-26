// Слой «нормы» на живой карте (FR-156, эпик 39): переключатель подсвечивает
// узлы с нормативной опорой; по клику на ЗТ-3 — пункт ГОСТ и что система
// делает на шаге.
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { i18n } from '@/shared/i18n'
import LiveMapView from '../ui/LiveMapView.vue'
import BpmnMapViewer from '../ui/BpmnMapViewer.vue'
import { frameMorning } from './fixtures'
import { installSvgStubs } from './svg-env'

installSvgStubs()
vi.setConfig({ testTimeout: 30_000 })
const WAIT = { timeout: 15_000 }

let wrapper: VueWrapper | null = null
beforeEach(() => setActivePinia(createPinia()))
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  document.body.innerHTML = ''
})

describe('слой «нормы»', () => {
  it('подсветка узлов с опорой и опоры ЗТ-3 по клику', async () => {
    wrapper = mount(LiveMapView, { props: { data: frameMorning(), period: 'shift' }, attachTo: document.body, global: { plugins: [i18n] } })
    await vi.waitFor(() => expect(document.querySelector('[data-step="welding.zt3_acceptance"]')).not.toBeNull(), WAIT)
    expect(document.querySelectorAll('.djs-element.ant-norm')).toHaveLength(0)

    document.querySelector<HTMLElement>('[data-testid="norms-toggle"]')!.click()
    await flushPromises()
    await vi.waitFor(() => expect(document.querySelectorAll('.djs-element.ant-norm').length).toBe(33), WAIT)
    expect(document.querySelector('[data-testid="norms-hint"]')!.textContent).toContain('33')

    wrapper.findComponent(BpmnMapViewer).vm.$emit('select-node', 'welding.zt3_acceptance')
    await flushPromises()
    const panel = document.querySelector('[data-testid="norms-panel"]')!
    expect(panel.textContent).toContain('ГОСТ')
    expect(panel.textContent).toContain('Что делает система')
  })
})
