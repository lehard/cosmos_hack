<script setup lang="ts">
/**
 * Материал доказательства сигнала (AD-23, FR-102): кадр контроля по адресу
 * содержимого — метаданные (`materials.material.read`) и байты
 * (`materials.material.content`). Иллюстрация из открытого набора помечена
 * «ИЛЛЮСТРАЦИЯ» и подписана: это не снимок этого изделия (NFR-UI-4).
 * Щелчок по кадру — показать крупно / вернуть.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMaterialsMaterialContent, useMaterialsMaterialRead } from '@/shared/api/generated/client'

const props = defineProps<{ address: string }>()
const { t, d } = useI18n()

const infoQ = useMaterialsMaterialRead(() => props.address, { query: { retry: false } })
const info = computed(() => (infoQ.data.value?.status === 200 ? infoQ.data.value.data : null))
const isImage = computed(() => !!info.value?.media_type.startsWith('image/'))
const contentQ = useMaterialsMaterialContent(() => props.address, { query: { retry: false, enabled: isImage } })
const src = computed(() => {
  const r = contentQ.data.value
  return info.value && r?.status === 200 && typeof r.data === 'string' ? `data:${info.value.media_type};base64,${r.data}` : null
})
const failed = computed(() => !!infoQ.error.value || !!contentQ.error.value)
const zoomed = ref(false)
</script>

<template>
  <figure class="material" :data-zoomed="zoomed || undefined" :data-illustration="info?.is_illustration || undefined" data-testid="evidence-material">
    <button v-if="src" type="button" class="frame" :title="zoomed ? t('ncCard.material.zoomOut') : t('ncCard.material.zoomIn')" @click="zoomed = !zoomed">
      <img :src="src" :alt="t('ncCard.material.alt')" />
      <span v-if="info?.is_illustration" class="badge">{{ t('empty.illustration') }}</span>
    </button>
    <div v-else class="frame placeholder ant-wrap">
      {{ failed ? t('ncCard.material.unavailable') : isImage || infoQ.isPending.value ? t('ncCard.material.loading') : t('ncCard.material.notImage') }}
    </div>
    <figcaption class="caption ant-wrap">
      <template v-if="info?.is_illustration">{{ t('empty.illustrationBadge') }}</template>
      <template v-else-if="info?.captured_at">{{ t('ncCard.material.capturedAt', { time: d(new Date(info.captured_at), 'dateTime') }) }}</template>
      <template v-if="info?.provenance_note"> · {{ info.provenance_note }}</template>
    </figcaption>
  </figure>
</template>

<style scoped>
.material {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  margin: 0;
}

.frame {
  position: relative;
  display: block;
  width: 100%;
  padding: 0;
  overflow: hidden;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface-subtle);
  cursor: zoom-in;
}

.material[data-zoomed] .frame {
  cursor: zoom-out;
}

.frame img {
  display: block;
  width: 100%;
  max-height: 220px;
  object-fit: contain;
}

.material[data-zoomed] .frame img {
  max-height: none;
}

.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 120px;
  padding: var(--ant-space-3);
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  cursor: default;
}

.badge {
  position: absolute;
  top: var(--ant-space-2);
  left: var(--ant-space-2);
  padding: 0 var(--ant-space-2);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-status-attention);
  color: var(--ant-n-900);
  font-size: var(--ant-fs-xs);
  font-weight: var(--ant-fw-bold);
}

.caption {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}
</style>
