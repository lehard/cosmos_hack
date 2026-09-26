// Живая карта на настоящем bpmn-js (happy-dom + заглушки SVG): наложения по
// step_key, клик по точке → паспорт, клик по узлу → карточка с documentation,
// ограничение линии, режим инцидента и проигрывание главной истории (FR-2, 5, 7,
// 9, 154, 155).
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { LiveMapData } from '@/entities/live-map'
import { i18n } from '@/shared/i18n'
import LiveMapView from '../ui/LiveMapView.vue'
import BpmnMapViewer from '../ui/BpmnMapViewer.vue'
import { frameMorning, frameScope13, frameScope34, frameScope6, storyFrames, V0 } from './fixtures'
import { installSvgStubs } from './svg-env'

installSvgStubs()
// bpmn-js открывает схему на 300 узлов — под общей нагрузкой прогона это секунды.
vi.setConfig({ testTimeout: 30_000 })
const WAIT = { timeout: 15_000 }

let wrapper: VueWrapper | null = null
beforeEach(() => setActivePinia(createPinia()))
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  document.body.innerHTML = ''
})

async function mountMap(data: LiveMapData) {
  wrapper = mount(LiveMapView, { props: { data, period: 'shift' }, attachTo: document.body, global: { plugins: [i18n] } })
  await vi.waitFor(() => expect(document.querySelector('[data-step="welding.zt3_acceptance"]')).not.toBeNull(), WAIT)
  return wrapper
}

const dot = (itemId: string) => document.querySelector<HTMLElement>(`.dot[data-item="${itemId}"]`)

describe('живая карта', () => {
  it('счётчики на узлах по step_key и изделия-точки (FR-2)', async () => {
    await mountMap(frameMorning())
    const badge = document.querySelector('.node-badge[data-step="welding.zt3_acceptance"]')!
    expect(badge.querySelector('[data-counter="queue"]')!.textContent).toBe('12')
    expect(badge.querySelector('[data-counter="passed"]')!.textContent).toBe('22')
    // 12 изделий на ЗТ-3: 6 точек и «+6».
    const dots = document.querySelector('.node-dots[data-step="welding.zt3_acceptance"]')!
    expect(dots.querySelectorAll('.dot')).toHaveLength(6)
    expect(dots.querySelector('.more')!.textContent).toBe('+6')
    // Изделие прежней версии на этой карте не показано, но есть заметка (FR-1).
    expect(dot('ENT:FL-0090')).toBeNull()
    expect(norm(document.querySelector('[data-testid="other-versions"]')!.textContent)).toContain('2 изделия')
    // Цех: изделия в дорожке (FR-130).
    expect(norm(document.querySelector('[data-lane="Lane_WC"]')!.textContent)).toContain('14 изделий')
    // Перемещение между цехами — отдельный вид точки.
    expect(dot('ENT:FL-0042')!.classList.contains('transit')).toBe(true)
  })

  it('клик по точке открывает паспорт изделия (FR-2, FR-7)', async () => {
    const w = await mountMap(frameMorning())
    dot('ENT:FL-0041')!.click()
    expect(w.emitted('open-item')).toEqual([['ENT:FL-0041']])
  })

  it('ограничение линии подсвечено на фигуре и подписано (FR-5)', async () => {
    await mountMap(frameMorning())
    expect(document.querySelector('.djs-element[data-element-id="W5"]')!.classList.contains('ant-bottleneck')).toBe(true)
    expect(document.querySelector('.node-badge[data-step="welding.zt3_acceptance"] [data-flag="bottleneck"]')!.getAttribute('title')).toContain('37 мин')
  })

  it('клик по узлу — правое окно (Д-70): шаг в заголовке, описание и счётчики, внизу — изделия узла (FR-154)', async () => {
    const w = await mountMap(frameScope34())
    const viewer = w.findComponent(BpmnMapViewer)
    viewer.vm.$emit('select-node', 'welding.kt3_camera')
    await flushPromises()
    const card = document.querySelector('[data-record="node"] .node-card[data-step="welding.kt3_camera"]')!
    expect(document.querySelector('[data-record="node"] [data-testid="record-drawer-head"]')!.textContent).toContain('КТ-3 Камера на шов (участки 40 мм)')
    expect(card.querySelector('[data-testid="node-doc"]')!.textContent!.length).toBeGreaterThan(40)
    expect(card.querySelector('[data-counter="defects"]')!.textContent).toBe('1')
    expect(norm(card.querySelector('[data-counter="nonconformities"]')!.textContent)).toBe('1 несоответствие')
    document.querySelector<HTMLElement>('[data-record="node"] [data-testid="record-drawer-actions"] [data-action="open-node"]')!.click()
    expect(w.emitted('open-node')).toEqual([['welding.kt3_camera']])
    // Выбранный узел помечен на схеме.
    expect(document.querySelector('.djs-element[data-element-id="W3"]')!.classList.contains('ant-selected')).toBe(true)
  })

  it('клик по фигуре узла на холсте выбирает узел', async () => {
    const w = await mountMap(frameMorning())
    const gfx = document.querySelector('.djs-element[data-element-id="W2"]')!
    gfx.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(document.querySelector('.node-card[data-step="welding.weld"]')).not.toBeNull()
    expect(w.emitted('open-item')).toBeUndefined()
  })

  it('режим инцидента: цвета по статусу в инциденте, легенда, версия области (FR-9)', async () => {
    await mountMap(frameScope34())
    expect(document.querySelector('[data-testid="scope-reduction"]')!.textContent).toContain('34 → 34')
    expect(document.querySelectorAll('[data-legend]')).toHaveLength(4)
    expect(document.querySelector('[data-testid="incident"]')!.textContent).toContain('Цвет — статус относительно этого инцидента')
    // Подтверждённое — первой точкой узла, даже если пришло не первым.
    const first = document.querySelector<HTMLElement>('.node-dots[data-step="welding.zt3_acceptance"] .dot')!
    expect(first.dataset.item).toBe('ENT:FL-0001')
    expect(first.dataset.tone).toBe('danger')
    expect(dot('ENT:FL-0002')!.dataset.tone).toBe('attention')
    // Изделие вне области — приглушено, а не «серое = нет данных».
    expect(dot('ENT:FL-0041')!.classList.contains('dimmed')).toBe(true)
  })

  it('главная история на карте: область тает 34 → 13 → 6 (FR-155)', async () => {
    const [morning, ...rest] = storyFrames()
    const w = await mountMap(morning!)
    expect(document.querySelector('[data-testid="incident"]')).toBeNull()
    const seen: string[] = []
    for (const frame of rest) {
      await w.setProps({ data: frame })
      await flushPromises()
      seen.push(document.querySelector('[data-testid="scope-reduction"]')!.textContent!.split('·')[0]!.trim())
    }
    expect(seen).toEqual(['Область сокращена: 34 → 34', 'Область сокращена: 34 → 13', 'Область сокращена: 34 → 6'])
    expect(document.querySelector('[data-testid="scope-version"]')!.textContent).toBe('Версия области 3')
    expect(dot('ENT:FL-0027')!.dataset.tone).toBe('success')
    expect(dot('ENT:FL-0005')!.dataset.tone).toBe('neutral')
  })

  it('смена версии — схема и точки той версии (FR-1)', async () => {
    const w = await mountMap(frameMorning())
    const f = frameMorning()
    await w.setProps({ data: { ...f, process_version: f.versions[1]! } })
    await vi.waitFor(() => expect(dot('ENT:FL-0090')).not.toBeNull(), WAIT)
    expect(dot('ENT:FL-0041')).toBeNull()
    expect(w.find('[data-version]').attributes('data-version')).toBe(V0)
  })

  it('битый XML — сообщение, а не пустой холст', async () => {
    wrapper = mount(LiveMapView, { props: { data: { ...frameScope13(), bpmn_xml: '<nope' }, period: 'shift' }, attachTo: document.body, global: { plugins: [i18n] } })
    await vi.waitFor(() => expect(document.querySelector('.import-error')).not.toBeNull(), WAIT)
  })

  it('смена периода уходит наружу (FR-3)', async () => {
    const w = await mountMap(frameScope6())
    await w.find('[data-period="week"] input').setValue(true)
    expect(w.emitted('update:period')?.at(-1)).toEqual(['week'])
  })
})

/** Неразрывные пробелы текстов → обычные. */
const norm = (s: string | null | undefined) => (s ?? '').replace(/\u00a0/g, ' ')
