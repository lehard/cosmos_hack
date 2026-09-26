/**
 * План прогона на пульте (Д-85): «Сейчас ждём: ‹роль› — ‹действие›
 * (‹изделие›)» и ближайшие события с доменным временем — решения людей с
 * остановками, события машин и внешних систем, запланированные сбои (видны
 * заранее: «на 2-й сварке ИС-2 уйдёт из уставки»).
 *
 * Операция `simulation.run.plan` — GET /api/v1/runs/{run_id}/plan (ветка
 * агента симуляции fix/scenario-start-step).
 * TODO(после слияния fix/scenario-start-step в main): типы RunPlan, PlanEntry,
 * PlanEntryKind и функцию simulationRunPlan брать из сгенерированного клиента
 * (`@/shared/api/generated`), локальные ниже — удалить. Имена полей — те же.
 */
import type { RunState, RunWait } from '@/shared/api/generated/model'

/** Вид строки плана: решение человека, событие машины или внешней системы, запланированный сбой, служебное. */
export type PlanEntryKind = 'decision' | 'event' | 'alert' | 'stand' | 'tamper'

/** Строка плана прогона (контракт PlanEntry). */
export interface PlanEntry {
  /** Доменное время по плану; пока прогон ждёт человека, время следующих строк не «убегает». */
  at: string
  /** Конец группы однотипных событий (сводки тока по минутам). */
  until?: string
  kind: PlanEntryKind
  /** Что произойдёт, по-русски. */
  title: string
  /** Прогон ждёт нажатия человека на его столе. */
  stop: boolean
  /** Роль стола решения. */
  role?: string
  /** Персона (псевдоним) решения. */
  persona?: string
  /** operationId решения (x-ant-action). */
  operation?: string
  /** Изделие прогона. */
  item_id?: string
  /** Объект решения, если уже известен. */
  object_id?: string
  /** Метка шага определения. */
  label?: string
  /** Уже произошло. */
  done: boolean
  /** Прогон ждёт именно этого решения сейчас. */
  waiting: boolean
}

/** План прогона (контракт RunPlan). */
export interface RunPlan {
  run_id: string
  state: RunState
  /** Доменное «сейчас» прогона. */
  clock_at: string
  /** Ускорение ×1…×1000. */
  speed: number
  /** Начало живой части (сценарий показа): раньше — готовая история. */
  live_from?: string
  /** Чьё решение ждёт прогон сейчас. */
  waiting_for?: RunWait
  items: PlanEntry[]
}

/** Параметры `simulation.run.plan`. */
export interface RunPlanParams {
  /** Весь план (прошедшее — done); по умолчанию — от текущего места. */
  all?: boolean
  /** Сколько строк (по умолчанию 50). */
  limit?: number
}

/**
 * `simulation.run.plan` — по образцу сгенерированного клиента: ответ
 * `{ data, status, headers }`, отказ — ошибка с `info` (problem+json) и `status`.
 */
export async function simulationRunPlan(runId: string, params: RunPlanParams = {}, options?: RequestInit) {
  const q = new URLSearchParams()
  if (params.all !== undefined) q.append('all', String(params.all))
  if (params.limit !== undefined) q.append('limit', String(params.limit))
  const qs = q.toString()
  // Временно (AD-20): операции ещё нет в сгенерированном клиенте main — см. TODO вверху.
  // eslint-disable-next-line no-restricted-globals
  const res = await fetch(`/api/v1/runs/${encodeURIComponent(runId)}/plan${qs ? `?${qs}` : ''}`, { ...options, method: 'GET' })
  const body = [204, 205, 304].includes(res.status) ? null : await res.text()
  if (!res.ok) {
    const err: Error & { info?: unknown; status?: number } = new Error()
    err.info = body ? (JSON.parse(body) as unknown) : {}
    err.status = res.status
    throw err
  }
  const data = (body ? JSON.parse(body) : {}) as RunPlan
  return { data, status: res.status, headers: res.headers }
}

/** Что ждём сейчас: строка плана с `waiting`, иначе ожидание прогона. */
export interface PlanWait {
  role: string
  /** Действие словами. */
  what: string
  /** Изделие или объект решения. */
  object?: string
  persona?: string
  at?: string
}

/**
 * «Сейчас ждём»: строка плана с `waiting` (в ней изделие и персона), иначе
 * `waiting_for` прогона. Не ждём ничего — null.
 */
export function planWait(plan: Pick<RunPlan, 'items' | 'waiting_for'>): PlanWait | null {
  const e = plan.items.find((x) => x.waiting)
  const w = plan.waiting_for
  if (e && (e.role || w?.role)) {
    return {
      role: e.role ?? w!.role,
      what: e.title || w?.title || e.operation || w?.action || '',
      object: e.item_id || e.object_id || w?.object_id || undefined,
      persona: e.persona,
      at: e.at,
    }
  }
  if (w) return { role: w.role, what: w.title || w.action, object: w.object_id || undefined }
  return null
}

/** Ближайшие строки плана: ещё не случившиеся, в порядке сервера (он — по доменному времени). */
export const upcoming = (plan: Pick<RunPlan, 'items'>, limit = 12): PlanEntry[] => plan.items.filter((e) => !e.done).slice(0, limit)

/** Запланированные сбои впереди — видны заранее, даже если за пределами ближайших строк. */
export const plannedAlerts = (plan: Pick<RunPlan, 'items'>): PlanEntry[] => plan.items.filter((e) => e.kind === 'alert' && !e.done)
