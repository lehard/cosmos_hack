/**
 * Источник данных виджета «process-versions» (FR-24): список версий, выбранная
 * версия целиком, действующая целиком и отличия выбранной от действующей
 * (операции `process.version.list`, `process.version.read`, `process.version.diff`).
 * Отличия считает сервер; если его перечня нет, экран сравнивает две
 * прочитанные версии сам (entities/process-version, diffVersions).
 */
import { computed, ref } from 'vue'
import { useProcessVersion, useProcessVersionDiff, useProcessVersionList, type ProcessVersion } from '@/entities/process-version'

/** Данные виджета «process-versions». */
export function useProcessVersionsSource() {
  const list = useProcessVersionList()
  const selected = ref<string | null>(null)
  const activeId = computed(() => list.versions.value?.find((v) => v.status === 'active')?.version_id ?? null)
  const current = useProcessVersion(selected)
  const active = useProcessVersion(activeId)
  const diff = useProcessVersionDiff(selected, activeId)

  /** Сводки списка, у выбранной и действующей — с элементами. */
  const versions = computed<ProcessVersion[] | null>(() => {
    const vs = list.versions.value
    if (!vs) return null
    const full = [current.version.value, active.version.value].filter((v): v is ProcessVersion => v !== null)
    return vs.map((v) => full.find((f) => f.version_id === v.version_id) ?? v)
  })

  return {
    versions,
    selected,
    diff: diff.entries,
    mode: list.mode,
    isPending: computed(() => list.query.isLoading.value),
    error: computed(() => list.query.error.value ?? current.query.error.value ?? undefined),
  }
}
