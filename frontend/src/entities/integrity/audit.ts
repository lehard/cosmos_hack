/**
 * Стол Аудитора ИБ — только чтение (эпик 14; FR-79, FR-106, FR-108; AD-28,
 * AD-44, AD-46; кейс Т3): отчёты верификатора, журнал критических действий,
 * шина событий безопасности.
 *
 * Операции — `security.verifier_report.list|read`, `security.critical_action.list|read`,
 * `security.event.list` (contracts/openapi.yaml), сгенерированный клиент. Ключи —
 * `[integrity, …]`: SSE `integrity` (новый отчёт, нарушение) перечитывает весь стол.
 * Журналы читаются на момент (AD-37): в воспроизведении — «что было на момент».
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import {
  securityCriticalActionList,
  securityCriticalActionRead,
  securityEventList,
  securityVerifierReportList,
  securityVerifierReportRead,
} from '@/shared/api/generated/client'
import type {
  CriticalAction,
  CriticalActionCaGroup,
  SecurityCriticalActionListParams,
  SecurityEvent,
  SecurityEventSeverity,
  VerifierCheckRow,
  VerifierReport,
  VerifierReportSummary,
} from '@/shared/api/generated/model'
import { entityKeys } from '@/shared/api/keys'
import { useMomentStore } from '@/shared/model/moment'

export type { CriticalAction, CriticalActionCaGroup, SecurityEvent, SecurityEventSeverity, VerifierCheckRow, VerifierReport, VerifierReportSummary }

const keys = entityKeys('integrity')

/**
 * Период перечитывания журналов «сейчас», мс: критические действия и события
 * безопасности рождаются на любых объектах, отдельного сообщения SSE у них нет.
 */
export const AUDIT_POLL_MS = 15_000

/** Отчёты верификатора, новые сверху (`security.verifier_report.list`, AD-46). */
export function useVerifierReports() {
  return useQuery({
    queryKey: keys.list('verifier-reports'),
    queryFn: ({ signal }) => securityVerifierReportList({ limit: 50 }, { signal }),
    retry: false,
    refetchInterval: 60_000,
  })
}

/** Отчёт верификатора целиком: проверки, классы подписей, бумажные решения. */
export function useVerifierReport(digest: MaybeRefOrGetter<string | null>) {
  return useQuery({
    queryKey: computed(() => keys.list('verifier-report', toValue(digest) ?? '')),
    queryFn: ({ signal }) => securityVerifierReportRead(toValue(digest) ?? '', { signal }),
    enabled: computed(() => !!toValue(digest)),
    retry: false,
  })
}

/** Фильтр журнала критических действий. */
export type CriticalActionFilter = Pick<SecurityCriticalActionListParams, 'group' | 'actor_id' | 'subject' | 'id'>

/** Журнал критических действий на момент (`security.critical_action.list`, AD-28). */
export function useCriticalActions(filter: MaybeRefOrGetter<CriticalActionFilter> = {}) {
  const moment = useMomentStore()
  const params = computed(() => ({ limit: 200, ...toValue(filter), ...moment.params }))
  return useQuery({
    queryKey: computed(() => keys.list('critical-actions', params.value)),
    queryFn: ({ signal }) => securityCriticalActionList(params.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
    refetchInterval: () => (moment.isReplay ? false : AUDIT_POLL_MS),
  })
}

/** Критическое действие CA-‹n› целиком (`security.critical_action.read`). */
export function useCriticalAction(caRef: MaybeRefOrGetter<string | null>) {
  return useQuery({
    queryKey: computed(() => keys.list('critical-action', toValue(caRef) ?? '')),
    queryFn: ({ signal }) => securityCriticalActionRead(toValue(caRef) ?? '', { signal }),
    enabled: computed(() => !!toValue(caRef)),
    retry: false,
  })
}

/** События безопасности на момент (`security.event.list`); тип — фильтр семейства security. */
export function useSecurityEvents(eventType: MaybeRefOrGetter<string | null> = null) {
  const moment = useMomentStore()
  const params = computed(() => {
    const type = toValue(eventType)
    return { limit: 200, ...(type ? { event_type: type } : {}), ...moment.params }
  })
  return useQuery({
    queryKey: computed(() => keys.list('security-events', params.value)),
    queryFn: ({ signal }) => securityEventList(params.value, { signal }),
    retry: false,
    placeholderData: keepPreviousData,
    refetchInterval: () => (moment.isReplay ? false : AUDIT_POLL_MS),
  })
}

/** Типы записей семейства security (contracts/events/security) — фильтр шины событий. */
export const SECURITY_EVENT_TYPES: readonly string[] = [
  'security.auth.failed',
  'security.signature.invalid',
  'security.integrity.violated',
  'security.integrity.checked',
  'security.access.denied',
  'security.admission.denied',
  'security.idempotency.conflict',
  'security.critical_action.recorded',
  'security.keeper.alert',
  'security.key.alert',
  'security.presence.deviation',
]

/** Группы критических действий (AD-28) — фильтр журнала, в порядке контракта. */
export const CA_GROUPS: readonly CriticalActionCaGroup[] = [
  'product_decision',
  'nc_decision',
  'cause',
  'risk_scope',
  'control_change',
  'authority',
  'protected_data',
  'admin_security',
]
