/**
 * Открытая запись — правое окно записи (Д-70, UI-7). Состояние живёт в адресе:
 * `?open=‹тип›:‹id›`, поэтому ссылка из уведомления, поиска в шапке или
 * «требует внимания» открывает то же окно, а «назад» в браузере его закрывает.
 *
 * Тип — вид сущности контракта SSE (`item`, `nonconformity`, …); id может
 * содержать двоеточие (`item:ENT01:F-017`), тип отделяется по первому.
 * Какие типы окно умеет показывать, решает реестр оболочки (app/record) и
 * сообщает виджетам через provide — см. RECORD_DRAWER.
 */
import { computed, inject, type ComputedRef, type InjectionKey } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import type { DrillRef } from './drill'

/** Параметр адреса открытой записи. */
export const RECORD_QUERY = 'open'

/** Разобрать `‹тип›:‹id›`; не то — null. */
export function parseRecord(value: unknown): DrillRef | null {
  const v = Array.isArray(value) ? value[0] : value
  if (typeof v !== 'string') return null
  const i = v.indexOf(':')
  if (i <= 0 || i === v.length - 1) return null
  return { entity: v.slice(0, i) as DrillRef['entity'], id: v.slice(i + 1) }
}

/** Записать ссылку в значение параметра. */
export const formatRecord = (ref: DrillRef): string => `${ref.entity}:${ref.id}`

/** Одна и та же запись. */
export const sameRecord = (a: DrillRef | null, b: DrillRef | null): boolean => !!a && !!b && a.entity === b.entity && a.id === b.id

/** Запрос адреса без открытой записи. */
export function withoutRecord(query: LocationQuery): LocationQuery {
  const rest = { ...query }
  delete rest[RECORD_QUERY]
  return rest
}

/** Что оболочка сообщает виджетам об окне записи. */
export interface RecordDrawerApi {
  /** Типы записей, для которых есть содержимое окна. */
  kinds: ReadonlySet<string>
}

/**
 * Ключ provide: оболочка (ShellLayout) ставит окно записи. Нет окна (тест
 * виджета, экран входа) — ссылки ведут на страницы, как раньше.
 */
export const RECORD_DRAWER: InjectionKey<RecordDrawerApi> = Symbol('record-drawer')

/** Открытая запись и переходы к ней. */
export function useRecordLink(): {
  current: ComputedRef<DrillRef | null>
  canOpen: (ref: DrillRef) => boolean
  open: (ref: DrillRef) => boolean
  close: () => void
} {
  const router = useRouter()
  const route = useRoute()
  const drawer = inject(RECORD_DRAWER, null)
  const current = computed(() => parseRecord(route?.query[RECORD_QUERY]))

  const canOpen = (ref: DrillRef): boolean => !!drawer && drawer.kinds.has(ref.entity)

  /** Открыть окно записи; false — окна для такого типа нет. */
  function open(ref: DrillRef): boolean {
    if (!canOpen(ref)) return false
    if (sameRecord(current.value, ref)) return true
    // push, а не replace: «назад» в браузере закрывает окно.
    void router.push({ query: { ...route.query, [RECORD_QUERY]: formatRecord(ref) } })
    return true
  }

  /** Закрыть: если окно открыли с этой же страницы — шаг назад, иначе убрать параметр. */
  function close(): void {
    if (!current.value) return
    const target = router.resolve({ query: withoutRecord(route.query) }).fullPath
    const back = typeof window !== 'undefined' ? (window.history.state as { back?: unknown } | null)?.back : undefined
    if (back === target) router.back()
    else void router.replace({ query: withoutRecord(route.query) })
  }

  return { current, canOpen, open, close }
}
