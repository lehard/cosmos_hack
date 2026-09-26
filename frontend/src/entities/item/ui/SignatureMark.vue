<script setup lang="ts">
/**
 * Подпись записи и результат её автоматической проверки (FR-42, FR-68, FR-139):
 * кто подписал, способ (агент токена / бумага / устройство), уровень, итог
 * проверки. «Ключ недоступен» ≠ «подпись действительна»; бумажная подпись
 * показывает заверителя и учётный номер оригинала.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { SIGNATURE_CHECK, SIGNATURE_METHOD_TEXT, signatureColor } from '../model/passport'
import type { RecordSignature } from '../model/types'

const props = defineProps<{ signature: RecordSignature }>()
const { t } = useI18n()

const check = computed(() => SIGNATURE_CHECK[props.signature.verification])
const who = computed(() => props.signature.signer ?? t('common.words.unknown'))
const paperLine = computed(() =>
  props.signature.method === 'paper'
    ? t('documents.paper.signedOnPaper', {
        attester: props.signature.attested_by ?? t('common.words.unknown'),
        archiveNo: props.signature.paper_original_no ?? t('common.words.unknown'),
      })
    : null,
)
</script>

<template>
  <span class="signature" :data-verification="signature.verification" :data-method="signature.method">
    <span class="dot" :style="{ background: signatureColor(signature) }" aria-hidden="true" />
    <span class="check">{{ t(check.key) }}</span>
    <span class="meta">
      {{ t('timeline.marks.signedBy', { who }) }} · {{ t(SIGNATURE_METHOD_TEXT[signature.method]) }} ·
      {{ t('widgets.passport.signature.level', { level: signature.level }) }}
    </span>
    <span v-if="signature.stamp_id" class="meta">{{ t('audit.digitalStampOfInspector') }}: {{ signature.stamp_id }}</span>
    <span v-if="paperLine" class="meta" data-testid="paper-line">{{ paperLine }}</span>
  </span>
</template>

<style scoped>
.signature {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px 6px;
  align-items: baseline;
  font-size: 12px;
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.check {
  font-weight: 600;
}

.meta {
  color: #6b7280;
}
</style>
