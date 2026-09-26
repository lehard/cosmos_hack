<script setup lang="ts">
/**
 * Подпись записи и итог её автоматической проверки (FR-42, FR-68, FR-139): кто
 * подписал, класс происхождения (устройство / личная / бумага / сервер…),
 * уровень, итог проверки. «Не проверяемо» и «не проверялась» ≠ «действительна»;
 * бумажная подпись показывает заверителя, подпись ключом — класс хранения
 * ключа (физический ключ / ключ в браузере, AD-14, Д-72).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { SIGNATURE_CHECK, SIGNATURE_CLASS_TEXT, signatureColor } from '../model/passport'
import type { RecordSignature } from '../model/types'

const props = defineProps<{ signature: RecordSignature }>()
const { t } = useI18n()

const check = computed(() => SIGNATURE_CHECK[props.signature.check])
</script>

<template>
  <span class="signature" :data-check="signature.check" :data-class="signature.class">
    <span class="dot" :style="{ background: signatureColor(signature) }" aria-hidden="true" />
    <span class="check">{{ t(check.key) }}</span>
    <span class="meta">
      {{ t('timeline.marks.signedBy', { who: signature.signer_id }) }} · {{ t(SIGNATURE_CLASS_TEXT[signature.class]) }} ·
      {{ t('widgets.passport.signature.level', { level: signature.level }) }}
    </span>
    <span v-if="signature.key_storage" class="meta" data-testid="key-storage">
      {{ t(`common.header.tokenStorage.${signature.key_storage}`) }}
    </span>
    <span v-if="signature.class === 'paper'" class="meta" data-testid="paper-line">
      {{ t('widgets.passport.signature.paperAttested', { attester: signature.attested_by ?? t('common.words.unknown') }) }}
    </span>
  </span>
</template>

<style scoped>
.signature {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px 6px;
  align-items: baseline;
  font-size: var(--ant-fs-meta);
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.check {
  font-weight: var(--ant-fw-bold);
}

.meta {
  color: var(--ant-text-3);
}
</style>
