<script setup lang="ts">
/**
 * Плашка «словами и тоном» для кодов журнала вне осей статуса изделия (вид
 * записи, статус подписи, вердикт верификатора): вид — как у StatusTag
 * (мягкий фон тона и цветная точка), текст — готовый, из словаря виджета.
 */
import type { StatusTone } from '@/shared/api/generated/statuses'

defineProps<{ label: string; tone: StatusTone; title?: string }>()
</script>

<template>
  <span class="tone-tag" :data-tone="tone" :title="title ?? label" :style="{ background: `var(--ant-status-${tone}-soft)` }">
    <span class="dot" :style="{ background: `var(--ant-status-${tone})` }" aria-hidden="true" />
    <span class="ant-ellipsis">{{ label }}</span>
  </span>
</template>

<style scoped>
.tone-tag {
  display: inline-flex;
  gap: var(--ant-space-1);
  align-items: center;
  max-width: 100%;
  min-width: 0;
  padding: 0 var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
  color: var(--ant-n-800);
  font-size: var(--ant-fs-meta);
  vertical-align: middle;
}

.dot {
  flex: none;
  width: var(--ant-space-2);
  height: var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
}
</style>
