<script setup lang="ts">
/**
 * Страница «Карта дефицита данных» (эпик 42; FR-143): по каждому виду
 * недостающих сведений разбора (FR-58) — в скольких расследованиях не
 * хватало, где, какая цифровизация закрыла бы пробел и насколько она сузила
 * бы область риска (по уже сделанным сужениям с основаниями; не с чем
 * сравнить — «оценка невозможна», а не ноль). Щелчок по строке — правое окно
 * (Д-70). Ниже — узлы, где оценка невозможна из-за отсутствия данных источника
 * (`data_gaps` счётчиков узлов).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeCounterSet } from '@/entities/metric'
import { useDataDeficit, useOpenRecord } from '@/entities/suggestion'
import { backendModeOf } from '@/shared/api/response'
import type { WidgetDataState, WidgetProps } from '@/shared/config/widget'
import { DataTable, EmptyState, SectionPanel, WidgetFrame } from '@/shared/ui'
import DeficitDrawer from './DeficitDrawer.vue'

defineProps<WidgetProps>()
const { t } = useI18n()
const q = useDataDeficit()
const nodes = useNodeCounterSet()
const map = computed(() => q.data.value?.data ?? null)
const rows = computed(() => map.value?.rows ?? [])
const gaps = computed(() => nodes.data.value?.data_gaps ?? [])
/** Есть узлы без данных — «оценка невозможна», а не «норма». */
const state = computed<WidgetDataState>(() => (gaps.value.length ? 'unable_to_assess' : 'normal'))

const { open, setOpen } = useOpenRecord(['deficit'] as const)
const current = computed(() => (open.value ? (rows.value.find((r) => r.kind === open.value?.id) ?? null) : null))

const missingKey = (kind: string) => kind.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase())
/** Доля расследований для полосы (целые проценты). */
const share = (n: number) => (map.value?.investigations ? Math.round((n * 100) / map.value.investigations) : 0)
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="backendModeOf(q.data.value)"
    :state="state"
    :loading="q.isPending.value && !map"
    :error="map ? undefined : q.error.value"
    :data-widget="widgetId"
  >
    <div v-if="map" class="deficit" data-testid="deficit">
      <p class="principle ant-wrap">{{ t('widgets.deficit.principle') }}</p>
      <SectionPanel :title="t('widgets.deficit.title')" :subtitle="t('widgets.deficit.subtitle', { n: map.investigations })">
        <EmptyState v-if="!rows.length" compact :title="t('widgets.deficit.empty')" data-testid="no-deficit" />
        <DataTable v-else :caption="t('widgets.deficit.title')">
          <thead>
            <tr>
              <th scope="col">{{ t('widgets.deficit.col.what') }}</th>
              <th scope="col">{{ t('widgets.deficit.col.count') }}</th>
              <th scope="col">{{ t('widgets.deficit.col.where') }}</th>
              <th scope="col">{{ t('widgets.deficit.col.digitization') }}</th>
              <th scope="col">{{ t('widgets.deficit.col.estimate') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="r in rows"
              :key="r.kind"
              class="row"
              tabindex="0"
              :data-missing="r.kind"
              :aria-selected="open?.id === r.kind"
              @click="setOpen({ kind: 'deficit', id: r.kind })"
              @keydown.enter.prevent="setOpen({ kind: 'deficit', id: r.kind })"
            >
              <th scope="row" class="ant-box"><span class="ant-wrap">{{ t(`widgets.analysis.missing.${missingKey(r.kind)}`) }}</span></th>
              <td class="ant-box count">
                <span class="ant-wrap">{{ t('widgets.deficit.ofTotal', { n: r.investigations, total: map.investigations }) }}</span>
                <span class="bar" aria-hidden="true"><span class="fill" :style="{ width: `${share(r.investigations)}%` }" /></span>
              </td>
              <td class="ant-box"><span class="ant-clamp-2">{{ r.places.map((p) => `${p.place} (${p.count})`).join(', ') }}</span></td>
              <td class="ant-box"><span class="ant-clamp-2">{{ r.digitization }}</span></td>
              <td class="ant-box">
                <span v-if="r.estimate" class="ant-wrap" data-testid="estimate">{{ r.estimate.text }}</span>
                <span v-else class="absent ant-wrap" :title="t('widgets.deficit.noEstimateHint')">{{ t('widgets.deficit.noEstimate') }}</span>
              </td>
            </tr>
          </tbody>
        </DataTable>
      </SectionPanel>

      <SectionPanel variant="subtle" :title="t('widgets.deficit.gapsTitle')" :subtitle="t('hints.unableToAssess')">
        <ul v-if="gaps.length" class="gaps" data-testid="gaps">
          <li v-for="s in gaps" :key="s" :data-step="s"><code>{{ s }}</code> — {{ t('inspection.outcome.unableToAssess') }}</li>
        </ul>
        <p v-else class="absent" data-testid="no-gaps">{{ t('widgets.deficit.noGaps') }}</p>
      </SectionPanel>
    </div>
    <DeficitDrawer :row="current" :map="map" :density="density" @close="setOpen(null)" />
  </WidgetFrame>
</template>

<style scoped>
.deficit {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  font-size: var(--ant-fs-body);
}

.principle {
  margin: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border-left: 3px solid var(--ant-status-neutral);
  background: var(--ant-surface-subtle);
  font-size: var(--ant-fs-meta);
}

.row {
  cursor: pointer;
}

.row:hover {
  background: var(--ant-surface-hover);
}

.row[aria-selected='true'] {
  background: var(--ant-accent-soft);
}

.count {
  min-width: 120px;
}

.bar {
  display: block;
  height: 4px;
  margin-top: var(--ant-space-1);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-n-100);
}

.fill {
  display: block;
  height: 100%;
  border-radius: var(--ant-radius-pill);
  background: var(--ant-status-attention);
}

.absent {
  margin: 0;
  color: var(--ant-text-3);
}

.gaps {
  margin: 0;
  padding-left: var(--ant-space-5);
}
</style>
