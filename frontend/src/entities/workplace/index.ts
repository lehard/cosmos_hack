/**
 * Рабочее место и пост (FR-6, FR-81): ключи кэша и панель «Посты» под живой
 * картой — участок, назначенный сотрудник, присутствие (СКУД, ключ вставлен),
 * текущее изделие. Люди на схеме не показываются — только здесь (PRD §4.1).
 *
 * Операция — `access.workplace.list` (contracts/openapi.yaml), сгенерированный клиент.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { entityKeys } from '@/shared/api/keys'
import { accessWorkplaceList } from '@/shared/api/generated/client'
import type { PostRow, PostRowPresence } from '@/shared/api/generated/model'
import type { Envelope } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'

export const workplaceKeys = entityKeys('workplace')

/** Присутствие на посту (FR-6): «неизвестно» — не «на месте». */
export type PostPresence = PostRowPresence
export type { PostRow }

/** Посты под картой на момент из useMomentStore — `access.workplace.list`. */
export function usePosts(params: MaybeRefOrGetter<{ workshop?: string; run_id?: string }>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => workplaceKeys.list('posts', toValue(params), moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<PostRow[]>> => {
      const res = await accessWorkplaceList({ ...toValue(params), ...moment.params }, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
  })
}
