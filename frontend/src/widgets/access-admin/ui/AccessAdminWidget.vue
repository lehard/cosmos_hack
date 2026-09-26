<script setup lang="ts">
/**
 * Виджет «Доступ» (FR-78, FR-128, FR-145; AD-11, AD-15) — контейнер:
 * сотрудники (`access.person.list`), роли и полномочия (`access.role.list`),
 * клейма (`access.stamp.list`); выдача и отзыв (`access.policy.grant|revoke`,
 * command_id — UUIDv7, AD-7; policy_seq сеанса, AD-39). Разделы читаются
 * независимо: отказ одной операции не прячет остальные.
 * Срез: `section: persons | roles | stamps | grant` — раздел по умолчанию.
 */
import { computed, ref } from 'vue'
import { useActivateAccount, usePersons, usePolicyCommand, useRoles, useStamps, type AccessPerson } from '@/entities/policy'
import { useSession } from '@/entities/session'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetProps } from '@/shared/config/widget'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import { ACCESS_SECTIONS, type AccessSection, type ActivationDraft, type GrantDraft } from '../model/draft'
import AccessAdminView from './AccessAdminView.vue'

const props = defineProps<WidgetProps>()
const moment = useMomentStore()
const session = useSession()

const section = ref<AccessSection>(ACCESS_SECTIONS.includes(props.slice.section as AccessSection) ? (props.slice.section as AccessSection) : 'persons')

const personsQ = usePersons()
const rolesQ = useRoles()
const stampsQ = useStamps()
const persons = computed(() => personsQ.data.value?.data?.items ?? null)
const roles = computed(() => rolesQ.data.value?.data ?? null)
const command = usePolicyCommand()
const activate = useActivateAccount()
const result = ref<{ kind: 'grant' | 'revoke' | 'activate'; ca: string | null; seq?: number } | null>(null)
const today = new Date().toISOString()

/** Общие поля команды: id, версия политики из ответа чтения, рабочее место. */
function meta() {
  const s = session.data.value?.data
  const policySeq = roles.value?.policy_seq ?? s?.policy_seq ?? 0
  return { command_id: newCommandId(), basis_seq: policySeq, policy_seq: policySeq, ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}) }
}

/** Дата формы (ГГГГ-ММ-ДД) → начало суток UTC (RFC 3339). */
const dayStart = (v: string) => `${v}T00:00:00Z`

async function onGrant(dr: GrantDraft): Promise<void> {
  result.value = null
  activate.reset()
  const subject = dr.kind === 'role' ? { role_id: dr.subject_id } : dr.kind === 'authority' ? { authority_id: dr.subject_id } : { stamp_id: dr.subject_id }
  try {
    const res = await command.mutateAsync({
      kind: 'grant',
      body: {
        ...meta(),
        kind: dr.kind,
        person_id: dr.person_id.trim(),
        scope: dr.scope.trim(),
        valid_from: dayStart(dr.valid_from),
        ...subject,
        ...(dr.valid_until ? { valid_until: dayStart(dr.valid_until) } : {}),
        ...(dr.kind === 'stamp' ? { order_ref: dr.order_ref.trim(), inspection_kind: dr.inspection_kind.trim() } : {}),
        ...(dr.document_id.trim() ? { document_id: dr.document_id.trim() } : {}),
      },
    })
    result.value = { kind: 'grant', ca: res.data.ca_ref ?? null }
  } catch {
    // Текст ошибки — из command.error.
  }
}

async function onRevoke(p: AccessPerson, roleId: string, scope: string, reason: string): Promise<void> {
  result.value = null
  activate.reset()
  try {
    const res = await command.mutateAsync({
      kind: 'revoke',
      body: { ...meta(), basis_seq: p.policy_seq, kind: 'role', person_id: p.person_id, subject_id: roleId, scope, effective_from: new Date().toISOString(), reason: { text: reason } },
    })
    result.value = { kind: 'revoke', ca: res.data.ca_ref ?? null }
  } catch {
    // Текст ошибки — из command.error.
  }
}

/**
 * Активировать учётную запись (FR-128, эпик 08): basis_seq — версия политики,
 * на которой построена строка сотрудника; пароль — только если введён.
 */
async function onActivate(p: AccessPerson, dr: ActivationDraft): Promise<void> {
  result.value = null
  command.reset()
  try {
    const res = await activate.mutateAsync({
      person_id: p.person_id,
      body: {
        ...meta(),
        basis_seq: p.policy_seq,
        login: dr.login.trim(),
        ...(dr.password ? { password: dr.password } : {}),
        ...(dr.initial_role_id ? { initial_role_id: dr.initial_role_id } : {}),
        ...(dr.scope.trim() ? { scope: dr.scope.trim() } : {}),
      },
    })
    result.value = { kind: 'activate', ca: res.data.ca_ref ?? null, seq: res.data.seq }
  } catch {
    // Текст ошибки — из activate.error.
  }
}

const allFailed = computed(() => !!personsQ.error.value && !!rolesQ.error.value && !!stampsQ.error.value)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(personsQ.data.value ?? rolesQ.data.value)"
    :loading="personsQ.isPending.value && rolesQ.isPending.value && stampsQ.isPending.value"
    :state="allFailed ? 'input_error' : 'normal'"
    :data-widget="widgetId"
  >
    <AccessAdminView
      v-model:section="section"
      :persons="persons"
      :persons-error="persons ? undefined : (personsQ.error.value ?? undefined)"
      :roles="roles"
      :roles-error="roles ? undefined : (rolesQ.error.value ?? undefined)"
      :stamps="stampsQ.data.value?.data?.items ?? null"
      :stamps-error="stampsQ.data.value ? undefined : (stampsQ.error.value ?? undefined)"
      :can-act="!moment.isReplay"
      :busy="command.isPending.value || activate.isPending.value"
      :error="command.error.value ?? activate.error.value ?? undefined"
      :result="result"
      :today="today"
      :density="density"
      @grant="onGrant"
      @revoke="onRevoke"
      @activate="onActivate"
    />
  </WidgetFrame>
</template>
