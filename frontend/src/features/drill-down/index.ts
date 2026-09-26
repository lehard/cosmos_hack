/**
 * Проваливание в детали отовсюду (FR-7): изделие → паспорт; несоответствие →
 * карточка; тревога → объект тревоги; показатель → исходные записи.
 *
 * Ссылка — вид сущности контракта SSE и id (`DrillRef`). Куда вести, решает
 * таблица маршрутов: изделие — страница паспорта `item`; для остальных видов —
 * маршрут с именем вида сущности, если его уже зарегистрировал эпик-владелец
 * экрана (например, `nonconformity` — эпик 11). Нет маршрута — ссылка неактивна,
 * а не ведёт в пустоту.
 */
import { useRouter, type RouteLocationRaw } from 'vue-router'
import type { DrillRef } from '@/entities/live-map'

export type { DrillRef }

/** Куда ведёт ссылка, или null — экрана для этого вида ещё нет. */
export function drillTarget(ref: DrillRef, hasRoute: (name: string) => boolean): RouteLocationRaw | null {
  if (ref.entity === 'item') return { name: 'item', params: { id: ref.id } }
  if (hasRoute(ref.entity)) return { name: ref.entity, params: { id: ref.id } }
  return null
}

/** Переходы к деталям для виджетов. */
export function useDrillDown() {
  const router = useRouter()
  const target = (ref: DrillRef) => drillTarget(ref, (name) => router.hasRoute(name))
  return {
    /** Есть ли экран для ссылки. */
    canOpen: (ref: DrillRef): boolean => target(ref) !== null,
    /** Перейти; false — экрана ещё нет. */
    open(ref: DrillRef): boolean {
      const to = target(ref)
      if (!to) return false
      void router.push(to)
      return true
    },
  }
}
