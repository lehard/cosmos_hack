<script setup lang="ts">
/**
 * Несоответствие расследования ссылкой (шаг 10 показа): «НС-03 · Ф-003 · статус»;
 * щелчок — окно несоответствия справа (Д-70), там панель решения по правам
 * (например, «Переделка» у технолога). Пока карточка не пришла — «Несоответствие».
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NC_STATUS_TEXT, codeText, useNcCard } from '@/entities/nonconformity'

const props = defineProps<{ id: string }>()
const emit = defineEmits<{ open: [id: string] }>()
const { t } = useI18n()

const q = useNcCard(() => props.id)
const card = computed(() => q.data.value?.data ?? null)
const text = computed(() =>
  card.value
    ? [card.value.number, card.value.item_label, codeText(NC_STATUS_TEXT, card.value.status, t)].filter(Boolean).join(' · ')
    : t('common.words.nonconformity'),
)
</script>

<template>
  <button type="button" class="nc ant-wrap" :data-nc="id" data-testid="nc-link" @click="emit('open', id)">{{ text }}</button>
</template>

<style scoped>
.nc {
  padding: 0 var(--ant-space-2);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface);
  color: var(--ant-accent);
  font: inherit;
  font-size: var(--ant-fs-meta);
  text-align: left;
  cursor: pointer;
}

.nc:hover {
  background: var(--ant-surface-hover);
}
</style>
