// Полоса времени: «к следующему событию» и пояснение оси «что мы знали» / «как было».
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import TimelineBar from '../ui/TimelineBar.vue'

const range = { from: Date.parse('2026-09-23T08:00:00Z'), to: Date.parse('2026-09-23T12:00:00Z') }
const marks = [
  { mark_id: 'm1', at: '2026-09-23T08:21:00Z', kind: 'spike', title: 'ИС-2 вышел за уставку' },
  { mark_id: 'm2', at: '2026-09-23T10:04:00Z', kind: 'revision', title: 'Журнал ИС-2 пришёл с опозданием' },
] as never

const mountBar = (props: Record<string, unknown>) =>
  mount(TimelineBar, { props: { range, marks, asOf: null, axis: 'occurred', playing: false, speed: 60, ...props }, global: { plugins: [createPinia(), i18n] } })

describe('полоса времени', () => {
  it('в проигрывании — «к следующему событию»: переход к ближайшей отметке после текущего момента', async () => {
    const w = mountBar({ asOf: '2026-09-23T09:00:00Z' })
    const next = w.find('[data-action="next-mark"]')
    expect(next.exists()).toBe(true)
    await next.trigger('click')
    expect(w.emitted('jump')?.[0]).toEqual([Date.parse('2026-09-23T10:04:00Z')])
  })

  it('«сейчас» — без кнопки следующего события и без пояснения оси', () => {
    const w = mountBar({})
    expect(w.find('[data-action="next-mark"]').exists()).toBe(false)
    expect(w.find('[data-testid="axis-note"]').exists()).toBe(false)
  })

  it('ось словами: «что мы знали» — только пришедшее к моменту; «как было» — с поздними данными', () => {
    expect(mountBar({ asOf: '2026-09-23T09:00:00Z', axis: 'recorded' }).find('[data-testid="axis-note"]').text()).toContain('только данные, пришедшие к этому моменту')
    expect(mountBar({ asOf: '2026-09-23T09:00:00Z', axis: 'occurred' }).find('[data-testid="axis-note"]').text()).toContain('с учётом данных, пришедших позже')
  })
})
