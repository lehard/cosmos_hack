<script setup lang="ts">
/**
 * Вкладка стола: раскладка — CSS-сетка с именованными областями, слоты — виджеты
 * из реестра в своих областях, по порядку yaml. Плотность: слот ← вкладка ← стол.
 */
import { computed } from 'vue'
import { LAYOUT_AREAS, type Density, type DeskTab } from '@/entities/desk'
import WidgetHost from '@/widgets/WidgetHost.vue'

const props = defineProps<{ tab: DeskTab; density: Density }>()

const areas = computed(() => {
  const allowed: readonly string[] = LAYOUT_AREAS[props.tab.layout]
  const grouped = new Map<string, DeskTab['slots']>(allowed.map((a) => [a, []]))
  for (const slot of props.tab.slots) {
    // Область вне раскладки (make check это ловит) — в main, а не в никуда.
    grouped.get(allowed.includes(slot.area) ? slot.area : 'main')!.push(slot)
  }
  return [...grouped.entries()].filter(([, slots]) => slots.length)
})

const tabDensity = computed<Density>(() => props.tab.density ?? props.density)
</script>

<template>
  <div class="desk-grid" :class="`layout-${tab.layout}`" :data-layout="tab.layout">
    <div v-for="[area, slots] in areas" :key="area" class="area" :style="{ gridArea: area }" :data-area="area">
      <WidgetHost
        v-for="slot in slots"
        :key="slot.id"
        :widget="slot.widget"
        :slot-id="slot.id"
        :slice="slot.slice ?? {}"
        :density="slot.density ?? tabDensity"
        :data-slot="slot.id"
      />
    </div>
  </div>
</template>

<style scoped>
/* Широкая раскладка (PRD §3a): основная область + правая панель контекста. */
.desk-grid {
  display: grid;
  gap: 16px;
  align-items: start;
}

.area {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.layout-single {
  grid-template: 'main' auto / minmax(0, 1fr);
}

.layout-main-side {
  grid-template: 'main right' auto / minmax(0, 1fr) minmax(320px, 400px);
}

.layout-queue-main-side {
  grid-template: 'left main right' auto / minmax(280px, 360px) minmax(0, 1fr) minmax(320px, 400px);
}

.layout-overview {
  grid-template:
    'top top' auto
    'main right' auto
    'bottom bottom' auto / minmax(0, 1fr) minmax(320px, 400px);
}

@media (max-width: 1279px) {
  .desk-grid {
    grid-template: none;
  }

  .area {
    grid-area: auto !important;
  }
}
</style>
