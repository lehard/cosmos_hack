<script setup lang="ts">
/**
 * Общие факторы группы несоответствий «сколько из N» (FR-135, PRD §3a «Технолог»):
 * станок, инструмент, оснастка, программа, исполнитель, партия материала.
 * Фактор, общий для всей группы, — первым. Из строки — прямой вход в гипотезу
 * и в сужение области риска. Совпадение фактора — обстоятельство, а не причина.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { FACTOR_TEXT, isCommonToAll, sortFactors, type CommonFactorRow, type CommonFactorsModel } from '@/entities/incident'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'
import { ActionButton, DataTable } from '@/shared/ui'

const props = withDefaults(defineProps<{ model: CommonFactorsModel; density?: Density }>(), { density: 'compact' })
const emit = defineEmits<{
  /** Открыть как гипотезу. */
  'to-hypothesis': [row: CommonFactorRow]
  /** Сузить область риска по фактору. */
  'to-narrow-scope': [row: CommonFactorRow]
}>()

const { t } = useI18n()
const moment = useMomentStore()

const rows = computed(() => sortFactors(props.model.rows))
const n = computed(() => props.model.nc_count)
const share = (r: CommonFactorRow) => (n.value > 0 ? Math.min(100, (r.matches / n.value) * 100) : 0)
</script>

<template>
  <div class="factors" :class="`density-${density}`" data-testid="common-factors">
    <p class="group">
      <strong>{{ t('widgets.analysis.factors.group', { label: model.group_label }) }}</strong>
      · {{ t('plural.nonconformities', { n }, n) }}
    </p>
    <p class="muted" :title="t('hints.commonFactors')">{{ t('ncCard.commonFactors.subtitle') }}</p>

    <DataTable class="table">
      <thead>
        <tr>
          <th>{{ t('widgets.analysis.factors.factor') }}</th>
          <th>{{ t('widgets.analysis.factors.value') }}</th>
          <th class="share-col">{{ t('widgets.analysis.factors.share') }}</th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr v-for="r in rows" :key="r.factor" :data-factor="r.factor" :class="{ common: isCommonToAll(r, n) }">
          <td class="factor">{{ t(FACTOR_TEXT[r.factor]) }}</td>
          <td class="value">
            <template v-if="r.value === null">
              <span class="muted">{{ t('widgets.analysis.factors.valueUnknown') }}</span>
            </template>
            <template v-else-if="r.distinct_values > 1">
              <span :title="r.value">{{ t('widgets.analysis.factors.varies', { n: r.distinct_values }) }}</span>
            </template>
            <template v-else>{{ r.value }}</template>
            <span v-if="isCommonToAll(r, n)" class="badge">{{ t('widgets.analysis.factors.commonToAll') }}</span>
          </td>
          <td class="share-col">
            <div class="bar" role="img" :aria-label="t('ncCard.commonFactors.row', { factor: t(FACTOR_TEXT[r.factor]), k: r.matches, n })">
              <span class="fill" :style="{ width: `${share(r)}%` }" />
            </div>
            <span class="num">{{ t('common.words.outOf', { a: r.matches, b: n }) }}</span>
          </td>
          <td class="actions">
            <ActionButton overflow="wrap"
              :size="naiveSizeOf(density)"
              quaternary
              :disabled="moment.isReplay || r.value === null"
              data-testid="to-hypothesis"
              @click="emit('to-hypothesis', r)"
              :label="t('ncCard.commonFactors.toHypothesis')"
            />
            <ActionButton overflow="wrap"
              :size="naiveSizeOf(density)"
              quaternary
              :disabled="moment.isReplay || r.value === null"
              data-testid="to-narrow-scope"
              @click="emit('to-narrow-scope', r)"
              :label="t('ncCard.commonFactors.toNarrowScope')"
            />
          </td>
        </tr>
      </tbody>
    </DataTable>
  </div>
</template>

<style scoped>
.factors {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.group,
.muted {
  margin: 0;
}

.muted {
  color: var(--ant-text-3);
}

td {
  vertical-align: middle;
}

tr.common td {
  background: var(--ant-surface-subtle);
}

tr.common .factor {
  font-weight: var(--ant-fw-bold);
}

.badge {
  margin-left: 6px;
  padding: 0 6px;
  border-radius: var(--ant-radius-lg);
  background: var(--ant-accent-soft-border);
  color: var(--ant-accent-pressed);
  font-size: var(--ant-fs-xs);
  white-space: nowrap;
}

.share-col {
  white-space: nowrap;
}

.bar {
  display: inline-block;
  width: 64px;
  height: 6px;
  margin-right: 6px;
  border-radius: var(--ant-radius-sm);
  background: var(--ant-border);
  vertical-align: middle;
  overflow: hidden;
}

.fill {
  display: block;
  height: 100%;
  background: var(--ant-accent);
}

.num {
  font-family: var(--ant-font-mono);
}

.actions {
  text-align: right;
}
</style>
