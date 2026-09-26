/**
 * Права в интерфейсе (AD-15): единая способность @casl/vue, правила которой
 * следуют за списком прав с сервера и за режимом воспроизведения (в нём остаётся
 * только чтение, AD-21).
 */
import { watchEffect, type App } from 'vue'
import { abilitiesPlugin } from '@casl/vue'
import { usePermissions } from '@/entities/permission'
import { createAppAbility, rulesFrom } from '@/shared/lib/access'
import { useMomentStore } from '@/shared/model/moment'

/** Способность пользователя — одна на приложение. */
export const ability = createAppAbility()

/** Подключить @casl/vue (`<Can>`, `useAbility`). */
export function installAccess(app: App): void {
  app.use(abilitiesPlugin, ability, { useGlobalProperties: true })
}

/** Держать правила в согласии с сервером; вызывать внутри оболочки после входа. */
export function useAbilitySync(): void {
  const permissions = usePermissions()
  const moment = useMomentStore()
  watchEffect(() => {
    ability.update(rulesFrom(permissions.data.value?.data.items ?? [], moment.isReplay))
  })
}
