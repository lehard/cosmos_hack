<script setup lang="ts">
/**
 * Окно выписки паспорта (Д-70; эпик 41, FR-131, FR-132, AD-19): что в
 * выписке (предмет, плавка, материал, химсостав, механические свойства,
 * результаты контроля отправителя), статус происхождения с пояснением,
 * подписи с итогом проверки каждой (ключи людей — по актам, подписанным
 * корнем партнёра; корень или шлюз предприятия), контрольная точка
 * хранителя отправителя. Внизу — «Скачать пакет» (подписанный DSSE: его
 * проверяет получатель без доступа к нашему журналу) и у входящей —
 * «Проверить заново» (повторный приём того же пакета: подписи — сейчас).
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert } from 'naive-ui'
import { ORIGIN_TONE, downloadEnvelope, shortDigest, useExtract, useReceiveExtract } from '@/entities/federation'
import { useSession } from '@/entities/session'
import { useDrillDown, type DrillRef } from '@/features/drill-down'
import { statusPalette } from '@/shared/api/generated/statuses'
import { useProblemText } from '@/shared/i18n/problem'
import { newCommandId } from '@/shared/lib/command-id'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, DataTable, EmptyState, KeyValue, KeyValueList, RecordDrawer, SectionPanel } from '@/shared/ui'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t, d } = useI18n()
const problemText = useProblemText()
const moment = useMomentStore()
const session = useSession()
const drill = useDrillDown()

const q = useExtract(
  computed(() => props.id),
  computed(() => props.show && !!props.id),
)
const view = computed(() => q.data.value?.data ?? null)
const x = computed(() => view.value?.extract ?? null)

interface ControlRow {
  name: string
  result: string
  value?: string
  norm?: string
  by?: string
}
interface Content {
  extract_no?: string
  sender?: string
  recipient?: string
  issued_at?: string
  subject?: { kind?: string; global_id?: string; label?: string; quantity?: string; recipient_ref?: string }
  origin?: { heat_no?: string; material?: string; standard?: string; certificate?: string; chemistry?: Record<string, string>; mechanical?: Record<string, string> }
  control?: ControlRow[]
  decisions?: string[]
}
const c = computed<Content>(() => (view.value?.content ?? {}) as Content)
const pairs = (m: Record<string, string> | undefined) => Object.entries(m ?? {}).map(([k, v]) => `${k} ${v}`).join(' · ')
const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : '—')

const receiveM = useReceiveExtract()
const rechecked = ref(false)
watch(
  () => props.id,
  () => {
    rechecked.value = false
    tamperTried.value = false
    receiveM.reset()
    tamperM.reset()
  },
)

async function recheck(): Promise<void> {
  const v = view.value
  if (!v?.envelope) return
  rechecked.value = false
  const s = session.data.value?.data
  try {
    await receiveM.mutateAsync({ command_id: newCommandId(), basis_seq: 0, policy_seq: s?.policy_seq ?? 0, partner_code: v.extract.partner_code, envelope: v.envelope })
    rechecked.value = true
  } catch {
    // Текст — из receiveM.error.
  }
}

/**
 * Изменённая копия пакета для показа отказа (FR-132): в содержимом под
 * подписью меняется номер плавки, подписи остаются прежними.
 */
function tamperedCopy(envelope: string): string | null {
  try {
    const env = JSON.parse(envelope) as { payload: string }
    const bin = atob(env.payload)
    const changed = bin.replace(/"heat_no":"([^"]*)"/, (_m, h: string) => `"heat_no":"${h}1"`)
    if (changed === bin) return null
    return JSON.stringify({ ...env, payload: btoa(changed) })
  } catch {
    return null
  }
}
const tamperable = computed(() => !!view.value?.envelope && tamperedCopy(view.value.envelope) !== null)
const tamperM = useReceiveExtract()
const tamperTried = ref(false)

async function tryTampered(): Promise<void> {
  const v = view.value
  const copy = v?.envelope ? tamperedCopy(v.envelope) : null
  if (!v || !copy) return
  tamperTried.value = false
  const s = session.data.value?.data
  try {
    await tamperM.mutateAsync({ command_id: newCommandId(), basis_seq: 0, policy_seq: s?.policy_seq ?? 0, partner_code: v.extract.partner_code, envelope: copy })
  } catch {
    // Ожидаемый отказ — текст из tamperM.error.
  }
  tamperTried.value = true
}

const openSubject = () => {
  const s = x.value?.subject
  if (s) drill.open({ entity: s.entity, id: s.id } as DrillRef)
}
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('shell.record.extract.kind')"
    :number="c.extract_no || shortDigest(id)"
    :subtitle="x ? `${t(`widgets.federation.directions.${x.direction}`)} · ${x.partner_code}` : ''"
    :loading="q.isPending.value && !view"
    data-record="extract"
    @close="emit('close')"
  >
    <template v-if="x" #status>
      <span class="tag ant-box" :data-origin="x.origin_status" data-testid="extract-status">
        <span class="dot" :style="{ background: statusPalette[ORIGIN_TONE[x.origin_status] ?? 'neutral'] }" aria-hidden="true" />
        <span class="ant-ellipsis">{{ t(`widgets.federation.origin.${x.origin_status}`) }}</span>
      </span>
    </template>

    <EmptyState v-if="!view && q.error.value" compact :title="t('errors.loadFailed')" :description="problemText(q.error.value)" />
    <div v-else-if="view && x" class="sections ant-box">
      <p v-if="view.origin_reason" class="note ant-wrap" data-testid="extract-reason">{{ view.origin_reason }}</p>

      <SectionPanel :title="t('shell.record.extract.what')" variant="plain" :padded="false">
        <KeyValueList>
          <KeyValue :label="t('shell.record.extract.subject')">
            <button v-if="x.subject" type="button" class="link ant-wrap" data-testid="extract-subject" @click="openSubject">{{ c.subject?.label || x.subject.id }}</button>
            <span v-else class="ant-wrap">{{ c.subject?.label || '—' }}</span>
          </KeyValue>
          <KeyValue :label="t('shell.record.extract.globalId')" :value="x.global_id ?? c.subject?.global_id ?? null" mono />
          <KeyValue :label="t('shell.record.extract.heat')" :value="c.origin?.heat_no ?? null" />
          <KeyValue :label="t('shell.record.extract.material')" :value="[c.origin?.material, c.origin?.standard].filter(Boolean).join(', ') || null" />
          <KeyValue :label="t('shell.record.extract.certificate')" :value="c.origin?.certificate ?? null" />
          <KeyValue v-if="c.origin?.chemistry" :label="t('shell.record.extract.chemistry')" :value="pairs(c.origin?.chemistry)" />
          <KeyValue v-if="c.origin?.mechanical" :label="t('shell.record.extract.mechanical')" :value="pairs(c.origin?.mechanical)" />
          <KeyValue :label="t('shell.record.extract.issued')" :value="`${c.sender ?? '—'} → ${c.recipient ?? '—'} · ${time(c.issued_at)}`" />
        </KeyValueList>
      </SectionPanel>

      <SectionPanel v-if="c.control?.length" :title="t('shell.record.extract.control')" variant="plain" :padded="false">
        <DataTable>
          <tbody>
            <tr v-for="(r, i) in c.control" :key="i" data-testid="extract-control">
              <td><span class="ant-wrap">{{ r.name }}</span></td>
              <td><span class="ant-wrap">{{ r.result }}<template v-if="r.value"> · {{ r.value }}</template></span></td>
              <td><span class="muted ant-wrap">{{ r.norm ?? '' }}</span></td>
            </tr>
          </tbody>
        </DataTable>
        <p v-for="(s, i) in c.decisions ?? []" :key="i" class="ant-wrap">{{ s }}</p>
      </SectionPanel>

      <SectionPanel :title="t('shell.record.extract.signatures')" :subtitle="t('shell.record.extract.signaturesHint')" variant="plain" :padded="false">
        <EmptyState v-if="!view.signatures.length" compact :title="t('shell.record.extract.noSignatures')" />
        <DataTable v-else>
          <tbody>
            <tr v-for="s in view.signatures" :key="s.key_ref" :data-verification="s.verification" data-testid="extract-signature">
              <td><span class="ant-wrap">{{ s.name || s.key_ref }}</span><div class="muted mono ant-ellipsis">{{ s.key_ref }}</div></td>
              <td><span class="ant-wrap">{{ s.signer_role ?? '' }}</span><div class="muted ant-wrap">{{ t(s.human ? 'shell.record.extract.byAct' : 'shell.record.extract.byRoot') }}</div></td>
              <td><span class="ant-wrap" :class="{ err: s.verification === 'invalid' }">{{ t(`shell.record.extract.verification.${s.verification}`) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
        <p class="ant-wrap"><strong>{{ t('shell.record.extract.checkpoint') }}:</strong> <span class="mono">{{ view.checkpoint || t('shell.record.extract.noCheckpoint') }}</span></p>
      </SectionPanel>

      <SectionPanel :title="t('shell.record.extract.exchange')" variant="plain" :padded="false">
        <KeyValueList>
          <KeyValue :label="t('shell.record.extract.digest')" :value="x.extract_digest" mono />
          <KeyValue v-if="x.document_id" :label="t('shell.record.extract.document')" :value="x.document_id" mono />
          <KeyValue v-if="x.direction === 'outgoing'" :label="t('shell.record.extract.ack')">
            <span class="ant-wrap">{{ x.acknowledged == null ? t('widgets.federation.ackPending') : x.acknowledged ? t('widgets.federation.ackReceived') : t('widgets.federation.ackRejected') }}</span>
          </KeyValue>
          <KeyValue :label="t('widgets.federation.at')" :value="time(x.at)" />
        </KeyValueList>
      </SectionPanel>

      <NAlert v-if="tamperTried && tamperM.error.value" type="info" :bordered="false" :show-icon="false" data-testid="extract-tamper-rejected">
        {{ t('shell.record.extract.tamperRejected') }} {{ problemText(tamperM.error.value) }}
      </NAlert>
      <NAlert v-else-if="tamperTried" type="error" :bordered="false" :show-icon="false">{{ t('shell.record.extract.tamperAccepted') }}</NAlert>
      <NAlert v-if="receiveM.error.value" type="error" :bordered="false" :show-icon="false" data-testid="extract-recheck-error">{{ problemText(receiveM.error.value) }}</NAlert>
      <NAlert v-else-if="rechecked" type="success" :bordered="false" :show-icon="false" data-testid="extract-rechecked">{{ t('shell.record.extract.rechecked') }}</NAlert>
    </div>

    <template v-if="view" #actions>
      <ActionButton :disabled="!view.envelope" data-testid="extract-download" :label="t('shell.record.extract.download')" @click="downloadEnvelope(view)" />
      <ActionButton
        v-if="x?.direction === 'incoming' && view.envelope && !moment.isReplay"
        :disabled="receiveM.isPending.value"
        data-testid="extract-recheck"
        :label="t('shell.record.extract.recheck')"
        @click="recheck"
      />
      <ActionButton
        v-if="x?.direction === 'incoming' && tamperable && !moment.isReplay"
        :disabled="tamperM.isPending.value"
        :hint="t('shell.record.extract.tamperHint')"
        data-testid="extract-tamper"
        :label="t('shell.record.extract.tamper')"
        @click="tryTampered"
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

.mono {
  font-family: var(--ant-font-mono);
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent, inherit);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

p {
  margin: 0 0 var(--ant-space-2);
}
</style>
