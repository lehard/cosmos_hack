// Каждый стол normative/desks/*.yaml собирается оболочкой из реестра без кода под
// роль (AD-21, NFR-EXT-1): все слоты на месте, в них — виджеты из реестра.
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { parse } from 'yaml'
import { LAYOUT_AREAS, type Desk } from '@/entities/desk'
import { i18n } from '@/shared/i18n'
import DeskTabView from '../ui/DeskTabView.vue'

const DESKS = resolve(__dirname, '../../../../../normative/desks')
const files = readdirSync(DESKS).filter((f) => f.endsWith('.yaml'))

async function render(tab: Desk['tabs'][number], density: Desk['density']) {
  const w = mount(DeskTabView, { props: { tab, density }, global: { plugins: [createPinia(), i18n] } })
  await vi.dynamicImportSettled()
  await flushPromises()
  return w
}

describe('столы ролей из yaml', () => {
  it('есть столы всех ролей PRD §3a', () => {
    const roles = files.map((f) => f.replace('.yaml', '')).sort()
    expect(roles).toEqual(
      ['administrator', 'approver', 'customer_representative', 'performer', 'production_manager', 'quality_inspector', 'security_auditor', 'site_foreman', 'technologist'].sort(),
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
          expect(el.attributes('data-state')).toBe('normal')
        }
      })
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
