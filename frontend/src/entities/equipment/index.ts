/**
 * Оборудование (эпики 19, 23; FR-17, FR-121, FR-149): состояние по журналам
 * оборудования — работает ли, предупреждения («ресурс инструмента 73/75»),
 * текущее выполнение операции — и справочник с поверкой. Слой entities (FSD):
 * ключи `[equipment, …]` — изменение оборудования в SSE перечитывает их само.
 *
 * Операции — `machinelogs.equipment.list`, `machinelogs.run_profile.read`,
 * `reference.equipment.list` (contracts/openapi.yaml), сгенерированный клиент.
 * Состояние и предупреждения вычисляет сервер — здесь только показ.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { machinelogsEquipmentList, machinelogsRunProfileRead, referenceEquipmentList } from '@/shared/api/generated/client'
import type {
  EquipmentState,
  EquipmentStateCondition,
  EquipmentStateExecution,
  EquipmentVerificationStatus,
  EquipmentWarning,
  RefEquipment,
  RunProfile,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import type { Envelope } from '@/shared/api/response'
import { useMomentStore } from '@/shared/model/moment'

export type { EquipmentState, EquipmentStateCondition, EquipmentStateExecution, EquipmentVerificationStatus, EquipmentWarning, RefEquipment, RunProfile }

export const equipmentKeys = entityKeys('equipment')

/** Состояние оборудования на момент — `machinelogs.equipment.list`; пост пуст — всё. */
export function useEquipmentStates(stationId: MaybeRefOrGetter<string | null | undefined> = null) {
  const moment = useMomentStore()
  const params = computed(() => {
    const id = toValue(stationId)
    return id ? { station_id: id, ...moment.params } : { ...moment.params }
  })
  return useQuery({
    queryKey: computed(() => equipmentKeys.list('states', params.value)),
    queryFn: async ({ signal }): Promise<Envelope<EquipmentState[]>> => {
      const res = await machinelogsEquipmentList(params.value, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
    retry: false,
  })
}

/** Справочник оборудования и поверка на момент — `reference.equipment.list`. */
export function useEquipmentRegistry() {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => equipmentKeys.list('registry', moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<RefEquipment[]>> => {
      const res = await referenceEquipmentList({ ...moment.params }, { signal })
      return { data: res.data.items, headers: res.headers }
    },
    placeholderData: keepPreviousData,
    retry: false,
  })
}

/**
 * Профиль выполнения операции (FR-121) — `machinelogs.run_profile.read`: изделие,
 * операция, начало. Ключ — под изделием неизвестно каким, поэтому под
 * оборудованием: SSE оборудования (новое выполнение, конец цикла) перечитывает его.
 */
export function useRunProfile(runId: MaybeRefOrGetter<string | null | undefined>) {
  const moment = useMomentStore()
  return useQuery({
    queryKey: computed(() => equipmentKeys.list('run-profile', toValue(runId) ?? '', moment.params)),
    queryFn: async ({ signal }): Promise<Envelope<RunProfile>> => {
      const res = await machinelogsRunProfileRead(toValue(runId) ?? '', { ...moment.params }, { signal })
      return { data: res.data, headers: res.headers }
    },
    enabled: computed(() => !!toValue(runId)),
    retry: false,
  })
}

/** Ресурс инструмента «использовано из допустимого», если источник его передал. */
export function toolLifeOf(e: Pick<EquipmentState, 'tool_life_used' | 'tool_life_limit'>): { used: number; limit: number } | null {
  return typeof e.tool_life_used === 'number' && typeof e.tool_life_limit === 'number' ? { used: e.tool_life_used, limit: e.tool_life_limit } : null
}

/**
 * Можно ли работать на оборудовании по поверке (FR-17): справочник — источник
 * правды о пригодности; нет записи в справочнике — неизвестно (null).
 */
export function usableOf(registry: readonly RefEquipment[] | null | undefined, equipmentId: string): RefEquipment | null {
  return registry?.find((r) => r.equipment_id === equipmentId) ?? null
}
