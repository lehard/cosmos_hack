<script setup lang="ts">
/**
 * Схема шва (разбор «Камеры + ИИ», сцена 1): участки шва изделия лентой в
 * порядке справочника типа изделия (зоны вида «участок шва»); участок с
 * признаком подсвечен, у остальных — состояние проверки из паспорта изделия.
 * Обозначения — только из данных (названия зон), ничего не рисуем сверх них;
 * ширина — по контейнеру, без горизонтальной прокрутки.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SeamZone } from '../model/seam'


const props = defineProps<{ zones: readonly SeamZone[]; hits: readonly string[] }>()
const { t } = useI18n()
const hitSet = computed(() => new Set(props.hits))
const hitNames = computed(() => props.zones.filter((z) => hitSet.value.has(z.zone_id)).map((z) => z.name))
const tone = (z: SeamZone) => (hitSet.value.has(z.zone_id) ? 'hit' : z.status === 'inspected' ? 'ok' : z.status === 'stale' ? 'stale' : 'none')
</script>

<template>
  <figure class="seam" data-testid="seam-scheme">
    <ol class="strip" :aria-label="t('ncCard.seam.title')">
      <li v-for="(z, i) in zones" :key="z.zone_id" class="cell" :data-tone="tone(z)" :data-zone="z.zone_id" :title="`${z.name} · ${t(`ncCard.seam.tone.${tone(z)}`)}`">
        <span class="no">{{ i + 1 }}</span>
      </li>
    </ol>
    <figcaption class="caption ant-wrap">
      <template v-if="hitNames.length"><strong>{{ t('ncCard.seam.hit') }}:</strong> {{ hitNames.join(', ') }}</template>
      <template v-else>{{ t('ncCard.seam.noHit') }}</template>
      <span class="legend">
        <span class="swatch" data-tone="hit" />{{ t('ncCard.seam.tone.hit') }}
        <span class="swatch" data-tone="ok" />{{ t('ncCard.seam.tone.ok') }}
        <span class="swatch" data-tone="none" />{{ t('ncCard.seam.tone.none') }}
      </span>
    </figcaption>
  </figure>
</template>

<style scoped>
.seam {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  min-width: 0;
  margin: 0;
}

.strip {
  display: flex;
  gap: 3px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.cell {
  display: flex;
  flex: 1 1 0;
  align-items: center;
  justify-content: center;
  min-width: 0;
  height: 30px;
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface-subtle);
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
  font-variant-numeric: tabular-nums;
}

.cell[data-tone='ok'] {
  border-color: var(--ant-status-success);
  background: var(--ant-status-success-soft);
}

.cell[data-tone='stale'] {
  border-style: dashed;
}

.cell[data-tone='hit'] {
  border-color: var(--ant-status-danger);
  background: var(--ant-status-danger);
  color: var(--ant-surface);
  font-weight: var(--ant-fw-bold);
}

.caption {
  display: flex;
  flex-wrap: wrap;
  gap: 2px var(--ant-space-3);
  align-items: center;
  font-size: var(--ant-fs-meta);
}

.legend {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 2px var(--ant-space-2);
  align-items: center;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.swatch {
  display: inline-block;
  width: 10px;
  height: 10px;
  border: 1px solid var(--ant-border-strong);
  border-radius: 2px;
  background: var(--ant-surface-subtle);
}

.swatch[data-tone='hit'] {
  border-color: var(--ant-status-danger);
  background: var(--ant-status-danger);
}

.swatch[data-tone='ok'] {
  border-color: var(--ant-status-success);
  background: var(--ant-status-success-soft);
}
</style>
