/**
 * Источник данных виджета «common-factors»: общие факторы группы в разборе
 * (операция `analysis.common_factors.read`, FR-135).
 */
import { computed } from 'vue'
import { useCommonFactors, useFocusedGroup } from '@/entities/incident'

/** Данные виджета «common-factors». */
export function useCommonFactorsSource() {
  const { group } = useFocusedGroup()
  return useCommonFactors(computed(() => group.value?.group_key ?? null))
}
