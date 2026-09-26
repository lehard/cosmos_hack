<script setup lang="ts">
/**
 * Плитки показателей стола руководителя (FR-8, FR-86, FR-89; кейс §2.4):
 * название от сервера (Д-11 — «Прохождение контроля с первого раза»),
 * значение в своих единицах, сравнение с прошлым таким же периодом. Нажатие —
 * раскрытие числа до исходных записей (FR-7).
 *
 * Вёрстка неподвижная: сетка ровных колонок, плитки ряда одной высоты,
 * заголовок всегда занимает место двух строк, число — всегда на одной высоте,
 * строка прошлого периода прижата к низу. Оговорки (определение показателя,
 * происхождение времени, смысл интервала — кейс §5.2, FR-88) не плашками, а в
 * подсказке значка рядом с заголовком.
 * «Оценка невозможна» — словами и прочерком, не ноль (NFR-UI-4).
 */
import { useI18n } from 'vue-i18n'
import { NIcon, NTooltip } from 'naive-ui'
import { InfoCircle } from '@vicons/tabler'
import type { MetricTile } from '@/shared/api/generated/model'
import { deltaOf, formatValue, meaningOf, metricHintKey, MetricNumber, originOf } from '@/entities/metric'
import type { Density } from '@/shared/config/widget'

defineProps<{ tiles: MetricTile[]; density: Density }>()
const emit = defineEmits<{ open: [tile: MetricTile] }>()
const { t, n } = useI18n()
const texts = { t, n: (v: number, f: string) => n(v, f) }

/** Текст значения для подсказки при обрезке. */
const valueText = (tile: MetricTile): string => (tile.unknown ? t('widgets.analytics.value.unknown') : formatValue(texts, tile.value))

/** Интервал длительности словами: пояснение сервера важнее общего названия. */
function intervalNote(tile: MetricTile): string | null {
  const m = meaningOf(tile.value)
  if (!m) return null
  if (m.note) return t('widgets.analytics.tiles.interval', { note: m.note })
  if (m.code === 'active_processing') return t('widgets.analytics.tiles.intervalActive')
  if (m.code === 'time_at_station') return t('widgets.analytics.tiles.intervalStation')
  return null
}

/** Происхождение времени словами (для длительностей без происхождения — явная оговорка). */
function originNote(tile: MetricTile): string | null {
  const o = originOf(tile.value)
  if (!o) return null
  if (o.code === 'reported_by_source' || o.code === 'computed_by_system') return t(`widgets.analytics.tiles.origin.${o.code}`)
  return t(o.key)
}

/** Строки подсказки плитки: определение, интервал, происхождение. Пусто — значка нет. */
function notes(tile: MetricTile): string[] {
  const key = metricHintKey(tile.metric_id)
  return [key ? t(key) : null, intervalNote(tile), originNote(tile)].filter((x): x is string => Boolean(x))
}

/** Происхождение времени не передано — значок подсказки требует внимания. */
const warn = (tile: MetricTile): boolean => Boolean(originOf(tile.value)?.warn)

/** «было 12 (+3)» — только если прошлое значение пришло и единицы совпадают. */
function previousText(tile: MetricTile): string | null {
  if (!tile.previous || tile.unknown) return null
  const d = deltaOf(tile.value, tile.previous)
  const was = formatValue(texts, tile.previous)
  if (d === null) return t('widgets.analytics.tiles.previous', { value: was })
  // Доли сравниваем в процентных пунктах, а не в процентах от процента.
  const delta =
    tile.value.unit === 'bp'
      ? t('widgets.analytics.tiles.percentPoints', { value: n(Math.abs(d) / 100, 'decimal2') })
      : formatValue(texts, { ...tile.value, value: Math.abs(tile.value.value - tile.previous.value) })
  return t('widgets.analytics.tiles.previousDelta', { value: was, sign: d > 0 ? '+' : d < 0 ? '−' : '±', delta })
}
</script>

<template>
  <ul class="tiles" :class="`density-${density}`">
    <li v-for="tile in tiles" :key="tile.metric_id" class="tile" :data-metric="tile.metric_id" :data-unknown="tile.unknown || undefined">
      <button
        type="button"
        class="tile-button"
        :aria-label="[tile.title, ...notes(tile), t('common.actions.drillDown')].join('. ')"
        @click="emit('open', tile)"
      >
        <span class="tile-head">
          <span class="tile-title ant-clamp-2" data-testid="tile-title" :title="tile.title">{{ tile.title }}</span>
          <NTooltip v-if="notes(tile).length" :style="{ maxWidth: 'var(--ant-w-side)' }">
            <template #trigger>
              <!-- Щелчок по значку — только подсказка, не раскрытие числа. -->
              <NIcon class="tile-info" :data-warn="warn(tile) || undefined" data-testid="tile-info" aria-hidden="true" @click.stop>
                <InfoCircle />
              </NIcon>
            </template>
            <span v-for="(line, i) in notes(tile)" :key="i" class="tile-note ant-wrap" data-testid="tile-note">{{ line }}</span>
          </NTooltip>
        </span>
        <MetricNumber class="tile-value" :value="tile.value" :unknown="tile.unknown" :show-origin="false" :title="valueText(tile)" />
        <span v-if="previousText(tile)" class="tile-previous ant-ellipsis" data-testid="previous" :title="previousText(tile)!">{{ previousText(tile) }}</span>
      </button>
    </li>
  </ul>
</template>

<style scoped>
/* Ровные колонки одной ширины; ряды — по самой высокой плитке, без «лесенки». */
.tiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(calc(var(--ant-space-10) * 4), 1fr));
  grid-auto-rows: 1fr;
  gap: var(--ant-space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.tile {
  display: flex;
  min-width: 0;
}

/* Три строки сетки: заголовок (всегда высотой в две строки), число, прошлый период (внизу). */
.tile-button {
  display: grid;
  grid-template-rows: auto auto 1fr;
  row-gap: var(--ant-space-1);
  width: 100%;
  min-width: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.tile-button:hover,
.tile-button:focus-visible {
  border-color: var(--ant-text-3);
}

.tile-head {
  display: flex;
  gap: var(--ant-space-1);
  align-items: flex-start;
  min-width: 0;
}

.tile-title {
  flex: 1 1 auto;
  min-height: calc(2em * var(--ant-lh-tight));
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  line-height: var(--ant-lh-tight);
}

.tile-info {
  flex: none;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  line-height: var(--ant-lh-tight);
  cursor: help;
}

.tile-info[data-warn] {
  color: var(--ant-status-attention-text);
}

.tile-note {
  display: block;
}

.tile-note + .tile-note {
  margin-top: var(--ant-space-1);
}

/* Число — одна строка на одной высоте во всех плитках; длинное обрезается (полный текст — в title). */
.tile-value {
  /* Элемент сетки: inline-flex числа становится блочным сам. */
  min-width: 0;
  overflow: hidden;
  font-size: var(--ant-fs-xxl);
  font-weight: var(--ant-fw-bold);
  line-height: var(--ant-lh-tight);
}

.tile-value :deep(.value) {
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.density-large .tile-value {
  font-size: var(--ant-fs-display);
}

.density-compact .tile-value {
  font-size: var(--ant-fs-xl);
}

.tile-previous {
  align-self: end;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
  line-height: var(--ant-lh-tight);
}
</style>
