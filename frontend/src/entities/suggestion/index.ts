/**
 * Сущность «предложения и меры» (эпик 42; FR-63, FR-64, FR-138, FR-143):
 * предложения генераторов, корректирующие меры с планом эффективности и
 * взглядом руководителя по качеству, карта дефицита данных. Операции —
 * `analysis.suggestion.list|generate|forward|resolve`, `analysis.action.list|
 * implement|evaluate`, `analysis.data_deficit.read` (contracts/openapi.yaml) —
 * сгенерированным клиентом. Ключи кэша — сущность `incident` (shared/api/keys):
 * предложения и меры относятся к инцидентам; после команды перечитываются все три.
 */
import { computed } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import {
  analysisActionEvaluate,
  analysisActionImplement,
  analysisActionList,
  analysisDataDeficitRead,
  analysisSuggestionForward,
  analysisSuggestionGenerate,
  analysisSuggestionList,
  analysisSuggestionResolve,
} from '@/shared/api/generated/client'
import type {
  CorrectiveActionList,
  CorrectiveActionView,
  DataDeficitMap,
  DeficitRow,
  GeneratorInfo,
  MemoryEntry,
  RecurringProblem,
  Suggestion,
  SuggestionList,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { ApiError } from '@/shared/api/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'

export type { CorrectiveActionList, CorrectiveActionView, DataDeficitMap, DeficitRow, GeneratorInfo, MemoryEntry, RecurringProblem, Suggestion, SuggestionList }

export const suggestionKeys = entityKeys('incident')

/** Предложения и генераторы (`analysis.suggestion.list`). */
export function useSuggestions() {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => suggestionKeys.list('suggestions', moment.params)),
    queryFn: ({ signal }) => analysisSuggestionList(moment.params, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** Меры, взгляд руководителя по качеству, организационная память (`analysis.action.list`). */
export function useCorrectiveActions() {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => suggestionKeys.list('actions', moment.params)),
    queryFn: ({ signal }) => analysisActionList(moment.params, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** Карта дефицита данных (`analysis.data_deficit.read`). */
export function useDataDeficit() {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => suggestionKeys.list('data-deficit', moment.params)),
    queryFn: ({ signal }) => analysisDataDeficitRead(moment.params, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
  })
}

/** Метаданные команды: сеанс даёт policy_seq и рабочее место. */
export interface CommandContext {
  policySeq: number
  workplaceId?: string
}

function meta(c: CommandContext, basisSeq = 0) {
  return { command_id: newCommandId(), basis_seq: basisSeq, policy_seq: c.policySeq, ...(c.workplaceId ? { workplace_id: c.workplaceId } : {}) }
}

/** Команда по предложениям и мерам. */
export type SuggestionCommand =
  | { kind: 'generate' }
  | { kind: 'forward'; suggestion: Suggestion; responsibleId: string; responsibleRole?: string; note?: string }
  | { kind: 'resolve'; suggestion: Suggestion; resolution: 'accepted' | 'rejected'; reason: string }
  | { kind: 'implement'; action: CorrectiveActionView; note?: string }
  | { kind: 'evaluate'; action: CorrectiveActionView; result: 'effective' | 'failed'; evidence?: string }

function send(c: CommandContext, cmd: SuggestionCommand) {
  switch (cmd.kind) {
    case 'generate':
      return analysisSuggestionGenerate(meta(c))
    case 'forward':
      return analysisSuggestionForward(cmd.suggestion.suggestion_id, {
        ...meta(c, cmd.suggestion.basis_seq),
        responsible_id: cmd.responsibleId,
        ...(cmd.responsibleRole ? { responsible_role: cmd.responsibleRole } : {}),
        ...(cmd.note ? { note: cmd.note } : {}),
      })
    case 'resolve':
      return analysisSuggestionResolve(cmd.suggestion.suggestion_id, { ...meta(c, cmd.suggestion.basis_seq), resolution: cmd.resolution, reason: { text: cmd.reason } })
    // Меры не несут guard_relevant записей в потоке инцидента: гард — над проекцией меры, basis_seq не нужен.
    case 'implement':
      return analysisActionImplement(cmd.action.incident_id, cmd.action.action_id, { ...meta(c), ...(cmd.note ? { note: cmd.note } : {}) })
    case 'evaluate':
      return analysisActionEvaluate(cmd.action.incident_id, cmd.action.action_id, {
        ...meta(c),
        result: cmd.result,
        ...(cmd.evidence ? { evidence: cmd.evidence } : {}),
      })
  }
}

/**
 * Команда по предложению или мере (FR-63, FR-64): после успеха перечитываются
 * предложения, меры и карта дефицита. Отказ гарда (код и текст) — в `error`.
 */
export function useSuggestionCommand(context: () => CommandContext) {
  const queryClient = useQueryClient()
  return useMutation<Awaited<ReturnType<typeof send>>, ApiError, SuggestionCommand>({
    mutationFn: (cmd) => send(context(), cmd),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: suggestionKeys.all }),
  })
}
export { useOpenRecord, type OpenRecord } from './model/open'
