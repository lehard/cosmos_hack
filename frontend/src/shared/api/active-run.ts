/**
 * Активный прогон сценария (показ): пока на пульте идёт прогон, все столы работают в нём
 * без `?run=` в адресе — вошёл и сразу видишь процесс. Чтения `/api/v1/*` без `run_id`
 * получают `run_id` активного прогона; явный `run_id` (адрес `?run=`, срез стола) главнее.
 * Список прогонов, сеанс и справочники пульта не трогаются.
 */
import { ref } from 'vue'

export interface ActiveRun {
  run_id: string
  scenario_id?: string
  state: string
  step?: number
  steps?: number
  step_title?: string
  clock_at?: string
  /** Начало прогона: тот же run_id после пересоздания стенда — другой прогон. */
  started_at?: string
  speed?: number
  waiting_for?: { role: string; action: string; object_id?: string; title?: string } | null
}

const ACTIVE_STATES = new Set(['running', 'paused', 'waiting_for_decision'])
// Задачи — без run_id: задачи процесса от действий людей в интерфейсе создаются без run_id (фильтр их прятал).
const SKIP = [/^\/api\/v1\/runs(\/|$)/, /^\/api\/v1\/scenarios(\/|$)/, /^\/api\/v1\/auth(\/|$)/, /^\/api\/v1\/tasks(\/|$)/]

export const activeRun = ref<ActiveRun | null>(null)

let original: typeof fetch | null = null

/** Оборачивает fetch один раз при старте приложения. */
export function installRunFetch(): void {
  if (original) return
  original = window.fetch.bind(window)
  window.fetch = (input: RequestInfo | URL, init?: RequestInit) => {
    const runId = activeRun.value?.run_id
    const method = (init?.method ?? (input instanceof Request ? input.method : 'GET')).toUpperCase()
    if (runId && method === 'GET' && !(input instanceof Request)) {
      const url = new URL(String(input), window.location.origin)
      if (url.pathname.startsWith('/api/v1/') && !SKIP.some((r) => r.test(url.pathname)) && !url.searchParams.has('run_id')) {
        url.searchParams.set('run_id', runId)
        const same = url.origin === window.location.origin
        return original!(same ? url.pathname + url.search : url.toString(), init)
      }
    }
    return original!(input, init)
  }
}

/** Опрос активного прогона: первый прогон в состоянии «идёт / пауза / ждёт решения». */
export async function refreshActiveRun(): Promise<ActiveRun | null> {
  const f = original ?? window.fetch
  try {
    const res = await f('/api/v1/runs', { credentials: 'include' })
    if (!res.ok) return activeRun.value
    const body = (await res.json()) as { items?: ActiveRun[] }
    const run = (body.items ?? []).find((r) => ACTIVE_STATES.has(r.state)) ?? null
    activeRun.value = run
    return run
  } catch {
    return activeRun.value
  }
}
