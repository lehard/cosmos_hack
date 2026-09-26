import { afterEach, describe, expect, it, vi } from 'vitest'
import { activeRun, headMode, refreshActiveRun } from '../active-run'

const run = { run_id: 'show-is2-20260921-1', state: 'waiting_for_decision' }

describe('активный прогон и режим «голова»', () => {
  afterEach(() => {
    window.history.replaceState(null, '', '/')
    window.localStorage.clear()
    vi.unstubAllGlobals()
    activeRun.value = null
  })

  it('без ?head — подхватывает прогон «ждёт решения»', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ items: [run] }))))
    expect(await refreshActiveRun()).toEqual(run)
    expect(activeRun.value?.run_id).toBe(run.run_id)
  })

  it('?head=1 — прогон не подхватывается и режим запоминается; ?head=0 — возвращается', async () => {
    const f = vi.fn(async () => new Response(JSON.stringify({ items: [run] })))
    vi.stubGlobal('fetch', f)
    window.history.replaceState(null, '', '/?head=1')
    expect(await refreshActiveRun()).toBeNull()
    expect(f).not.toHaveBeenCalled()
    window.history.replaceState(null, '', '/desk')
    expect(headMode()).toBe(true)
    window.history.replaceState(null, '', '/?head=0')
    expect(headMode()).toBe(false)
    expect((await refreshActiveRun())?.run_id).toBe(run.run_id)
  })
})
