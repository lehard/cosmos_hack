/**
 * Данные рабочего стола технолога: инциденты, группы разбора (названия вида
 * дефекта и шага), меры (сводка и повторяющиеся проблемы), процессы и их версии.
 */
import { computed } from 'vue'
import { useQueries, useQuery } from '@tanstack/vue-query'
import { useIncidents, useNcGroups } from '@/entities/incident'
import { useCorrectiveActions } from '@/entities/suggestion'
import { processProcessList, processVersionList } from '@/shared/api/generated/client'
import { entityKeys } from '@/shared/api/keys'
import { useMomentStore } from '@/shared/model/moment'
import type { InboxInput } from './tasks'

const keys = entityKeys('process_version')

export function useInboxSource() {
  const moment = useMomentStore()
  const incidents = useIncidents()
  const groups = useNcGroups()
  const actions = useCorrectiveActions()
  const processes = useQuery({
    queryKey: computed(() => keys.list('processes', moment.params)),
    queryFn: ({ signal }) => processProcessList(moment.params, { signal }),
  })
  const procList = computed(() => processes.data.value?.data.items ?? [])
  const versionQs = useQueries({
    queries: computed(() =>
      procList.value.map((p) => ({
        queryKey: keys.list('editor', { ...moment.params, process_id: p.process_id }),
        queryFn: ({ signal }: { signal: AbortSignal }) => processVersionList({ ...moment.params, process_id: p.process_id }, { signal }),
      })),
    ),
  })

  const input = computed<InboxInput>(() => ({
    incidents: incidents.data.value?.data.items ?? [],
    groups: groups.data.value ?? [],
    actions: actions.data.value?.data ?? null,
    versions: procList.value.flatMap((p, i) => (versionQs.value[i]?.data?.data.items ?? []).map((v) => ({ process: p, version: v }))),
  }))
  const loading = computed(() => incidents.isPending.value || actions.isPending.value)
  const error = computed(() => incidents.error.value ?? null)
  return { input, loading, error }
}
