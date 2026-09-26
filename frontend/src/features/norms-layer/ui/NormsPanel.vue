<script setup lang="ts">
/**
 * Опоры выбранного шага на живой карте (FR-156): пункт стандарта, что
 * проверить и что делает система на этом шаге. Показывается при включённом
 * слое «нормы» над карточкой узла.
 */
import { useI18n } from 'vue-i18n'
import { EmptyState, SectionPanel } from '@/shared/ui'
import type { StepNorms } from '../model/norms'

defineProps<{ norms: StepNorms | null }>()
const { t } = useI18n()
</script>

<template>
  <SectionPanel variant="card" :title="t('normsLayer.title')" :subtitle="norms?.name" data-testid="norms-panel">
    <EmptyState v-if="!norms" compact :title="t('normsLayer.none')" />
    <ul v-else class="anchors">
      <li v-for="(a, i) in norms.anchors" :key="i" class="anchor ant-box" :data-standard="a.standard">
        <strong class="ant-wrap">{{ t('normsLayer.clause', { standard: a.standard, clause: a.clause }) }}</strong>
        <p v-if="a.check" class="ant-wrap"><span class="label">{{ t('normsLayer.check') }}:</span> {{ a.check }}</p>
        <p v-if="a.systemAction" class="ant-wrap"><span class="label">{{ t('normsLayer.systemAction') }}:</span> {{ a.systemAction }}</p>
      </li>
    </ul>
  </SectionPanel>
</template>

<style scoped>
.anchors {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.anchor p {
  margin: var(--ant-space-1) 0 0;
  font-size: var(--ant-fs-body);
}

.label {
  color: var(--ant-text-3);
}
</style>
