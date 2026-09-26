/**
 * Рабочее место и пост (FR-6, FR-81): ключи кэша и панель «Посты» под живой
 * картой — участок, назначенный сотрудник, присутствие (СКУД, ключ вставлен),
 * текущее изделие. Люди на схеме не показываются — только здесь (PRD §4.1).
 *
 * Операции ещё нет в contracts/openapi.yaml (эпик 02): запрос стоит на заглушке
 * `api.not_implemented`, форма данных — предложение для контракта.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { entityKeys } from '@/shared/api/keys'
import { pendingOperation, type Envelope } from '@/shared/api/pending'
import { useMomentStore } from '@/shared/model/moment'

export const workplaceKeys = entityKeys('workplace')

/**
 * Присутствие на посту (FR-6): на месте; по графику (СКУД) на месте, но ключ не
 * вставлен; ключ вставлен, а владельца нет в зоне; нет ни в зоне, ни ключа;
 * никто не назначен; данных нет — «неизвестно», а не «на месте».
 */
export type PostPresence = 'present' | 'key_missing' | 'owner_absent' | 'absent' | 'not_assigned' | 'unknown'

/** Строка панели «Посты». */
export interface PostRow {
  workplace_id: string
  /** Участок (пост) — подпись. */
  station: string
  /** Цех (FR-130). */
  workshop?: string
  /** Назначенный сотрудник: псевдоним исполнителя (соглашение «Идентификаторы»). */
  assigned: { person_id: string; display: string } | null
  presence: PostPresence
  current_item: { item_id: string; label: string } | null
}

/**
 * Посты под картой на момент из useMomentStore. Ожидаемая операция —
 * `access.workplace.list` (GET /api/v1/workplaces).
 */
export function usePosts(params: MaybeRefOrGetter<{ workshop?: string; run_id?: string }>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => workplaceKeys.list('posts', toValue(params), moment.params)),
    queryFn: () => pendingOperation<Envelope<PostRow[]>>('access.workplace.list')(),
    placeholderData: (prev: Envelope<PostRow[]> | undefined) => prev,
  })
}
