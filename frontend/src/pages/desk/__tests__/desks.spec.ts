// Каждый стол normative/desks/*.yaml собирается оболочкой из реестра без кода под
// роль (AD-21, NFR-EXT-1): все слоты на месте, в них — виджеты из реестра.
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { parse } from 'yaml'
import { LAYOUT_AREAS, type Desk } from '@/entities/desk'
import { i18n } from '@/shared/i18n'
import DeskTabView from '../ui/DeskTabView.vue'

const DESKS = resolve(__dirname, '../../../../../normative/desks')
const files = readdirSync(DESKS).filter((f) => f.endsWith('.yaml'))

async function render(tab: Desk['tabs'][number], density: Desk['density']) {
  // Наполненные виджеты читают сервер через Vue Query и ведут в детали роутером.
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
  // Сервера нет: наполненные виджеты получают problem+json, а не обрыв соединения.
  vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ type: 'urn:ant:problem:api.not_implemented', title: 'нет сервера', status: 501, code: 'api.not_implemented' }), { status: 501, headers: { 'Content-Type': 'application/problem+json' } })))
  const w = mount(DeskTabView, { props: { tab, density }, global: { plugins: [createPinia(), i18n, router, [VueQueryPlugin, { queryClient }]] } })
  await vi.dynamicImportSettled()
  await flushPromises()
  return w
}

describe('столы ролей из yaml', () => {
  it('есть столы всех ролей PRD §3a', () => {
    const roles = files.map((f) => f.replace('.yaml', '')).sort()
    expect(roles).toEqual(
      ['administrator', 'approver', 'customer_representative', 'head_of_qc', 'performer', 'production_manager', 'quality_inspector', 'security_auditor', 'site_foreman', 'technologist'].sort(),
    )
  })

  for (const f of files) {
    const desk = parse(readFileSync(join(DESKS, f), 'utf8')) as Desk
    for (const tab of desk.tabs) {
      it(`${f} → вкладка «${tab.id}»: все слоты с виджетами из реестра`, async () => {
        const w = await render(tab, desk.density)
        for (const slot of tab.slots) {
          const el = w.find(`[data-slot="${slot.id}"]`)
          expect(el.exists(), `слот ${slot.id}`).toBe(true)
          expect(el.attributes('data-widget')).toBe(slot.widget)
          // Состояние данных — дело виджета (без сервера наполненный виджет в «ошибке
          // входа»); здесь важно, что это виджет из реестра, а не рамка «неизвестный виджет».
          expect(el.attributes('data-state')).toBeDefined()
          expect(el.text()).not.toContain('стол роли ссылается на то, чего нет в реестре')
        }
        // Вкладка overview стола руководителя монтирует живую карту (bpmn-js) —
        // под нагрузкой машины сборки дольше 5 с по умолчанию.
      }, 30_000)
    }
  }

  it('неизвестный виджет — рамка «ошибка входа», а не пустое место', async () => {
    const w = await render({ id: 't', title_key: 'desks.dashboard', layout: 'single', slots: [{ id: 's', area: 'main', widget: 'no-such-widget' }] }, 'comfortable')
    expect(w.find('[data-slot="s"]').attributes('data-state')).toBe('input_error')
    expect(w.text()).toContain('no-such-widget')
  })

  it('таблица областей раскладок совпадает со схемой стола', () => {
    const schema = JSON.parse(readFileSync(join(DESKS, 'desk.schema.json'), 'utf8'))
    const fromSchema = Object.fromEntries(
      schema.definitions.tab.allOf.map((r: { if: { properties: { layout: { const: string } } }; then: { properties: { slots: { items: { properties: { area: { enum: string[] } } } } } } }) => [
        r.if.properties.layout.const,
        r.then.properties.slots.items.properties.area.enum,
      ]),
    )
    expect(fromSchema).toEqual(LAYOUT_AREAS)
  })
})
