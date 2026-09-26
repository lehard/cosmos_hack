<script setup lang="ts">
/**
 * Изделия-точки в текущем узле (FR-2, FR-9): клик по точке открывает паспорт.
 * Цвет — сводный статус изделия или, в режиме инцидента, статус относительно
 * инцидента; подпись статуса — в подсказке, текстом словаря статусов.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MapItem } from '@/entities/live-map'
import { DOTS_PER_NODE, dotLook, sortForDots } from '../model/overlays'

const props = defineProps<{ items: readonly MapItem[]; incidentMode: boolean }>()
const emit = defineEmits<{ open: [itemId: string]; more: [] }>()
const { t } = useI18n()

const shown = computed(() =>
  sortForDots(props.items, props.incidentMode).slice(0, DOTS_PER_NODE).map((item) => {
    const look = dotLook(item, props.incidentMode)
    const status = look.statusKey ? t(look.statusKey) : ''
    return { item, look, title: status ? `${item.label} — ${status}` : item.label }
  }),
)
const rest = computed(() => Math.max(0, props.items.length - DOTS_PER_NODE))
</script>

<template>
  <div class="node-dots">
    <button
      v-for="d in shown"
      :key="d.item.item_id"
      type="button"
      class="dot"
      :class="{ dimmed: d.look.dimmed, transit: d.item.position === 'in_transit' }"
      :style="{ background: d.look.color }"
      :title="d.title"
      :aria-label="d.title"
      :data-item="d.item.item_id"
      :data-tone="d.look.tone"
      @click.stop="emit('open', d.item.item_id)"
    />
    <button v-if="rest" type="button" class="more" :data-more="rest" @click.stop="emit('more')">+{{ rest }}</button>
  </div>
</template>

<style scoped>
.node-dots {
  display: flex;
  flex-wrap: wrap;
  gap: 3px;
  max-width: 110px;
}

.dot {
  width: 12px;
  height: 12px;
  padding: 0;
  border: 2px solid var(--ant-surface);
  border-radius: 50%;
  box-shadow: 0 0 0 1px rgb(0 0 0 / 25%);
  cursor: pointer;
}

.dot.dimmed {
  opacity: 0.35;
}

/* Передача между цехами — изделие в перемещении (FR-130). */
.dot.transit {
  border-style: dashed;
}

.more {
  padding: 0 3px;
  border: 0;
  background: transparent;
  color: var(--ant-n-700);
  font: 600 11px/12px 'PT Sans', sans-serif;
  cursor: pointer;
}
</style>
