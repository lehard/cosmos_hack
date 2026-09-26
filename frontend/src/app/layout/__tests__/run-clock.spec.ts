// Часы прогона в шапке (Д-85): только пока идёт прогон — «23.09 10:31 · ×60»;
// идут по скорости, в ожидании человека стоят; без прогона и в прошлом — ничего.
// Активный прогон — shared/api/active-run (своего опроса у часов нет).
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { activeRun } from '@/shared/api/active-run'
import { useMomentStore } from '@/shared/model/moment'
import RunClock from '../header/RunClock.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
afterEach(() => (activeRun.value = null))

const run = (state = 'running') => ({ run_id: 'R-1', state, clock_at: '2026-09-23T07:31:00Z', speed: 60 })
const mountClock = () => mount(RunClock, { global: { plugins: [pinia] } })

describe('часы прогона в шапке', () => {
  it('без прогона — шапку не трогают', () => {
    expect(mountClock().find('[data-testid="run-clock"]').exists()).toBe(false)
  })

  it('идёт прогон: доменное время и скорость, время идёт по скорости', async () => {
    activeRun.value = run()
    const w = mountClock()
    expect(w.find('[data-testid="run-clock"]').attributes('data-ticking')).toBe('true')
    expect(w.find('[data-testid="run-clock-time"]').text()).toBe('23.09 10:31')
    expect(w.find('[data-testid="run-clock-speed"]').text()).toBe('×60')
    // Больше текста нет: ни «сценарий ждёт», ни кнопок.
    expect(w.findAll('.run-clock > span').map((x) => x.text())).toEqual(['', '23.09 10:31', '·', '×60'])
    await new Promise((r) => setTimeout(r, 1100))
    expect(w.find('[data-testid="run-clock-time"]').text()).toBe('23.09 10:32')
    w.unmount()
  })

  it('ждёт человека — часы стоят', async () => {
    activeRun.value = run('waiting_for_decision')
    const w = mountClock()
    expect(w.find('[data-testid="run-clock"]').attributes('data-ticking')).toBeUndefined()
    await new Promise((r) => setTimeout(r, 1100))
    expect(w.find('[data-testid="run-clock-time"]').text()).toBe('23.09 10:31')
    w.unmount()
  })

  it('просмотр прошлого — часов нет', () => {
    activeRun.value = run()
    useMomentStore(pinia).travel('2026-09-23T06:00:00Z')
    expect(mountClock().find('[data-testid="run-clock"]').exists()).toBe(false)
  })
})
