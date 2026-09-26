<script setup lang="ts">
/**
 * Пустое состояние: значок, заголовок, пояснение, действие (слот).
 * «Нет записей» — это факт о данных, а не ошибка и не «норма» (NFR-UI-4).
 *
 * Когда брать: список, таблица или виджет без данных; заготовка виджета.
 */
import { NIcon } from 'naive-ui'
import { Inbox } from '@vicons/tabler'

withDefaults(
  defineProps<{
    /** Главная строка. */
    title: string
    description?: string
    /** Меньше отступов — внутри таблицы или узкой панели. */
    compact?: boolean
  }>(),
  { description: undefined, compact: false },
)
</script>

<template>
  <div class="empty-state" :class="{ 'is-compact': compact }">
    <div class="icon ant-box" aria-hidden="true">
      <slot name="icon"><NIcon :size="compact ? 20 : 28"><Inbox /></NIcon></slot>
    </div>
    <p class="title ant-wrap">{{ title }}</p>
    <p v-if="description" class="description ant-wrap">{{ description }}</p>
    <div v-if="$slots.default" class="action ant-box"><slot /></div>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  align-items: center;
  min-width: 0;
  padding: var(--ant-space-8) var(--ant-space-4);
  color: var(--ant-text-3);
  text-align: center;
}

.is-compact {
  padding: var(--ant-space-4) var(--ant-space-3);
}

.icon {
  display: flex;
  color: var(--ant-n-400);
}

.title {
  color: var(--ant-text-2);
  font-size: var(--ant-fs-body);
}

.description {
  max-width: 48ch;
  font-size: var(--ant-fs-meta);
}

.action {
  margin-top: var(--ant-space-2);
}
</style>
