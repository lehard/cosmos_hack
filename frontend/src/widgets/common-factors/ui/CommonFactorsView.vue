<script setup lang="ts">
/**
 * Общие факторы группы несоответствий «сколько из N» (FR-135, PRD §3a «Технолог»):
 * станок, инструмент, оснастка, программа, исполнитель, партия материала.
 * Фактор, общий для всей группы, — первым. Из строки — прямой вход в гипотезу
 * и в сужение области риска. Совпадение фактора — обстоятельство, а не причина.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { FACTOR_TEXT, isCommonToAll, sortFactors, type CommonFactorRow, type CommonFactorsModel } from '@/entities/incident'
import { naiveSizeOf, type Density } from '@/shared/config/widget'
import { useMomentStore } from '@/shared/model/moment'

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

    <table class="table">
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
            <NButton
              :size="naiveSizeOf(density)"
              quaternary
              :disabled="moment.isReplay || r.value === null"
              data-testid="to-hypothesis"
              @click="emit('to-hypothesis', r)"
            >
              {{ t('ncCard.commonFactors.toHypothesis') }}
            </NButton>
            <NButton
              :size="naiveSizeOf(density)"
              quaternary
              :disabled="moment.isReplay || r.value === null"
              data-testid="to-narrow-scope"
              @click="emit('to-narrow-scope', r)"
            >
              {{ t('ncCard.commonFactors.toNarrowScope') }}
            </NButton>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.factors {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
}

.density-large {
  font-size: 16px;
}

.group,
.muted {
  margin: 0;
}

.muted {
  color: #6b7280;
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table th {
  padding: 4px 6px;
  color: #6b7280;
  font-weight: 400;
  text-align: left;
  border-bottom: 1px solid #e5e7eb;
}

.table td {
  padding: 4px 6px;
  border-bottom: 1px solid #f3f4f6;
  vertical-align: middle;
}

tr.common td {
  background: #f8fafc;
}

tr.common .factor {
  font-weight: 700;
}

.badge {
  margin-left: 6px;
  padding: 0 6px;
  border-radius: 8px;
  background: #e0e7ff;
  color: #1e3a8a;
  font-size: 11px;
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
  border-radius: 3px;
  background: #e5e7eb;
  vertical-align: middle;
  overflow: hidden;
}

.fill {
  display: block;
  height: 100%;
  background: #2f6fdb;
}

.num {
  font-family: 'PT Mono', monospace;
}

.actions {
  white-space: nowrap;
  text-align: right;
}
</style>
