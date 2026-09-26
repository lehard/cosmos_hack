/**
 * Версии процесса через сгенерированный клиент (AD-20, AD-21): список
 * (`process.version.list`), читаемое представление (`process.version.read`) и
 * отличия от действующей (`process.version.diff`). Ключи —
 * `[process_version, id, …]` / `[process_version, @list, …]` с моментом последним.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { processVersionDiff, processVersionList, processVersionRead } from '@/shared/api/generated/client'
import type {
  ProcessDiffEntry as ApiDiffEntry,
  ProcessVersion as ApiVersion,
  ProcessVersionSummary,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import { backendModeOf } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'
import type { ProcessDiffEntry, ProcessPropertyKey, ProcessPropertyValue, ProcessVersion } from './model/types'

export const processVersionKeys = entityKeys('process_version')

const value = (v: unknown): ProcessPropertyValue =>
  v === null || v === undefined ? null : typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean' ? v : JSON.stringify(v)

/** Сводка из списка → версия без элементов (элементы — из чтения версии). */
export const fromSummary = (s: ProcessVersionSummary): ProcessVersion => ({
  version_id: s.version_id,
  label: s.label,
  status: s.status,
  created_at: s.created_at,
  effective_from: s.effective_from ?? null,
  quorum: s.quorum ?? null,
  author: null,
  elements: [],
})

/** Версия процесса в читаемом виде. */
export const toProcessVersion = (a: ApiVersion): ProcessVersion => ({
  version_id: a.version_id,
  label: a.label,
  status: a.status,
  author: a.author ?? null,
  created_at: a.created_at,
  effective_from: a.effective_from ?? null,
  quorum: a.quorum ?? null,
  elements: a.elements.map((e) => ({
    id: e.id,
    step_key: e.step_key ?? null,
    kind: e.kind,
    name: e.name,
    lane: e.lane ?? null,
    properties: Object.fromEntries(Object.entries(e.properties).map(([k, v]) => [k, value(v)])) as Partial<Record<ProcessPropertyKey, ProcessPropertyValue>>,
    thresholds: e.thresholds ?? undefined,
    next: e.next ?? [],
  })),
})

const num = (v: unknown): number | null => (typeof v === 'number' ? v : null)

/** Отличие от сервера → строка отличия экрана; неполная запись пропускается. */
export function toDiffEntry(a: ApiDiffEntry): ProcessDiffEntry | null {
  switch (a.kind) {
    case 'elementAdded':
    case 'elementRemoved':
      return a.element ? { kind: a.kind, element: a.element } : null
    case 'presentationPointAdded':
      return a.step ? { kind: a.kind, step: a.step } : null
    case 'thresholdChanged':
      return a.defectType ? { kind: a.kind, element: a.element ?? '', defectType: a.defectType, from: num(a.from), to: num(a.to) } : null
    case 'propertyChanged':
      return a.property
        ? { kind: a.kind, element: a.element ?? '', property: a.property as ProcessPropertyKey, from: value(a.from), to: value(a.to) }
        : null
  }
  return null
}

/** Список версий процесса. */
export function useProcessVersionList() {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => processVersionKeys.list(moment.params)),
    queryFn: ({ signal }) => processVersionList(moment.params, { signal }),
  })
  return {
    query: q,
    versions: computed(() => q.data.value?.data.items.map(fromSummary) ?? null),
    mode: computed(() => backendModeOf(q.data.value)),
  }
}

/** Версия процесса целиком; null — не выбрана. */
export function useProcessVersion(versionId: MaybeRefOrGetter<string | null>) {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => processVersionKeys.one(toValue(versionId) ?? '', moment.params)),
    queryFn: ({ signal }) => processVersionRead(toValue(versionId)!, moment.params, { signal }),
    enabled: computed(() => Boolean(toValue(versionId))),
  })
  return { query: q, version: computed(() => (q.data.value ? toProcessVersion(q.data.value.data) : null)) }
}

/** Отличия версии от другой (по умолчанию сервер берёт действующую). */
export function useProcessVersionDiff(versionId: MaybeRefOrGetter<string | null>, against: MaybeRefOrGetter<string | null>) {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => processVersionKeys.one(toValue(versionId) ?? '', 'diff', toValue(against), moment.params)),
    queryFn: ({ signal }) =>
      processVersionDiff(toValue(versionId)!, { ...moment.params, ...(toValue(against) ? { against: toValue(against)! } : {}) }, { signal }),
    enabled: computed(() => Boolean(toValue(versionId) && toValue(against) && toValue(versionId) !== toValue(against))),
  })
  return {
    query: q,
    entries: computed(() => (q.data.value ? q.data.value.data.entries.map(toDiffEntry).filter((e): e is ProcessDiffEntry => e !== null) : null)),
  }
}
