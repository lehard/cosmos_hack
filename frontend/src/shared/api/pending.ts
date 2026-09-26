/**
 * Заглушка «операция ещё не в контракте» для обёрток entities/* (FR-150, AD-36).
 *
 * Интерфейс ходит только через API: пока операции нет в contracts/openapi.yaml,
 * обёртка сущности ставит настоящий запрос Vue Query с настоящим ключом кэша, а
 * функция запроса отвечает так же, как ответит сервер на объявленную, но не
 * реализованную операцию, — `501 api.not_implemented` (contracts/errors.yaml).
 * Виджет показывает «ошибку входа», а не выдуманные данные. Когда операция
 * появится, функция запроса меняется на вызов сгенерированного клиента — ключи,
 * инвалидация SSE и виджеты остаются прежними.
 */
import type { ApiError } from './problem'

/**
 * Ошибка в форме сгенерированного клиента: `status` и тело problem+json в `info`.
 * @param operationId — id операции по соглашению `‹модуль›.‹объект›.‹действие›` (AD-40)
 */
export function notImplementedError(operationId: string): ApiError {
  return Object.assign(new Error(`${operationId}: операция ещё не реализована`), {
    status: 501,
    info: {
      type: 'urn:ant:problem:api.not_implemented',
      title: 'Операция ещё не реализована',
      status: 501,
      code: 'api.not_implemented',
      params: { operation_id: operationId },
    },
  })
}

/**
 * Ответ в форме сгенерированного клиента: тело и заголовки (из заголовка
 * `Ant-Backend` виджет берёт метку режима fixtures | live — backendModeOf).
 */
export interface Envelope<T> {
  data: T
  headers?: Headers
}

/**
 * Функция запроса для операции, которой ещё нет в контракте.
 * @param operationId — ожидаемый id операции
 */
export const pendingOperation =
  <T>(operationId: string) =>
  async (): Promise<T> => {
    throw notImplementedError(operationId)
  }
