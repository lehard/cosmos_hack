// Часы прогона в шапке всех столов (Д-85): «23.09 10:31 · ×60», идут между
// ответами сервера; в ожидании человека стоят и сказано, чью роль ждём; без
// прогона и при просмотре прошлого — ничего.
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mountWidget, settle } from '@/entities/run/__tests__/api'
import { idleRun, run } from '@/entities/run/__tests__/fixtures'
import type { Run } from '@/shared/api/generated/model'
import { useMomentStore } from '@/shared/model/moment'
import RunClock from '../header/RunClock.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

/** Сервер: список прогонов и чтение прогона; режим ответа — live или fixtures. */
function serve(r: Run, mode: 'live' | 'fixtures' = 'live') {
  const t0 = Date.now()
  // Идущий прогон: сервер отдаёт часы, сдвинутые по скорости с начала теста.
  const now = (): Run => (r.state === 'running' && mode === 'live' ? { ...r, clock_at: new Date(Date.parse(r.clock_at) + (Date.now() - t0) * r.speed).toISOString() } : r)
  vi.stubGlobal('fetch', async (url: string) => {
    const path = new URL(url, 'http://ant.local').pathname
    const body = path === '/api/v1/runs' ? { items: [now()] } : path === `/api/v1/runs/${r.run_id}` ? now() : null
    if (!body) return new Response('{}', { status: 501 })
    return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json', 'Ant-Backend': mode } })
  })
}

describe('часы прогона в шапке', () => {
  it('идущий прогон: доменное время и скорость; время идёт по скорости', async () => {
    serve(run({ speed: 60 }))
    const w = await mountWidget(RunClock, {}, pinia)
    const clock = w.find('[data-testid="run-clock"]')
    expect(clock.exists()).toBe(true)
    expect(clock.attributes('data-ticking')).toBe('true')
    expect(w.find('[data-testid="run-clock-time"]').text()).toMatch(/^23\.09 10:3[12]$/)
    expect(w.find('[data-testid="run-clock-speed"]').text()).toBe('×60')
    expect(w.find('[data-testid="run-clock-state"]').exists()).toBe(false)
    // Две реальные секунды при ×60 — ещё две доменные минуты.
    const before = w.find('[data-testid="run-clock-time"]').text()
    await new Promise((r) => setTimeout(r, 2100))
    await settle()
    const after = w.find('[data-testid="run-clock-time"]').text()
    expect(after).not.toBe(before)
    w.unmount()
  })

  it('ждёт решения — часы стоят, «стоим: ждём — Мастер участка»', async () => {
    serve(run({ state: 'waiting_for_decision', waiting_for: { role: 'site_foreman', action: 'process.movement.receive', object_id: 'Ф-001', title: 'принять Ф-001' } }))
    const w = await mountWidget(RunClock, {}, pinia)
    expect(w.find('[data-testid="run-clock"]').attributes('data-ticking')).toBeUndefined()
    expect(w.find('[data-testid="run-clock-time"]').text()).toBe('23.09 10:31')
    expect(w.find('[data-testid="run-clock-state"]').text()).toBe('стоим: ждём — Мастер участка')
    expect(w.find('[data-testid="run-clock"]').attributes('aria-label')).toContain('Ждём: Мастер участка — принять Ф-001')
    w.unmount()
  })

  it('заготовки: часы мира двигает пульт — не досчитываются', async () => {
    serve(run(), 'fixtures')
    const w = await mountWidget(RunClock, {}, pinia)
    expect(w.find('[data-testid="run-clock"]').attributes('data-ticking')).toBeUndefined()
    expect(w.find('[data-testid="run-clock-time"]').text()).toBe('23.09 10:31')
    w.unmount()
  })

  it('прогон не запущен — часов нет', async () => {
    serve(idleRun())
    const w = await mountWidget(RunClock, {}, pinia)
    expect(w.find('[data-testid="run-clock"]').exists()).toBe(false)
    w.unmount()
  })

  it('просмотр прошлого — часов прогона нет (время показывает «На момент»)', async () => {
    serve(run())
    useMomentStore(pinia).travel('2026-09-23T06:00:00Z')
    const w = await mountWidget(RunClock, {}, pinia)
    expect(w.find('[data-testid="run-clock"]').exists()).toBe(false)
    w.unmount()
  })
})
