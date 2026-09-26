<script setup lang="ts">
/**
 * Подпись на бумаге (FR-139, PRD §11.16 п. 3, AD-43) — второй равноправный путь.
 *
 * - «Печать» (подписант): бумажный экземпляр с QR, в QR — отпечаток документа
 *   `ant:doc:‹id›:‹отпечаток›` и ожидаемый подписант этапа; подписант ставит
 *   подпись ручкой. Лист с QR отрисовывает сервер из того же документа
 *   (AD-12) — интерфейс показывает содержимое QR и отпечаток для сверки.
 * - «Заверение» (второй человек): загрузить скан, указать учётный номер
 *   оригинала в архиве ОТК и заверить своей подписью уровня 2. Заверитель ≠
 *   подписант; скан с чужим QR сервер не принимает (`signing.qr_mismatch`).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NButton, NInput } from 'naive-ui'
import { canAttest, qrPayload, type DocumentHead } from '@/entities/document'
import { useProblemText } from '@/shared/i18n/problem'

const props = withDefaults(
  defineProps<{
    /** Документ, который подписывается. */
    document: Pick<DocumentHead, 'document_id' | 'doc_digest' | 'version'>
    /** Шаг: печать для подписанта или заверение скана. */
    mode: 'print' | 'attest'
    /** Ожидаемый подписант этапа (псевдоним). */
    expectedSigner?: string | null
    /** Текущий пользователь (псевдоним) — для «заверитель ≠ подписант». */
    currentUser?: string | null
    /** Этап маршрута разрешает бумагу. */
    paperAllowed?: boolean
    busy?: boolean
    error?: unknown
  }>(),
  { expectedSigner: null, currentUser: null, paperAllowed: true, busy: false, error: undefined },
)
const emit = defineEmits<{
  /** Распечатать экземпляр с QR. */
  print: []
  /** Загрузить скан и заверить. */
  attest: [payload: { file: File; archive_no: string }]
}>()

const { t } = useI18n()
const problemText = useProblemText()

const qr = computed(() => qrPayload(props.document))
const file = ref<File | null>(null)
const archiveNo = ref('')
const attesterOk = computed(() => canAttest(props.currentUser, props.expectedSigner))
const canSubmit = computed(() => props.paperAllowed && attesterOk.value && !!file.value && !!archiveNo.value.trim() && !props.busy)

function onFile(e: Event): void {
  file.value = (e.target as HTMLInputElement).files?.[0] ?? null
}

function submit(): void {
  if (!canSubmit.value || !file.value) return
  emit('attest', { file: file.value, archive_no: archiveNo.value.trim() })
}
</script>

<template>
  <section class="paper" :data-mode="mode" data-testid="paper-sign">
    <h4>{{ t('documents.paper.title') }}</h4>
    <NAlert v-if="!paperAllowed" type="error" :bordered="false" :show-icon="false" data-testid="paper-forbidden">
      {{ t('errors.signing.paperForbidden') }}
    </NAlert>
    <template v-else>
      <p class="muted" :title="t('hints.paperSignature')">{{ t('documents.paper.qrContains') }}</p>
      <code class="qr" data-testid="qr-payload">{{ qr }}</code>
      <p class="muted">{{ t('documents.fingerprint') }}: <code>{{ document.doc_digest }}</code></p>
      <p v-if="expectedSigner" class="muted" data-testid="expected-signer">{{ t('widgets.signing.expectedSigner', { who: expectedSigner }) }}</p>

      <template v-if="mode === 'print'">
        <NButton type="primary" :disabled="busy" :loading="busy" data-testid="print" @click="emit('print')">
          {{ t('documents.paper.printWithQr') }}
        </NButton>
        <p class="muted">{{ t('documents.paper.instruction') }}</p>
      </template>

      <template v-else>
        <NAlert v-if="!attesterOk" type="warning" :bordered="false" :show-icon="false" data-testid="attester-is-signer">
          {{ t('documents.paper.attesterNotSigner') }}
        </NAlert>
        <label class="field">
          <span>{{ t('documents.paper.uploadScan') }}</span>
          <input type="file" accept="application/pdf,image/*" :disabled="!attesterOk || busy" data-testid="scan-file" @change="onFile" />
        </label>
        <label class="field">
          <span>{{ t('documents.paper.archiveNumber') }}</span>
          <NInput v-model:value="archiveNo" size="small" :disabled="!attesterOk || busy" data-testid="archive-no" />
        </label>
        <NButton type="primary" :disabled="!canSubmit" :loading="busy" data-testid="attest" @click="submit">
          {{ t('documents.paper.attestScan') }}
        </NButton>
      </template>
    </template>
    <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="paper-error">{{ problemText(error) }}</NAlert>
  </section>
</template>

<style scoped>
.paper {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: flex-start;
}

h4 {
  margin: 0;
}

.muted {
  margin: 0;
  color: #6b7280;
  font-size: 12px;
}

.qr {
  padding: 4px 8px;
  border: 1px dashed #8a8f98;
  border-radius: 4px;
  font-family: 'PT Mono', monospace;
  word-break: break-all;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
  font-size: 12px;
}
</style>
