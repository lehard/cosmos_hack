/**
 * Права окна процесса (AD-15, FR-85): разрешено ли действие x-ant-action над
 * версией процесса по списку прав с сервера (@casl/vue). Без подключённых
 * прав (тест стола, экран вне оболочки) — ничего нельзя, но и не падаем.
 */
import { inject } from 'vue'
import { ABILITY_TOKEN } from '@casl/vue'

/** Функция «разрешено ли действие над версией процесса». */
export function useCanOnVersion(): (actionId: string) => boolean {
  const ability = inject(ABILITY_TOKEN, null)
  return (actionId) => ability?.can(actionId, 'process_version') ?? false
}
