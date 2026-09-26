<script setup lang="ts">
/**
 * Виджет «Требуется ваше решение» (FR-136) — стол редкого подписанта:
 * представитель заказчика и согласующий видят одну подготовленную карточку
 * (или список, если ждут несколько) и подписывают с неё без доступа к
 * остальным экранам.
 *
 * Запросы решения и подпись этапа — операции модуля documents (эпик 28).
 * Подпись: окно уровня 2 → агент токена или ключ в браузере через порт
 * подписи над содержимым документа (signing_payload_b64); без агента — бумага с QR (FR-139): печать
 * здесь (печатная форма сервера — в новой вкладке); если бумажная подпись этапа ждёт заверения текущим пользователем —
 * загрузка скана и заверение (заверитель ≠ подписант). Паспорт изделия —
 * правым окном записи (Д-70), не панелью сбоку.
 */
import { computed, ref, watch } from 'vue'
import { paperAllowed, useAttestPaper, useDecisionRequests, useDeclineDocument, usePrintPaper, useSignDocument, type SummaryField } from '@/entities/document'
import { useSession } from '@/entities/session'
import { useDrillDown } from '@/features/drill-down'
import { PaperSignPanel, SignDialog, payloadTypeOf, useSigningPort } from '@/features/sign-decision'
import { backendModeOf } from '@/shared/api/response'
import { openPrintWindow } from '@/shared/lib/print-window'
import { useProblemText } from '@/shared/i18n/problem'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { useServerNow } from '@/shared/model/server-clock'
import { ActionButton, WidgetFrame } from '@/shared/ui'
import DecisionRequestView from './DecisionRequestView.vue'

defineProps<WidgetProps>()
const problemText = useProblemText()
const moment = useMomentStore()
const port = useSigningPort()
const drill = useDrillDown()

const query = useDecisionRequests()
const list = computed(() => query.data.value?.data ?? null)
const index = ref(0)
watch(list, () => (index.value = 0))
const request = computed(() => list.value?.[index.value] ?? null)
const myStage = computed(() => request.value?.document.route.find((s) => s.stage === request.value?.my_stage) ?? null)

// «Сейчас» — по часам сервера (Ant-Now); в воспроизведении — момент воспроизведения.
const serverNow = useServerNow()
const now = computed(() => (moment.asOf ? Date.parse(moment.asOf) : serverNow.value))

const session = useSession()
const currentUser = computed(() => session.data.value?.data?.user.id ?? null)
const attest = useAttestPaper()
const attestation = computed(() => request.value?.awaiting_attestation ?? null)

const signDoc = useSignDocument()
const decline = useDeclineDocument()
const print = usePrintPaper()
const dialog = ref<'confirm' | 'paper' | null>(null)
const error = ref<unknown>(undefined)

/** Заголовок команды над документом (AD-39): seq документа, версия политики и рабочее место сеанса. */
function commandHeader() {
  const s = session.data.value?.data
  return {
    basis_seq: request.value?.document.basis_seq ?? 0,
    policy_seq: s?.policy_seq ?? 0,
    ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
  }
}

/** Сводка уровня 2: документ, изделие, предложение, этап, отпечаток. */
const summary = computed<SummaryField[]>(() => {
  const r = request.value
  if (!r) return []
  return [
    { labelKey: 'widgets.signing.fields.action', value: r.proposal.summary },
    { labelKey: 'common.words.item', value: r.proposal.item_label },
    { labelKey: 'documents.route.title', valueKey: 'documents.route.stage', valueParams: { n: r.my_stage ?? 0 } },
    { labelKey: 'documents.fingerprint', value: r.document.doc_digest },
    { labelKey: 'widgets.signing.fields.level', valueKey: 'decisions.signature.level2' },
  ]
})

async function signWithAgent(): Promise<void> {
  const r = request.value
  if (!r || r.my_stage == null) return
  error.value = undefined
  try {
    // Содержимое документа от сервера (AD-12, AD-14): расширение подписывает
    // его и само сверяет отпечаток с doc_digest; по одному отпечатку не подписывает.
    const envelope = await port.sign({
      level: 2,
      payload_type: payloadTypeOf('document-signature'),
      payload_b64: r.document.signing_payload_b64 ?? '',
      template_ref: r.document.template_ref,
      expected_doc_digest: r.document.doc_digest,
    })
    const sig = envelope.signatures[0]
    await signDoc.mutateAsync({
      document_id: r.document.document_id,
      version: r.document.version,
      stage: r.my_stage,
      key_ref: sig?.keyid ?? '',
      signature_b64: sig?.sig ?? '',
      doc_digest: r.document.doc_digest,
      ...commandHeader(),
      signature: envelope,
    })
    dialog.value = null
  } catch (err) {
    error.value = err
  }
}

/** Распечатать с QR: запись «напечатан» и печатная форма сервера в новой вкладке (FR-139). */
async function printPaper(): Promise<void> {
  const r = request.value
  if (!r) return
  const tab = openPrintWindow()
  try {
    const res = await print.mutateAsync({ document_id: r.document.document_id, version: r.document.version, ...commandHeader() })
    tab.write(res.data.html)
  } catch {
    // Ошибку показывает окно подписи (print.error).
    tab.close()
  }
}

/** Заверить бумажную подпись: скан и учётный номер оригинала (FR-139, AD-43). */
function attestScan(payload: { file: File; archive_no: string }): void {
  const r = request.value
  const a = attestation.value
  if (!r || !a) return
  attest.mutate({
    document_id: r.document.document_id,
    version: r.document.version,
    stage: a.stage,
    signer_person_id: a.signer,
    doc_digest: r.document.doc_digest,
    item_id: r.proposal.item_id,
    ...commandHeader(),
    ...payload,
  })
}

function sendDecline(comment: string): void {
  const r = request.value
  if (!r || r.my_stage == null) return
  decline.mutate({
    document_id: r.document.document_id,
    version: r.document.version,
    stage: r.my_stage,
    doc_digest: r.document.doc_digest,
    comment,
    ...commandHeader(),
  })
}
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(query.data.value)"
    :loading="query.isPending.value && !list"
    :error="list ? undefined : query.error.value"
    :empty="!!list && !list.length"
    empty-key="empty.queueEmpty"
    :data-widget="widgetId"
  >
    <nav v-if="list && list.length > 1" class="switch" data-testid="requests-nav">
      <ActionButton
        v-for="(r, i) in list"
        :key="r.document.document_id"
        size="small"
        :type="i === index ? 'primary' : 'default'"
        :secondary="i !== index"
        @click="index = i"
        :label="r.proposal.item_label"
      />
    </nav>
    <DecisionRequestView
      v-if="request"
      :request="request"
      :now="now"
      :can-act="!moment.isReplay"
      :busy="signDoc.isPending.value || decline.isPending.value"
      :density="density"
      @sign="dialog = 'confirm'"
      @decline="sendDecline"
      @open-item="(id) => drill.open({ entity: 'item', id })"
    />
    <PaperSignPanel
      v-if="request && attestation"
      class="attest"
      :document="request.document"
      mode="attest"
      :expected-signer="attestation.signer"
      :current-user="currentUser"
      :busy="attest.isPending.value"
      :error="attest.error.value ?? undefined"
      data-testid="attest-panel"
      @attest="attestScan"
    />
    <p v-if="decline.error.value" class="error" data-testid="decline-error">{{ problemText(decline.error.value) }}</p>
    <SignDialog
      :show="dialog !== null"
      :stage="dialog === 'paper' ? 'paper' : 'confirm'"
      :summary="summary"
      :token-status="port.status.value"
      :paper-allowed="paperAllowed(myStage)"
      :busy="signDoc.isPending.value || print.isPending.value"
      :error="error ?? print.error.value ?? undefined"
      :document="request?.document ?? null"
      :expected-signer="request?.expected_signer ?? null"
      @confirm-token="signWithAgent"
      @sign-paper="dialog = 'paper'"
      @print="printPaper"
      @close="dialog = null"
    />
  </WidgetFrame>
</template>

<style scoped>
.switch {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.error {
  color: var(--ant-status-danger);
}

.attest {
  margin-top: 12px;
  padding-top: 8px;
  border-top: 1px solid var(--ant-border);
}
</style>
