/**
 * Данные дорожки решения: карта реакций (`quality.reaction_map.read`) —
 * название, условие и порог сработавшего правила. Наблюдение и паспорт
 * анализатора — хуки сущности analyzer-passport.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { qualityReactionMapRead } from '@/shared/api/generated/client'
import type { ReactionRule } from '@/shared/api/generated/model'
import { useMomentStore } from '@/shared/model/moment'

/** Правило действующей карты реакций по номеру; нет номера — не читаем. */
export function useReactionRule(ruleId: MaybeRefOrGetter<string | undefined>) {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => ['reaction_map', moment.params] as const),
    queryFn: ({ signal }) => qualityReactionMapRead({ ...moment.params }, { signal }),
    enabled: computed(() => !!toValue(ruleId)),
    retry: false,
    staleTime: 300_000,
  })
  const rule = computed<ReactionRule | null>(() => q.data.value?.data.rules.find((r) => r.rule_id === toValue(ruleId)) ?? null)
  return { rule, query: q }
}
