<script setup lang="ts">
/**
 * Модальное окно подписи решения (FR-66, FR-139): шаг «проверьте перед
 * подписью» (уровень 2) и, если выбрана бумага, шаг «распечатать с QR».
 * Состояние шагов держит виджет; окно только показывает.
 */
import { useI18n } from 'vue-i18n'
import { NModal } from 'naive-ui'
import type { DocumentHead, SummaryField } from '@/entities/document'
import type { TokenStatus } from '@/shared/lib/token-agent'
import PaperSignPanel from './PaperSignPanel.vue'
import SignConfirmPanel from './SignConfirmPanel.vue'

withDefaults(
  defineProps<{
    show: boolean
    /** Шаг: подтверждение или печать бумажного экземпляра. */
    stage: 'confirm' | 'paper'
    summary: SummaryField[]
    tokenStatus: TokenStatus
    paperAllowed?: boolean
    busy?: boolean
    error?: unknown
    /** Документ решения для бумажного пути. */
    document?: Pick<DocumentHead, 'document_id' | 'doc_digest' | 'version'> | null
    expectedSigner?: string | null
    batch?: number
    /** Демо без агента токена — см. SignConfirmPanel. */
    demoUnsigned?: boolean
  }>(),
  { paperAllowed: true, busy: false, error: undefined, document: null, expectedSigner: null, batch: 1, demoUnsigned: false },
)
const emit = defineEmits<{
  'confirm-token': []
  'sign-paper': []
  'confirm-unsigned': []
  print: []
  close: []
}>()

const { t } = useI18n()
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('decisions.signature.signClosingDecision')"
    :mask-closable="!busy"
    style="width: min(560px, 96vw)"
    @update:show="(v: boolean) => !v && emit('close')"
  >
    <SignConfirmPanel
      v-if="stage === 'confirm'"
      :summary="summary"
      :token-status="tokenStatus"
      :paper-allowed="paperAllowed"
      :busy="busy"
      :error="error"
      :batch="batch"
      :demo-unsigned="demoUnsigned"
      @confirm-token="emit('confirm-token')"
      @confirm-unsigned="emit('confirm-unsigned')"
      @sign-paper="emit('sign-paper')"
      @cancel="emit('close')"
    />
    <PaperSignPanel
      v-else-if="document"
      :document="document"
      mode="print"
      :expected-signer="expectedSigner"
      :paper-allowed="paperAllowed"
      :busy="busy"
      :error="error"
      @print="emit('print')"
    />
  </NModal>
</template>
