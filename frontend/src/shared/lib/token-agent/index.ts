/**
 * Порт агента токена для шапки (AD-14, FR-69). Агент — отдельный процесс на
 * рабочем месте, браузер говорит с ним через расширение (Native Messaging,
 * contracts/internal/token-agent), а не через сервер ant.
 *
 * Пока расширения нет (эпик 38), статус — «агент токена не найден»: подпись на
 * бумаге с заверением остаётся доступной (AD-43). Эпик 38 заменяет реализацию
 * `useTokenStatus`, не трогая шапку.
 */
import { readonly, ref, type Ref } from 'vue'

/** Состояние токена на рабочем месте. */
export type TokenStatus = 'inserted' | 'missing' | 'agent_missing'

const status = ref<TokenStatus>('agent_missing')

/** Текущее состояние токена (только чтение). */
export function useTokenStatus(): Readonly<Ref<TokenStatus>> {
  return readonly(status)
}
