<script setup lang="ts">
/**
 * Левое меню разделов (Д-73): пункты со значком и подписью, выбранный пункт —
 * акцентом. Развёрнуто по умолчанию; сворачивается в узкую полосу значков
 * (подпись — подсказкой справа), но не скрывается совсем — меню всегда на виду.
 *
 * Когда брать: переходы между разделами одного места (разделы стола роли).
 * Не брать: когда пункт один — меню не нужно; для действий над записью — окно
 * записи (RecordDrawer).
 */
import type { Component } from 'vue'
import { NIcon, NTooltip } from 'naive-ui'
import { ChevronsLeft, ChevronsRight } from '@vicons/tabler'

export interface SideNavItem {
  id: string
  /** Подпись пункта (готовый текст). */
  label: string
  icon: Component
  /** Дополнение к подсказке (горячая клавиша). */
  hint?: string
}

defineProps<{
  items: SideNavItem[]
  /** Выбранный пункт; нет — ничего не выделено (страница вне разделов). */
  active?: string
  /** Свёрнуто в полосу значков (v-model:collapsed). */
  collapsed: boolean
  /** Подпись меню для чтения с экрана. */
  label: string
  /** Подписи кнопки сворачивания. */
  collapseLabel: string
  expandLabel: string
}>()
const emit = defineEmits<{ select: [id: string]; 'update:collapsed': [value: boolean] }>()
</script>

<template>
  <nav class="side-nav" :class="{ collapsed }" :aria-label="label" :data-collapsed="collapsed">
    <ul class="items">
      <li v-for="x in items" :key="x.id">
        <NTooltip placement="right" :disabled="!collapsed && !x.hint" :delay="collapsed ? 0 : 600">
          <template #trigger>
            <button
              type="button"
              class="item"
              :data-item="x.id"
              :aria-current="x.id === active ? 'page' : undefined"
              :aria-label="collapsed ? x.label : undefined"
              @click="emit('select', x.id)"
            >
              <NIcon class="icon" size="20"><component :is="x.icon" /></NIcon>
              <span v-if="!collapsed" class="text ant-ellipsis">{{ x.label }}</span>
            </button>
          </template>
          <span class="ant-wrap">{{ collapsed ? x.label : '' }}<template v-if="x.hint">{{ collapsed ? ' · ' : '' }}{{ x.hint }}</template></span>
        </NTooltip>
      </li>
    </ul>
    <button
      type="button"
      class="item toggle"
      data-testid="side-nav-toggle"
      :aria-expanded="!collapsed"
      :title="collapsed ? expandLabel : collapseLabel"
      @click="emit('update:collapsed', !collapsed)"
    >
      <NIcon class="icon" size="20"><component :is="collapsed ? ChevronsRight : ChevronsLeft" /></NIcon>
      <span v-if="!collapsed" class="text ant-ellipsis">{{ collapseLabel }}</span>
    </button>
  </nav>
</template>

<style scoped>
.side-nav {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  width: var(--ant-w-nav);
  height: 100%;
  padding: var(--ant-space-3) var(--ant-space-2);
  border-right: 1px solid var(--ant-border);
  background: var(--ant-surface);
  transition: width 0.15s ease;
}

.side-nav.collapsed {
  width: var(--ant-w-nav-collapsed);
}

.items {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.item {
  display: flex;
  gap: var(--ant-space-3);
  align-items: center;
  width: 100%;
  min-width: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 0;
  border-radius: var(--ant-radius-md);
  background: none;
  color: var(--ant-text-2);
  font: inherit;
  font-size: var(--ant-fs-body);
  text-align: left;
  cursor: pointer;
}

.collapsed .item {
  justify-content: center;
  padding: var(--ant-space-2) 0;
}

.item:hover {
  background: var(--ant-surface-hover);
  color: var(--ant-text);
}

.item:focus-visible {
  outline: 2px solid var(--ant-accent);
  outline-offset: -2px;
}

.item[aria-current='page'] {
  background: var(--ant-accent-soft);
  color: var(--ant-accent);
  font-weight: var(--ant-fw-bold);
}

.icon {
  flex: none;
}

.text {
  min-width: 0;
}

.toggle {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-sm);
}
</style>
