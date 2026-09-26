/**
 * Открытая запись правого окна страниц эпика 42 (Д-70): `?open=‹вид›:‹id›` в
 * адресе, поэтому ссылка из задачи или уведомления открывает то же окно.
 * Без маршрутизатора (тест виджета) состояние держится локально.
 */
import { computed, inject, ref } from 'vue'
import { routeLocationKey, routerKey } from 'vue-router'

export interface OpenRecord<K extends string> {
  kind: K
  id: string
}

/** Открытая запись заданных видов и переход к ней. */
export function useOpenRecord<K extends string>(kinds: readonly K[]) {
  // inject с умолчанием: без маршрутизатора (тест виджета) — без предупреждений.
  const route = inject(routeLocationKey, null)
  const router = inject(routerKey, null)
  const local = ref<string | null>(null)

  const raw = computed<string | null>(() => {
    if (!route) return local.value
    const v = route.query.open
    return typeof v === 'string' && v ? v : null
  })

  const open = computed<OpenRecord<K> | null>(() => {
    const v = raw.value
    if (!v) return null
    const i = v.indexOf(':')
    if (i <= 0) return null
    const kind = v.slice(0, i) as K
    const id = v.slice(i + 1)
    return kinds.includes(kind) && id ? { kind, id } : null
  })

  function setOpen(ref: OpenRecord<K> | null): void {
    const value = ref ? `${ref.kind}:${ref.id}` : null
    if (!route || !router) {
      local.value = value
      return
    }
    const query = { ...route.query }
    if (value) query.open = value
    else delete query.open
    void router.replace({ query })
  }

  return { open, setOpen }
}
