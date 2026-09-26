<script setup lang="ts">
/**
 * Виджет «Требуется ваше решение» (FR-136) — стол редкого подписанта:
 * представитель заказчика и согласующий видят одну подготовленную карточку
 * (или список, если ждут несколько) и подписывают с неё без доступа к
 * остальным экранам.
 *
 * Операций модуля documents в контракте пока нет — запрос и подпись стоят на
 * заглушке `api.not_implemented` (entities/document). Подпись: окно уровня 2 →
 * агент токена через порт подписи; без агента — бумага с QR (FR-139): печать
 * здесь, заверение скана — вторым человеком.
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { NButton } from 'naive-ui'
import { paperAllowed, useDecisionRequests, useDeclineDocument, usePrintPaper, useSignDocument, type SummaryField } from '@/entities/document'
import { SignDialog, payloadTypeOf, useSigningPort } from '@/features/sign-decision'
import { backendModeOf } from '@/shared/api/response'
import { useProblemText } from '@/shared/i18n/problem'
import type { WidgetProps } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { WidgetFrame } from '@/shared/ui'
import DecisionRequestView from './DecisionRequestView.vue'

defineProps<WidgetProps>()
const problemText = useProblemText()
const moment = useMomentStore()
const port = useSigningPort()

const query = useDecisionRequests()
const list = computed(() => query.data.value?.data ?? null)
const index = ref(0)
watch(list, () => (index.value = 0))
const request = computed(() => list.value?.[index.value] ?? null)
const myStage = computed(() => request.value?.document.route.find((s) => s.stage === request.value?.my_stage) ?? null)

const tick = ref(Date.now())
const timer = setInterval(() => (tick.value = Date.now()), 30_000)
onBeforeUnmount(() => clearInterval(timer))
const now = computed(() => (moment.asOf ? Date.parse(moment.asOf) : tick.value))

const signDoc = useSignDocument()
const decline = useDeclineDocument()
const print = usePrintPaper()
const dialog = ref<'confirm' | 'paper' | null>(null)
const error = ref<unknown>(undefined)

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
    const envelope = await port.sign({
      level: 2,
      payload_type: payloadTypeOf('document-signature'),
      payload_b64: '',
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
    })
    dialog.value = null
  } catch (err) {
    error.value = err
  }
}

function printPaper(): void {
  const r = request.value
  if (!r) return
  print.mutate({ document_id: r.document.document_id, version: r.document.version })
}

function sendDecline(comment: string): void {
  const r = request.value
  if (!r || r.my_stage == null) return
  decline.mutate({ document_id: r.document.document_id, version: r.document.version, stage: r.my_stage, comment })
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
      <NButton
        v-for="(r, i) in list"
        :key="r.document.document_id"
        size="small"
        :type="i === index ? 'primary' : 'default'"
        :secondary="i !== index"
        @click="index = i"
      >
        {{ r.proposal.item_label }}
      </NButton>
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
  color: #d64545;
}
</style>
