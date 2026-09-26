<script setup lang="ts">
/**
 * Плитки показателей стола руководителя (FR-8, FR-86, FR-89; кейс §2.4):
 * название от сервера (Д-11 — «Прохождение контроля с первого раза»),
 * значение в своих единицах, происхождение времени, сравнение с прошлым таким
 * же периодом. Нажатие — раскрытие числа до исходных записей (FR-7).
 * «Оценка невозможна» — словами, не ноль (NFR-UI-4).
 */
import { useI18n } from 'vue-i18n'
import type { MetricTile } from '@/shared/api/generated/model'
import { deltaOf, formatValue, metricHintKey, MetricNumber } from '@/entities/metric'
import type { Density } from '@/shared/config/widget'

defineProps<{ tiles: MetricTile[]; density: Density }>()
const emit = defineEmits<{ open: [tile: MetricTile] }>()
const { t, n } = useI18n()

const hint = (tile: MetricTile) => {
  const key = metricHintKey(tile.metric_id)
  return key ? t(key) : undefined
}

/** «было 12 (+3)» — только если прошлое значение пришло и единицы совпадают. */
function previousText(tile: MetricTile): string | null {
  if (!tile.previous || tile.unknown) return null
  const d = deltaOf(tile.value, tile.previous)
  const was = formatValue({ t, n: (v, f) => n(v, f) }, tile.previous)
  if (d === null) return t('widgets.analytics.tiles.previous', { value: was })
  // Доли сравниваем в процентных пунктах, а не в процентах от процента.
  const delta =
    tile.value.unit === 'bp'
      ? t('widgets.analytics.tiles.percentPoints', { value: n(Math.abs(d) / 100, 'decimal2') })
      : formatValue({ t, n: (v, f) => n(v, f) }, { ...tile.value, value: Math.abs(tile.value.value - tile.previous.value) })
  return t('widgets.analytics.tiles.previousDelta', { value: was, sign: d > 0 ? '+' : d < 0 ? '−' : '±', delta })
}
</script>

<template>
  <ul class="tiles" :class="`density-${density}`">
    <li v-for="tile in tiles" :key="tile.metric_id" class="tile" :data-metric="tile.metric_id" :data-unknown="tile.unknown || undefined">
      <button type="button" class="tile-button" :title="hint(tile)" :aria-label="`${tile.title}: ${t('common.actions.drillDown')}`" @click="emit('open', tile)">
        <span class="tile-title" data-testid="tile-title">{{ tile.title }}</span>
        <MetricNumber class="tile-value" :value="tile.value" :unknown="tile.unknown" />
        <span v-if="previousText(tile)" class="tile-previous" data-testid="previous">{{ previousText(tile) }}</span>
      </button>
    </li>
  </ul>
</template>

<style scoped>
.tiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.tile-button {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  height: 100%;
  padding: 8px 10px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: #fff;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.tile-button:hover,
.tile-button:focus-visible {
  border-color: #6b7280;
}

.tile-title {
  color: #4b5563;
  font-size: 12px;
  line-height: 1.3;
}

.tile-value {
  font-size: 22px;
  font-weight: 600;
}

.density-large .tile-value {
  font-size: 28px;
}

.density-compact .tile-value {
  font-size: 18px;
}

.tile-previous {
  color: #6b7280;
  font-size: 11px;
}
</style>
