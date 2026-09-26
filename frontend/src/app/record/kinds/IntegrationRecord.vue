<script setup lang="ts">
/**
 * Окно интеграции (Д-70; FR-157, AD-47, эпик 48): состояние «включена /
 * выключена / стенд», адрес и живой канал, последний обмен, ошибки,
 * последняя проверка соединения, последнее решение администратора, очередь и
 * карантин исходящих с ручной переотправкой (FR-96). Внизу — «Проверить
 * соединение», «Включить» / «Выключить», «Стенд ↔ реальная система»:
 * администратор решает единолично (Д-71), основание попадает в журнал.
 * Неустановленная система — пояснение, как её установить, без кнопок.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput } from 'naive-ui'
import {
  LEDGER_SYSTEMS, integrationActions, isQuarantined, useCheckIntegration, useIntegrations, useLedgerMessages, useResendPosting, useSetIntegration,
  type ErpMessage, type IntegrationStateCode,
} from '@/entities/integration'
import { useSession } from '@/entities/session'
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, DataTable, EmptyState, KeyValue, KeyValueList, RecordDrawer, SectionPanel } from '@/shared/ui'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t, te, d } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()

const listQ = useIntegrations()
const entry = computed(() => listQ.data.value?.data.items.find((e) => e.system === props.id) ?? null)
const profile = computed(() => listQ.data.value?.data.profile ?? '')
const actions = computed(() => (entry.value ? integrationActions(entry.value) : { enable: null, disable: false, toggle: null }))

const ledger = computed(() => LEDGER_SYSTEMS.has(props.id) && !!entry.value?.installed)
const messagesQ = useLedgerMessages(
  computed(() => props.id),
  computed(() => ledger.value && props.show),
)
const quarantined = computed<ErpMessage[]>(() => (messagesQ.data.value?.data.items ?? []).filter(isQuarantined))

const setM = useSetIntegration()
const checkM = useCheckIntegration()
const resendM = useResendPosting()
const reason = ref('')
const done = ref<'set' | 'check' | null>(null)
const resent = ref<string | null>(null)
watch(
  () => props.id,
  () => {
    reason.value = ''
    done.value = null
    resent.value = null
  },
)

const STATE_TONE: Record<string, StatusTone> = { enabled: 'success', stand: 'info', disabled: 'neutral' }
const systemName = computed(() => (te(`widgets.integrations.systems.${props.id}`) ? t(`widgets.integrations.systems.${props.id}`) : props.id))
const stateText = (s: string) => t(`widgets.integrations.states.${s}`)
const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : '—')
const canAct = computed(() => !moment.isReplay && !!entry.value?.installed)
const busy = computed(() => setM.isPending.value || checkM.isPending.value)

function meta() {
  const s = session.data.value?.data
  return {
    command_id: newCommandId(),
    policy_seq: s?.policy_seq ?? 0,
    ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
  }
}

async function setState(state: IntegrationStateCode): Promise<void> {
  const e = entry.value
  if (!e || !reason.value.trim()) return
  done.value = null
  try {
    await setM.mutateAsync({ system: e.system, body: { ...meta(), basis_seq: e.basis_seq, state, reason: { text: reason.value.trim() } } })
    done.value = 'set'
    reason.value = ''
  } catch {
    // Текст отказа — из setM.error (код гарда: ops.stand_forbidden и др.).
  }
}

async function check(): Promise<void> {
  done.value = null
  try {
    const m = meta()
    await checkM.mutateAsync({ system: props.id, commandId: m.command_id, policySeq: m.policy_seq })
    done.value = 'check'
  } catch {
    // Текст — из checkM.error.
  }
}

async function resend(m: ErpMessage): Promise<void> {
  resent.value = null
  try {
    await resendM.mutateAsync({
      businessKey: m.business_key,
      body: { ...meta(), basis_seq: m.basis_seq, request_event_id: m.request_event_id, reason: { text: t('shell.record.integration.resend') } },
    })
    resent.value = m.business_key
  } catch {
    // Текст — из resendM.error.
  }
}

const error = computed(() => setM.error.value ?? checkM.error.value ?? null)
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('shell.record.integration.kind')"
    :number="systemName"
    :subtitle="entry?.endpoint ?? ''"
    :loading="listQ.isPending.value && !entry"
    data-record="integration"
    @close="emit('close')"
  >
    <template v-if="entry" #status>
      <span class="tag ant-box" :data-state="entry.installed ? entry.state : 'not_installed'" data-testid="integration-state">
        <span class="dot" :style="{ background: statusPalette[entry.installed ? (STATE_TONE[entry.state] ?? 'neutral') : 'neutral'] }" aria-hidden="true" />
        <span class="ant-ellipsis">{{ entry.installed ? stateText(entry.state) : t('widgets.integrations.states.notInstalled') }}</span>
      </span>
    </template>

    <EmptyState v-if="!entry && listQ.error.value" compact :title="t('errors.loadFailed')" :description="problemText(listQ.error.value)" />
    <EmptyState v-else-if="!entry" compact :title="t('shell.record.integration.notFound')" />
    <div v-else class="sections ant-box">
      <p v-if="!entry.installed" class="note ant-wrap" data-testid="integration-not-installed">{{ t('shell.record.integration.notInstalledHint') }}</p>

      <SectionPanel v-if="entry.installed" :title="t('shell.record.integration.state')" variant="plain" :padded="false" data-testid="integration-now">
        <KeyValueList>
          <KeyValue :label="t('shell.record.integration.mode')">
            <span class="ant-wrap">{{ stateText(entry.state) }}<template v-if="entry.default"> · {{ t('widgets.integrations.byDefault') }}</template></span>
          </KeyValue>
          <KeyValue :label="t('shell.record.integration.endpoint')" :value="entry.endpoint ?? null" mono />
          <KeyValue :label="t('shell.record.integration.channel')">
            <span class="ant-wrap">{{ entry.channel ? t(`widgets.integrations.channels.${entry.channel}`) : '—' }}<template v-if="entry.detail"> — {{ entry.detail }}</template></span>
          </KeyValue>
          <KeyValue :label="t('shell.record.integration.checkedAt')" :value="time(entry.checked_at)" />
          <KeyValue :label="t('shell.record.integration.lastExchange')" :value="time(entry.last_exchange_at)" />
          <KeyValue v-if="entry.queued != null" :label="t('shell.record.integration.queued')">
            <span class="ant-wrap" data-testid="integration-queued">{{ entry.queued }}</span>
            <span v-if="entry.state === 'disabled'" class="muted ant-wrap"> · {{ t('shell.record.integration.queuedDisabled') }}</span>
          </KeyValue>
          <KeyValue v-if="entry.quarantined != null" :label="t('shell.record.integration.quarantined')" :value="String(entry.quarantined)" />
        </KeyValueList>
      </SectionPanel>

      <SectionPanel v-if="entry.installed" :title="t('shell.record.integration.errors')" variant="plain" :padded="false" data-testid="integration-errors">
        <p v-if="entry.last_error" class="ant-wrap err">{{ time(entry.last_error.at) }} — {{ entry.last_error.detail }}</p>
        <p v-else class="muted ant-wrap">{{ t('shell.record.integration.noErrors') }}</p>
        <p class="ant-wrap" data-testid="integration-last-check">
          <strong>{{ t('shell.record.integration.lastCheck') }}:</strong>
          <template v-if="entry.last_check">
            {{ t(`shell.record.integration.checkResults.${entry.last_check.result}`) }}, {{ time(entry.last_check.at) }}<template v-if="entry.last_check.detail"> — {{ entry.last_check.detail }}</template>
          </template>
          <span v-else class="muted">{{ t('shell.record.integration.noCheck') }}</span>
        </p>
      </SectionPanel>

      <SectionPanel v-if="ledger" :title="t('shell.record.integration.quarantine')" variant="plain" :padded="false" data-testid="integration-quarantine">
        <EmptyState v-if="messagesQ.error.value" compact :title="t('errors.loadFailed')" :description="problemText(messagesQ.error.value)" />
        <EmptyState v-else-if="messagesQ.data.value && !quarantined.length" compact :title="t('shell.record.integration.noQuarantine')" />
        <DataTable v-else-if="quarantined.length">
          <thead>
            <tr>
              <th>{{ t('shell.record.integration.message') }}</th>
              <th>{{ t('shell.record.integration.error') }}</th>
              <th />
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in quarantined" :key="m.business_key" data-testid="quarantined-message">
              <td><span class="mono ant-wrap">{{ m.business_key }}</span></td>
              <td><span class="ant-wrap">{{ m.attempts[m.attempts.length - 1]?.error_message ?? m.status }}</span></td>
              <td>
                <span v-if="resent === m.business_key" class="ok ant-wrap">{{ t('shell.record.integration.resent') }}</span>
                <ActionButton v-else size="tiny" :disabled="moment.isReplay || resendM.isPending.value" data-testid="resend" :label="t('shell.record.integration.resend')" @click="resend(m)" />
              </td>
            </tr>
          </tbody>
        </DataTable>
        <NAlert v-if="resendM.error.value" type="error" :bordered="false" :show-icon="false">{{ problemText(resendM.error.value) }}</NAlert>
      </SectionPanel>

      <SectionPanel v-if="entry.installed" :title="t('shell.record.integration.decision')" variant="plain" :padded="false" data-testid="integration-decision">
        <p v-if="entry.last_decision" class="ant-wrap">
          {{ t('shell.record.integration.decisionText', { state: stateText(entry.last_decision.state), actor: entry.last_decision.actor ?? '—', at: time(entry.last_decision.at) }) }}
          <span class="muted"> · {{ entry.last_decision.reason }}</span>
        </p>
        <p v-else class="muted ant-wrap">{{ t('shell.record.integration.noDecision') }}</p>
        <p v-if="profile === 'prod'" class="muted ant-wrap">{{ t('widgets.integrations.prodNoStand') }}</p>
      </SectionPanel>

      <SectionPanel v-if="canAct" :title="t('shell.record.integration.reason')" :subtitle="t('shell.record.integration.soleNote')" variant="plain" :padded="false">
        <NInput v-model:value="reason" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" :maxlength="2000" :placeholder="t('shell.record.integration.reasonPlaceholder')" data-testid="integration-reason" />
      </SectionPanel>
      <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="integration-error">{{ problemText(error) }}</NAlert>
      <NAlert v-else-if="done" type="success" :bordered="false" :show-icon="false" data-testid="integration-done">
        {{ t(done === 'set' ? 'shell.record.integration.done' : 'shell.record.integration.checked') }}
      </NAlert>
    </div>

    <template v-if="entry && canAct" #actions>
      <ActionButton :disabled="busy" data-testid="integration-check" :label="t('shell.record.integration.check')" @click="check" />
      <ActionButton
        v-if="actions.toggle"
        :disabled="busy || !reason.trim()"
        :hint="reason.trim() ? undefined : t('shell.record.integration.reasonRequired')"
        data-testid="integration-toggle"
        :label="t(actions.toggle === 'stand' ? 'shell.record.integration.toStand' : 'shell.record.integration.toReal')"
        @click="setState(actions.toggle)"
      />
      <ActionButton
        v-if="actions.disable"
        :disabled="busy || !reason.trim()"
        :hint="reason.trim() ? undefined : t('shell.record.integration.reasonRequired')"
        data-testid="integration-disable"
        :label="t('shell.record.integration.disable')"
        @click="setState('disabled')"
      />
      <ActionButton
        v-if="actions.enable"
        type="primary"
        :disabled="busy || !reason.trim()"
        :hint="reason.trim() ? undefined : t('shell.record.integration.reasonRequired')"
        data-testid="integration-enable"
        :label="t('shell.record.integration.enable')"
        @click="setState(actions.enable)"
      />
    </template>
  </RecordDrawer>
</template>

<style scoped>
.sections {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-6);
}

.tag {
  display: inline-flex;
  gap: var(--ant-space-2);
  align-items: center;
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.note {
  margin: 0;
  color: var(--ant-text-2);
}

.muted {
  color: var(--ant-text-3);
}

.err {
  color: var(--ant-status-danger);
}

.ok {
  color: var(--ant-text-2);
}

.mono {
  font-family: var(--ant-font-mono);
}

p {
  margin: 0 0 var(--ant-space-2);
}
</style>
