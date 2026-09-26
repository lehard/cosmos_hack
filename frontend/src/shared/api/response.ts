/**
 * Помощники над ответами сгенерированного клиента (он отдаёт `{ data, status, headers }`).
 */
import type { BackendMode } from './generated/model'

/** Заголовок режима ведущих портов (AD-21, AD-36). */
export const BACKEND_HEADER = 'Ant-Backend'

/**
 * Режим `fixtures | live` из ответа — для метки на виджете (FR-150).
 * Нет заголовка — null: метку не показываем, а не угадываем.
 */
export function backendModeOf(response: { headers?: Headers } | undefined | null): BackendMode | null {
  const value = response?.headers?.get(BACKEND_HEADER)
  return value === 'fixtures' || value === 'live' ? value : null
}

/**
 * Ответ в форме сгенерированного клиента: тело и заголовки (из заголовка
 * `Ant-Backend` виджет берёт метку режима fixtures | live — backendModeOf).
 * Обёртки entities/* отдают виджетам тело уже распакованным (без `items`).
 */
export interface Envelope<T> {
  data: T
  headers?: Headers
}
