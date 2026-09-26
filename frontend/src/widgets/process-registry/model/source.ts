/**
 * Данные и команды виджета «Редактор процесса» (эпик 39; FR-22…FR-25, UJ-4):
 * версии процесса (`process.version.list` c `process_id`, UI-11), BPMN версии
 * (`process.version.bpmn`), читаемая разница (`process.version.diff`), лист
 * утверждения (`documents.document.read`) и путь версии —
 * `process.version.draft | submit | activate | retire`.
 * После команды перечитываются версии, живая карта и документы.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import {
  documentsDocumentRead,
  processVersionActivate,
  processVersionBpmn,
  processProcessList,
  processVersionDiff,
  processVersionDraft,
  processVersionList,
  processVersionRetire,
  processVersionSubmit,
} from '@/shared/api/generated/client'
import type { ProcessSummary, ProcessVersionSummary } from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import { backendModeOf } from '@/shared/api/response'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { useSession } from '@/entities/session'

export type { ProcessSummary, ProcessVersionSummary }

const keys = entityKeys('process_version')
const docKeys = entityKeys('document')

/** Процессы для выбора (process.process.list, UI-11). */
export function useProcesses() {
  const moment = useMomentStore()
  const q = useQuery({
    queryKey: computed(() => keys.list('processes', moment.params)),
    queryFn: ({ signal }) => processProcessList(moment.params, { signal }),
  })
  return {
    query: q,
    processes: computed<ProcessSummary[] | null>(() => q.data.value?.data.items ?? null),
    mode: computed(() => backendModeOf(q.data.value)),
  }
}

/** BPMN XML версии для скачивания (process.version.bpmn). */
export async function fetchBpmn(versionId: string): Promise<string> {
  return (await processVersionBpmn(versionId)).data.bpmn_xml
}

/** Версии процесса processId (пусто — основной процесс). */
export function useVersions(processId: MaybeRefOrGetter<string | null>, enabled: MaybeRefOrGetter<boolean> = true) {
  const moment = useMomentStore()
  const params = computed(() => ({ ...moment.params, ...(toValue(processId) ? { process_id: toValue(processId)! } : {}) }))
  const q = useQuery({
    queryKey: computed(() => keys.list('editor', params.value)),
    queryFn: ({ signal }) => processVersionList(params.value, { signal }),
    enabled: computed(() => toValue(enabled)),
  })
  return {
    query: q,
    versions: computed<ProcessVersionSummary[] | null>(() => q.data.value?.data.items ?? null),
    mode: computed(() => backendModeOf(q.data.value)),
  }
}

/** BPMN XML версии как загружен (AD-17). */
export function useBpmn(versionId: MaybeRefOrGetter<string | null>) {
  return useQuery({
    queryKey: computed(() => keys.one(toValue(versionId) ?? '', 'bpmn')),
    queryFn: ({ signal }) => processVersionBpmn(toValue(versionId)!, { signal }),
    enabled: computed(() => Boolean(toValue(versionId))),
    staleTime: Infinity,
  })
}

/** Разница версии с действующей версией её процесса (FR-24). */
export function useDiff(versionId: MaybeRefOrGetter<string | null>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => keys.one(toValue(versionId) ?? '', 'diff', moment.params)),
    queryFn: ({ signal }) => processVersionDiff(toValue(versionId)!, moment.params, { signal }),
    enabled: computed(() => Boolean(toValue(versionId))),
  })
}

/** Лист утверждения версии (документ с маршрутом кворума, FR-23). */
export function useApprovalDocument(documentId: MaybeRefOrGetter<string | null | undefined>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => docKeys.one(toValue(documentId) ?? '', moment.params)),
    queryFn: ({ signal }) => documentsDocumentRead(toValue(documentId)!, moment.params, { signal }),
    enabled: computed(() => Boolean(toValue(documentId))),
  })
}

/** Команда пути версии. */
export type LifecycleCommand =
  | { kind: 'draft'; label: string; base_version_id?: string; bpmn_xml: string }
  | { kind: 'submit'; version_id: string; note: string }
  | { kind: 'activate'; version_id: string; route_closed_event_id: string }
  | { kind: 'retire'; version_id: string; reason: string }

/** Команды пути версии (один command_id на намерение, AD-7). */
export function useLifecycle() {
  const qc = useQueryClient()
  const session = useSession()
  return useMutation({
    mutationFn: async (c: LifecycleCommand) => {
      const header = { command_id: newCommandId(), basis_seq: 0, policy_seq: session.data.value?.data.policy_seq ?? 0 }
      switch (c.kind) {
        case 'draft':
          return processVersionDraft({ ...header, label: c.label, ...(c.base_version_id ? { base_version_id: c.base_version_id } : {}), bpmn_xml: c.bpmn_xml })
        case 'submit':
          return processVersionSubmit(c.version_id, { ...header, ...(c.note ? { note: c.note } : {}) })
        case 'activate':
          return processVersionActivate(c.version_id, { ...header, route_closed_event_id: c.route_closed_event_id })
        case 'retire':
          return processVersionRetire(c.version_id, { ...header, reason: c.reason })
      }
    },
    onSuccess: () => {
      for (const key of [['process_version'], ['live_map'], ['document']]) void qc.invalidateQueries({ queryKey: key })
    },
  })
}
