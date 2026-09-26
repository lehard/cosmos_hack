<script setup lang="ts">
/**
 * Доступ — представление (FR-78, FR-128, FR-145; AD-11, AD-15, AD-28): сотрудники
 * с ролями в области и сроками, роли и полномочия политики, реестр цифровых
 * клейм; выдача роли, полномочия или клейма и отзыв с основанием.
 * Привилегированная выдача исполняется только после второй подписи
 * независимой стороны — это решает сервер, здесь только сказано.
 */
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NButton, NInput, NRadioButton, NRadioGroup, NSelect } from 'naive-ui'
import type { AccessPerson, AccessRoleList, AccessStamp, GrantPolicyKind } from '@/entities/policy'
import { ACCESS_SECTIONS, grantReady as isReady, type AccessSection, type GrantDraft } from '../model/draft'
import type { Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'

const props = withDefaults(
  defineProps<{
    section: AccessSection
    persons?: AccessPerson[] | null
    personsError?: unknown
    roles?: AccessRoleList | null
    rolesError?: unknown
    stamps?: AccessStamp[] | null
    stampsError?: unknown
    canAct?: boolean
    busy?: boolean
    error?: unknown
    /** Итог последней команды: номер критического действия или «ждёт второй подписи». */
    result?: { kind: 'grant' | 'revoke'; ca: string | null } | null
    /** «Сейчас» для даты «действует с» по умолчанию (RFC 3339). */
    today: string
    density?: Density
  }>(),
  {
    persons: null,
    personsError: undefined,
    roles: null,
    rolesError: undefined,
    stamps: null,
    stampsError: undefined,
    canAct: true,
    busy: false,
    error: undefined,
    result: null,
    density: 'comfortable',
  },
)
const emit = defineEmits<{
  'update:section': [section: AccessSection]
  grant: [draft: GrantDraft]
  revoke: [person: AccessPerson, roleId: string, scope: string, reason: string]
}>()
const { t, d } = useI18n()
const problemText = useProblemText()
const date = (iso: string | undefined) => (iso ? d(new Date(iso), 'date') : '—')
const roleTitle = (id: string) => props.roles?.items.find((r) => r.id === id)?.title ?? id

// ── выдача ──
const draft = reactive<GrantDraft>({ kind: 'role', person_id: '', subject_id: '', scope: 'ent01', valid_from: props.today.slice(0, 10), valid_until: '', order_ref: '', inspection_kind: '', document_id: '' })
const personOptions = computed(() => (props.persons ?? []).map((p) => ({ label: `${p.display_name} (${p.person_id})`, value: p.person_id })))
const subjectOptions = computed(() => {
  if (!props.roles) return []
  if (draft.kind === 'role') return props.roles.items.map((r) => ({ label: r.title, value: r.id }))
  if (draft.kind === 'authority') return props.roles.authorities.map((a) => ({ label: a.title, value: a.id }))
  return []
})
const kindOptions = computed(() => (props.roles?.stamp_kinds ?? []).map((k) => ({ label: k, value: k })))
const grantReady = computed(() => isReady(draft))
function setKind(k: GrantPolicyKind): void {
  draft.kind = k
  draft.subject_id = ''
}

// ── отзыв ──
const revoking = ref<{ person: string; role: string; scope: string } | null>(null)
const revokeReason = ref('')
function confirmRevoke(p: AccessPerson): void {
  if (!revoking.value || !revokeReason.value.trim()) return
  emit('revoke', p, revoking.value.role, revoking.value.scope, revokeReason.value.trim())
  revoking.value = null
  revokeReason.value = ''
}
</script>

<template>
  <div class="access" :class="`density-${density}`" data-testid="access-admin">
    <NRadioGroup :value="section" size="small" @update:value="(v: AccessSection) => emit('update:section', v)">
      <NRadioButton v-for="s in ACCESS_SECTIONS" :key="s" :value="s" :data-testid="`section-${s}`">
        {{ t(`widgets.admin.access.sections.${s}`) }}
      </NRadioButton>
    </NRadioGroup>
    <p class="muted">{{ t('access.policyChangesLogged') }}</p>

    <section v-if="section === 'persons'" data-testid="persons">
      <p v-if="personsError" class="error">{{ problemText(personsError) }}</p>
      <p v-else-if="persons && !persons.length" class="muted">{{ t('empty.noRecords') }}</p>
      <table v-else-if="persons">
        <thead>
          <tr>
            <th>{{ t('access.conditionalId') }}</th>
            <th>{{ t('access.role') }} · {{ t('access.scope') }}</th>
            <th>{{ t('common.words.status') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in persons" :key="p.person_id" :data-person="p.person_id">
            <td>
              <strong>{{ p.display_name }}</strong>
              <div class="muted">{{ p.person_id }}<template v-if="p.org_unit"> · {{ p.org_unit }}</template></div>
            </td>
            <td>
              <p v-if="!p.roles.length" class="muted">{{ t('widgets.admin.access.noRoles') }}</p>
              <div v-for="r in p.roles" :key="`${r.role_id}@${r.scope}`" class="grant" :data-role="r.role_id">
                <span>{{ roleTitle(r.role_id) }}</span>
                <span class="muted">{{ r.scope }} · {{ date(r.valid_from) }} — {{ date(r.valid_until) }}</span>
                <form
                  v-if="revoking?.person === p.person_id && revoking.role === r.role_id && revoking.scope === r.scope"
                  class="inline"
                  @submit.prevent="confirmRevoke(p)"
                >
                  <NInput v-model:value="revokeReason" size="tiny" :placeholder="t('widgets.admin.access.revokeReason')" :aria-label="t('widgets.admin.access.revokeReason')" />
                  <NButton size="tiny" type="error" attr-type="submit" :disabled="!canAct || busy || !revokeReason.trim()" data-testid="confirm-revoke">
                    {{ t('widgets.admin.access.revoke') }}
                  </NButton>
                  <NButton size="tiny" quaternary @click="revoking = null">{{ t('common.actions.cancel') }}</NButton>
                </form>
                <NButton
                  v-else
                  size="tiny"
                  quaternary
                  :disabled="!canAct || busy"
                  data-testid="revoke"
                  @click="revoking = { person: p.person_id, role: r.role_id, scope: r.scope }"
                  >{{ t('widgets.admin.access.revoke') }}</NButton
                >
              </div>
            </td>
            <td>{{ t(`widgets.admin.access.accountStatus.${p.account_status}`) }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <section v-else-if="section === 'roles'" data-testid="roles">
      <p v-if="rolesError" class="error">{{ problemText(rolesError) }}</p>
      <template v-else-if="roles">
        <ul class="plain">
          <li v-for="r in roles.items" :key="r.id" :data-role="r.id">
            <strong>{{ r.title }}</strong>
            <span class="muted">{{ r.id }}<template v-if="r.case_role"> · {{ t('widgets.admin.access.caseRole') }}</template></span>
            <span v-if="r.inherits.length" class="muted">{{ t('widgets.admin.access.inherits', { roles: r.inherits.map(roleTitle).join(', ') }) }}</span>
            <span class="muted">{{ t('widgets.admin.access.actions', { n: r.actions.length }) }}</span>
          </li>
        </ul>
        <h4>{{ t('access.authorities') }}</h4>
        <ul class="plain">
          <li v-for="a in roles.authorities" :key="a.id" :data-authority="a.id">{{ a.title }} <span class="muted">{{ a.id }}</span></li>
        </ul>
      </template>
    </section>

    <section v-else-if="section === 'stamps'" data-testid="stamps">
      <h4>{{ t('access.stampRegistry') }}</h4>
      <p v-if="stampsError" class="error">{{ problemText(stampsError) }}</p>
      <p v-else-if="stamps && !stamps.length" class="muted">{{ t('empty.noRecords') }}</p>
      <ul v-else-if="stamps" class="plain">
        <li v-for="s in stamps" :key="s.stamp_id" :data-stamp="s.stamp_id" :data-status="s.status">
          <strong>{{ s.stamp_id }}</strong> · {{ s.person_id }} ·
          {{ t(`widgets.admin.access.stampStatus.${s.status}`) }}
          <div class="muted">{{ t('audit.stampIssued', { order: s.order_ref, kind: s.inspection_kind, validUntil: date(s.valid_until) }) }} · {{ s.scope }}</div>
        </li>
      </ul>
    </section>

    <form v-else class="grant-form" data-testid="grant" @submit.prevent="grantReady && emit('grant', { ...draft })">
      <NAlert type="warning" :bordered="false" :show-icon="false">{{ t('access.privilegedGrant') }}</NAlert>
      <NRadioGroup :value="draft.kind" size="small" @update:value="setKind">
        <NRadioButton v-for="k in ['role', 'authority', 'stamp'] as const" :key="k" :value="k" :data-testid="`kind-${k}`">{{ t(`widgets.admin.access.kind.${k}`) }}</NRadioButton>
      </NRadioGroup>
      <label>
        <span>{{ t('widgets.admin.access.person') }}</span>
        <NSelect v-if="personOptions.length" v-model:value="draft.person_id" size="small" filterable :options="personOptions" data-testid="person" />
        <NInput v-else v-model:value="draft.person_id" size="small" data-testid="person" />
      </label>
      <label>
        <span>{{ t('widgets.admin.access.subject') }}</span>
        <NSelect v-if="subjectOptions.length" v-model:value="draft.subject_id" size="small" filterable :options="subjectOptions" data-testid="subject" />
        <NInput v-else v-model:value="draft.subject_id" size="small" data-testid="subject" />
      </label>
      <template v-if="draft.kind === 'stamp'">
        <label>
          <span>{{ t('widgets.admin.access.inspectionKind') }}</span>
          <NSelect v-if="kindOptions.length" v-model:value="draft.inspection_kind" size="small" :options="kindOptions" data-testid="inspection-kind" />
          <NInput v-else v-model:value="draft.inspection_kind" size="small" data-testid="inspection-kind" />
        </label>
        <label>
          <span>{{ t('widgets.admin.access.order') }}</span>
          <NInput v-model:value="draft.order_ref" size="small" data-testid="order" />
        </label>
      </template>
      <label>
        <span>{{ t('access.scope') }} <span class="muted">({{ t('access.scopeLevels') }})</span></span>
        <NInput v-model:value="draft.scope" size="small" data-testid="scope" />
      </label>
      <div class="dates">
        <label>
          <span>{{ t('widgets.admin.access.validFrom') }}</span>
          <input v-model="draft.valid_from" type="date" data-testid="valid-from" />
        </label>
        <label>
          <span>{{ t('access.validUntil') }}</span>
          <input v-model="draft.valid_until" type="date" data-testid="valid-until" />
        </label>
      </div>
      <label>
        <span>{{ t('widgets.admin.access.document') }}</span>
        <NInput v-model:value="draft.document_id" size="small" data-testid="document" />
      </label>
      <NButton type="primary" attr-type="submit" :disabled="!canAct || busy || !grantReady" data-testid="submit-grant">{{ t('common.actions.confirm') }}</NButton>
    </form>

    <p v-if="result" class="ok" data-testid="result">
      <template v-if="result.kind === 'grant'">{{ result.ca ? t('widgets.admin.access.granted', { ca: result.ca }) : t('widgets.admin.access.grantedPending') }}</template>
      <template v-else>{{ t('widgets.admin.access.revoked', { ca: result.ca ?? '—' }) }}</template>
    </p>
    <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
  </div>
</template>

<style scoped>
.access {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}

h4 {
  margin: 8px 0 4px;
  font-size: 1em;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  padding: 3px 6px;
  border-bottom: 1px solid #e5e7eb;
  text-align: left;
  vertical-align: top;
}

th {
  color: #6b7280;
  font-weight: 400;
}

.grant,
.inline {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 8px;
  align-items: center;
}

.plain {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.plain li {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 8px;
  align-items: baseline;
}

.grant-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 520px;
}

.grant-form label {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.dates {
  display: flex;
  gap: 12px;
}

.muted {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}

.error {
  color: #d64545;
}

.ok {
  margin: 0;
  color: #2e9e5b;
}
</style>
