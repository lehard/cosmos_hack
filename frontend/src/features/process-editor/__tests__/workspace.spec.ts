// Рабочее место схемы на весь экран (UI-34): правка — с меткой и «Сохранить черновик»,
// просмотр — без правки; закрыть с несохранёнными правками — только после подтверждения.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { i18n } from '@/shared/i18n'

// bpmn-js в jsdom не рисует — модельер подменён заглушкой с тем же API.
vi.mock('../ui/ProcessModeler.vue', () => ({
  default: defineComponent({
    props: { xml: String, editable: Boolean, view: String },
    emits: ['dirty'],
    setup(props, { emit, expose }) {
      expose({ saveXML: async () => '<bpmn/>', fit: () => undefined })
      return () => h('div', { 'data-testid': 'modeler-stub', 'data-view': props.view, 'data-editable': String(props.editable), onClick: () => emit('dirty') })
    },
  }),
}))

const { default: ProcessWorkspace } = await import('../ui/ProcessWorkspace.vue')

const mountWs = (props: Record<string, unknown> = {}) =>
  mount(ProcessWorkspace, {
    props: { show: true, xml: '<x/>', title: 'Кронштейн', versionLabel: 'v1', statusText: 'Действующая', editable: true, label: 'v2', ...props },
    global: { plugins: [createPinia(), i18n] },
    attachTo: document.body,
  })

describe('рабочее место схемы', () => {
  it('правка: схема — вид рабочего места, сохранить — после изменений, с меткой', async () => {
    const w = mountWs()
    const body = () => document.body
    expect(body().querySelector('[data-testid="modeler-stub"]')?.getAttribute('data-view')).toBe('workspace')
    expect(body().querySelector('[data-testid="modeler-stub"]')?.getAttribute('data-editable')).toBe('true')
    const save = () => body().querySelector('[data-action="saveDraft"]') as HTMLButtonElement
    expect(save().disabled).toBe(true)
    ;(body().querySelector('[data-testid="modeler-stub"]') as HTMLElement).click()
    await w.vm.$nextTick()
    expect(save().disabled).toBe(false)
    save().click()
    await new Promise((r) => setTimeout(r, 0))
    expect(w.emitted('save')?.[0]).toEqual(['<bpmn/>', 'v2'])
    w.unmount()
  })

  it('просмотр: без метки и без «Сохранить»; «Закрыть» закрывает сразу', async () => {
    const w = mountWs({ editable: false })
    expect(document.body.querySelector('[data-action="saveDraft"]')).toBeNull()
    expect(document.body.querySelector('[data-testid="workspace-label"]')).toBeNull()
    ;(document.body.querySelector('[data-action="close"]') as HTMLButtonElement).click()
    expect(w.emitted('close')).toHaveLength(1)
    w.unmount()
  })
})
