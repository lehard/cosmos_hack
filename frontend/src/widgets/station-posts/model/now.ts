/**
 * «Сейчас» мастера участка (разбор стола мастера): что требует действия — открытые
 * задачи, несостыковки цифрового и физического (изолировано в системе, физически
 * не перемещено; по графику на месте — ключ не вставлен / владельца нет в зоне),
 * посты без исполнителя; сводка смены одной строкой. Ничего не досчитывается за
 * сервер: задачи, тревоги, присутствие и счётчики — готовые.
 */
import type { AlertEntry } from '@/entities/notification'
import type { TaskEntry } from '@/entities/task'
import type { PostRow } from '@/shared/api/generated/model'
import type { DrillRef } from '@/shared/model/drill'
import type { StationStep } from './station'

/** Вид строки «требует действий». */
export type NowActionKind = 'task' | 'not_moved' | 'presence' | 'unassigned'

export interface NowAction {
  key: string
  kind: NowActionKind
  tone: 'danger' | 'warn'
  /** Готовый текст (задача, пост) или параметры для текста. */
  title?: string
  item?: string
  post?: string
  person?: string
  presence?: PostRow['presence']
  since?: string | null
  ref?: DrillRef | null
  workplaceId?: string
}

const MISMATCH = new Set<PostRow['presence']>(['key_missing', 'owner_absent', 'absent'])

/**
 * Строки «требует действий» по важности: несостыковки изоляции, просроченные и
 * открытые задачи, люди не на месте по графику, посты без исполнителя.
 */
export function nowActions(tasks: readonly TaskEntry[], alerts: readonly AlertEntry[], posts: readonly PostRow[]): NowAction[] {
  const out: NowAction[] = []
  for (const a of alerts)
    if (a.kind === 'not_moved_to_isolator') out.push({ key: `a:${a.alert_id}`, kind: 'not_moved', tone: 'danger', item: a.item, since: a.at, ref: a.ref ?? null })
  for (const t of tasks.filter((x) => x.state === 'open').sort((a, b) => Number(b.overdue) - Number(a.overdue)))
    out.push({ key: `t:${t.task_id}`, kind: 'task', tone: t.overdue ? 'danger' : 'warn', title: t.title, since: t.due_at, ref: t.ref ?? null })
  for (const p of posts) {
    if (p.assigned && MISMATCH.has(p.presence))
      out.push({ key: `p:${p.workplace_id}`, kind: 'presence', tone: 'warn', post: p.station, person: p.assigned.display, presence: p.presence, workplaceId: p.workplace_id })
    if (!p.assigned || p.presence === 'not_assigned') out.push({ key: `u:${p.workplace_id}`, kind: 'unassigned', tone: 'warn', post: p.station, workplaceId: p.workplace_id })
  }
  return out
}

/** Сводка смены: требуют действия, постов работает, без исполнителя, изделий в работе. */
export function shiftSummary(actions: readonly NowAction[], posts: readonly PostRow[], steps: readonly StationStep[] | null) {
  const inWork = (steps ?? []).reduce((n, s) => n + (s.counters?.in_progress ?? 0), 0)
  return {
    actions: actions.length,
    working: posts.filter((p) => p.presence === 'present').length,
    unassigned: posts.filter((p) => !p.assigned || p.presence === 'not_assigned').length,
    inWork,
  }
}

/** Тон карточки поста: на месте — норма; не назначен — пусто; расхождение присутствия — внимание. */
export const postTone = (p: PostRow): 'ok' | 'warn' | 'none' => (p.presence === 'present' ? 'ok' : !p.assigned || p.presence === 'not_assigned' ? 'none' : 'warn')
