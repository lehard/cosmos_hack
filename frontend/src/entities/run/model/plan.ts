/**
 * План прогона на пульте (Д-85): «Сейчас ждём: ‹роль› — ‹действие›
 * (‹изделие›)» и ближайшие события с доменным временем — решения людей с
 * остановками, события машин и внешних систем, запланированные сбои (видны
 * заранее: «на 2-й сварке ИС-2 уйдёт из уставки»). Операция
 * `simulation.run.plan` — сгенерированным клиентом; здесь — чистые правила.
 */
import type { PlanEntry, PlanEntryKind, RunPlan, SimulationRunPlanParams } from '@/shared/api/generated/model'

export type { PlanEntry, PlanEntryKind, RunPlan, SimulationRunPlanParams }

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
