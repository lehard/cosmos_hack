/**
 * Сеанс пользователя (FR-128, AD-15): кто вошёл, в какой роли, области и смене.
 * Серверное состояние — только в кэше Vue Query (AD-21), ключ ['session', 'me'].
 * Вход демо-персоной без пароля — демо-трек (заметка эпика 08); пароль заложен,
 * пока не обязателен.
 */
import { useQueryClient } from '@tanstack/vue-query'
import {
  accessSessionRead,
  useAccessPersonaList,
  useAccessSessionCreate,
  useAccessSessionDelete,
  useAccessSessionRead,
} from '@/shared/api/generated/client'

export type { DemoPersona, RoleRef, Session, SessionCreate } from '@/shared/api/generated/model'

/** Ключ кэша текущего сеанса. */
export const sessionKey = ['session', 'me'] as const

/** Ключ кэша демо-персон. */
export const personasKey = ['session', 'personas'] as const

/** Опции запроса сеанса — для охранника маршрутов (queryClient.fetchQuery). */
export const sessionQueryOptions = () => ({
  queryKey: sessionKey,
  queryFn: ({ signal }: { signal: AbortSignal }) => accessSessionRead({ signal }),
  retry: false,
  staleTime: 60_000,
})

/** Текущий сеанс. */
export const useSession = () => useAccessSessionRead({ query: { queryKey: sessionKey, retry: false, staleTime: 60_000 } })

/** Демо-персоны для входа без пароля (404 вне демо-профиля). */
export const usePersonas = () => useAccessPersonaList({ query: { queryKey: personasKey, retry: false, staleTime: Infinity } })

/** Войти: после входа весь кэш прежнего пользователя сбрасывается. */
export function useLogin() {
  const queryClient = useQueryClient()
  return useAccessSessionCreate({
    mutation: {
      onSuccess: (response) => {
        queryClient.removeQueries({ predicate: (q) => q.queryKey[0] !== personasKey[0] })
        queryClient.setQueryData(sessionKey, { ...response, status: 200 })
      },
    },
  })
}

/** Выйти и забыть все серверные данные. */
export function useLogout() {
  const queryClient = useQueryClient()
  return useAccessSessionDelete({ mutation: { onSettled: () => queryClient.clear() } })
}
