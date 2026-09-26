/**
 * Проваливание в детали отовсюду (FR-7): изделие → паспорт; несоответствие →
 * карточка; тревога → объект тревоги; показатель → исходные записи.
 *
 * Ссылка — вид сущности контракта SSE и id (`DrillRef`). Куда вести, решает
 * таблица маршрутов: изделие — страница паспорта `item`; для остальных видов —
 * маршрут с именем вида сущности, если его уже зарегистрировал эпик-владелец
 * экрана (например, `nonconformity` — эпик 11). Нет маршрута — ссылка неактивна,
 * а не ведёт в пустоту.
 *
 * Д-70 (UI-7): если оболочка поставила правое окно записи и у него есть
 * содержимое для этого вида, ссылка открывает окно (`?open=‹тип›:‹id›`) поверх
 * текущего стола — пользователь не теряет список. Страница целиком — `openPage`.
 */
import { useRouter, type RouteLocationRaw } from 'vue-router'
import type { DrillRef } from '@/shared/model/drill'
import { useRecordLink } from '@/shared/model/record'

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
  const record = useRecordLink()
  const target = (ref: DrillRef) => drillTarget(ref, (name) => router.hasRoute(name))
  /** Перейти на страницу; false — экрана ещё нет. */
  function openPage(ref: DrillRef): boolean {
    const to = target(ref)
    if (!to) return false
    void router.push(to)
    return true
  }
  return {
    /** Есть ли окно записи или экран для ссылки. */
    canOpen: (ref: DrillRef): boolean => record.canOpen(ref) || target(ref) !== null,
    /** Открыть: окно записи справа, иначе страницу; false — показать нечем. */
    open(ref: DrillRef): boolean {
      return record.open(ref) || openPage(ref)
    },
    openPage,
  }
}
