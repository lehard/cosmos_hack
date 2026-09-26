/**
 * «Сейчас» по часам сервера. Сервер отдаёт в каждом ответе заголовок `Ant-Now`
 * (RFC 3339): в заготовках — часы шага мира, в демо — часы сценария, в работе —
 * реальное время. Свежесть и «сколько прошло» (целостность журнала, сроки,
 * «осталось / просрочено») считаются от него, а не от часов браузера — иначе
 * мир заготовок 23.09 выглядит «давно устаревшим».
 *
 * Храним смещение серверных часов относительно браузера: между ответами время
 * идёт своим ходом. Заголовка нет — смещение 0, то есть часы браузера.
 * Время для идентификаторов команд (UUIDv7) сюда не относится — там реальное.
 */
import { computed, onScopeDispose, ref, type ComputedRef } from 'vue'

/** Заголовок ответа с «сейчас» сервера. */
export const SERVER_NOW_HEADER = 'Ant-Now'

const offset = ref(0)

/**
 * Запомнить «сейчас» сервера из заголовков ответа.
 * @param headers — заголовки ответа API (нет или без `Ant-Now` — ничего не меняет)
 */
export function noteServerNow(headers: Headers | undefined | null): void {
  const v = headers?.get(SERVER_NOW_HEADER)
  if (!v) return
  const at = Date.parse(v)
  if (!Number.isNaN(at)) offset.value = at - Date.now()
}

/** «Сейчас» сервера, мс UTC. */
export const serverNow = (): number => Date.now() + offset.value

/**
 * Реактивное «сейчас» сервера: идёт раз в `everyMs` и сразу меняется при новом `Ant-Now`.
 * @param everyMs — шаг, мс (по умолчанию 30 с)
 */
export function useServerNow(everyMs = 30_000): ComputedRef<number> {
  const tick = ref(Date.now())
  const timer = setInterval(() => (tick.value = Date.now()), everyMs)
  onScopeDispose(() => clearInterval(timer))
  return computed(() => tick.value + offset.value)
}
