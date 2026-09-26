<script setup lang="ts">
/**
 * Окно партнёра (Д-70; эпик 41, AD-19): код предприятия, адрес порта
 * межзаводского обмена, корни доверия из нашего акта регистрации (доверие
 * ключам сотрудников партнёра выводится из него), акт регистрации, канал и
 * выписки этого партнёра. Внизу — «Отправить выписку паспорта» (изделие и
 * документ выписки с закрытым маршрутом): отправляет роль outbox, квитанция
 * придёт событием.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput } from 'naive-ui'
import { shortDigest, useExtracts, usePartners, useSendExtract } from '@/entities/federation'
import { useSession } from '@/entities/session'
import { useDrillDown, type DrillRef } from '@/features/drill-down'
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

const listQ = usePartners()
const partner = computed(() => listQ.data.value?.data.items.find((p) => p.partner_code === props.id) ?? null)
const extractsQ = useExtracts(computed(() => props.id))
const extracts = computed(() => extractsQ.data.value?.data.items ?? [])
const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : '—')

const sendM = useSendExtract()
const item = ref('')
const doc = ref('')
const sent = ref(false)
watch(
  () => props.id,
  () => {
    item.value = doc.value = ''
    sent.value = false
    sendM.reset()
  },
)

async function send(): Promise<void> {
  if (!item.value.trim() || !doc.value.trim()) return
  sent.value = false
  const s = session.data.value?.data
  try {
    await sendM.mutateAsync({
      command_id: newCommandId(),
      basis_seq: 0,
      policy_seq: s?.policy_seq ?? 0,
      partner_code: props.id,
      kind: 'passport_extract',
      subject: { entity: 'item', id: item.value.trim() },
      document_id: doc.value.trim(),
    })
    sent.value = true
  } catch {
    // Текст — из sendM.error.
  }
}

const openExtract = (digest: string) => drill.open({ entity: 'extract', id: digest } as unknown as DrillRef)
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('shell.record.partner.kind')"
    :number="partner?.name ?? id"
    :subtitle="partner?.endpoint ?? ''"
    :loading="listQ.isPending.value && !partner"
    data-record="partner"
    @close="emit('close')"
  >
    <EmptyState v-if="!partner && listQ.error.value" compact :title="t('errors.loadFailed')" :description="problemText(listQ.error.value)" />
    <EmptyState v-else-if="!partner" compact :title="t('shell.record.partner.notFound')" />
    <div v-else class="sections ant-box">
      <SectionPanel :title="t('shell.record.partner.trust')" :subtitle="t('shell.record.partner.trustHint')" variant="plain" :padded="false">
        <KeyValueList>
          <KeyValue :label="t('widgets.federation.code')" :value="partner.partner_code" mono />
          <KeyValue :label="t('widgets.federation.endpoint')" :value="partner.endpoint ?? null" mono />
          <KeyValue :label="t('shell.record.partner.act')" :value="partner.document_id" mono />
          <KeyValue :label="t('shell.record.partner.registered')" :value="time(partner.registered_at)" />
          <KeyValue :label="t('widgets.federation.channel')" :value="t(`widgets.federation.channels.${partner.channel}`)" />
        </KeyValueList>
        <p class="label">{{ t('widgets.federation.rootsCol') }}</p>
        <p v-for="f in partner.root_fingerprints" :key="f" class="mono ant-wrap" data-testid="partner-root">{{ f }}</p>
      </SectionPanel>

      <SectionPanel :title="t('widgets.federation.extracts')" variant="plain" :padded="false">
        <EmptyState v-if="!extracts.length" compact :title="t('widgets.federation.noExtracts')" />
        <DataTable v-else>
          <tbody>
            <tr v-for="e in extracts" :key="e.direction + e.extract_digest" class="row" tabindex="0" @click="openExtract(e.extract_digest)" @keydown.enter="openExtract(e.extract_digest)">
              <td><span class="ant-ellipsis">{{ t(`widgets.federation.directions.${e.direction}`) }}</span></td>
              <td><span class="ant-wrap">{{ e.label || shortDigest(e.extract_digest) }}</span></td>
              <td><span class="ant-ellipsis">{{ e.direction === 'incoming' ? t(`widgets.federation.origin.${e.origin_status}`) : '' }}</span></td>
            </tr>
          </tbody>
        </DataTable>
      </SectionPanel>

      <SectionPanel v-if="!moment.isReplay" :title="t('shell.record.partner.send')" :subtitle="t('shell.record.partner.sendHint')" variant="plain" :padded="false">
        <div class="form">
          <NInput v-model:value="item" :placeholder="t('shell.record.partner.item')" :maxlength="128" data-testid="send-item" />
          <NInput v-model:value="doc" :placeholder="t('shell.record.partner.document')" :maxlength="128" data-testid="send-doc" />
        </div>
      </SectionPanel>
      <NAlert v-if="sendM.error.value" type="error" :bordered="false" :show-icon="false">{{ problemText(sendM.error.value) }}</NAlert>
      <NAlert v-else-if="sent" type="success" :bordered="false" :show-icon="false" data-testid="send-done">{{ t('shell.record.partner.sent') }}</NAlert>
    </div>

    <template v-if="partner && !moment.isReplay" #actions>
      <ActionButton type="primary" :disabled="sendM.isPending.value || !item.trim() || !doc.trim()" data-testid="send-submit" :label="t('shell.record.partner.sendSubmit')" @click="send" />
    </template>
  </RecordDrawer>
</template>

<style scoped>
.sections {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-6);
}

.form {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.row {
  cursor: pointer;
}

.label {
  margin: var(--ant-space-3) 0 var(--ant-space-1);
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.mono {
  margin: 0;
  font-family: var(--ant-font-mono);
  font-size: var(--ant-fs-meta);
}
</style>
