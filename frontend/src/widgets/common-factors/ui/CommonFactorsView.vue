<script setup lang="ts">
/**
 * Что общего у несоответствий группы (FR-135, PRD §3a «Технолог»; UI-33): станок,
 * инструмент, оснастка, программа, исполнитель, партия материала — фразами, а
 * не таблицей: «Станок: ИС-2 — у всех 3», «у 2 из 3», «разное у 3», «неизвестно».
 * Общий для всех — первым и выделен. Совпадение фактора — обстоятельство, а не
 * причина: проверяется гипотезой. Узкая колонка — без горизонтальной прокрутки.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { FACTOR_TEXT, isCommonToAll, sortFactors, type CommonFactorRow, type CommonFactorsModel } from '@/entities/incident'
import type { Density } from '@/shared/config/widget'

const props = withDefaults(defineProps<{ model: CommonFactorsModel; density?: Density }>(), { density: 'compact' })

const { t } = useI18n()

const rows = computed(() => sortFactors(props.model.rows))
const n = computed(() => props.model.nc_count)

type Kind = 'all' | 'some' | 'varies' | 'unknown'
const kindOf = (r: CommonFactorRow): Kind => {
  if (r.value === null) return 'unknown'
  if (isCommonToAll(r, n.value)) return 'all'
  // Разное и ни одно значение не повторилось — «не совпадает»; иначе — самое частое «у k из n».
  if (r.distinct_values > 1 && r.matches <= 1) return 'varies'
  return 'some'
}
/** Совпавшие у двух и более — фишками; не совпавшие и без данных — одной строкой. */
const shared = computed(() => rows.value.filter((r) => kindOf(r) === 'all' || kindOf(r) === 'some'))
const differs = computed(() => rows.value.filter((r) => kindOf(r) === 'varies').map((r) => t(FACTOR_TEXT[r.factor]).toLowerCase()))
const unknownList = computed(() => rows.value.filter((r) => kindOf(r) === 'unknown').map((r) => t(FACTOR_TEXT[r.factor]).toLowerCase()))

</script>

<template>
  <div class="factors" :class="`density-${density}`" data-testid="common-factors">
    <p class="lead ant-wrap" :title="t('hints.commonFactors')">{{ t('widgets.analysis.factors.together') }} · {{ t('widgets.analysis.factors.lead', { n }) }}</p>
    <ul v-if="shared.length" class="chips">
      <li v-for="r in shared" :key="r.factor" class="chip" :data-factor="r.factor" :data-kind="kindOf(r)">
        <span class="ant-wrap">{{ t(FACTOR_TEXT[r.factor]) }}: <strong class="value">{{ r.value_label || r.value }}</strong></span>
        <span class="share" data-testid="how-many">{{ r.matches }}/{{ n }}</span>
      </li>
    </ul>
    <p v-if="differs.length" class="muted ant-wrap" data-testid="differs">{{ t('widgets.analysis.factors.differs', { what: differs.join(', ') }) }}</p>
    <p v-if="unknownList.length" class="muted ant-wrap" data-testid="unknown">{{ t('widgets.analysis.factors.unknownList', { what: unknownList.join(', ') }) }}</p>
    <p class="note ant-wrap">{{ t('widgets.analysis.factors.notACause') }}</p>
  </div>
</template>

<style scoped>
.factors {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

p {
  margin: 0;
}

.lead {
  font-weight: var(--ant-fw-bold);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.chip {
  display: inline-flex;
  gap: var(--ant-space-2);
  align-items: baseline;
  min-width: 0;
  padding: 2px var(--ant-space-2);
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-surface-subtle);
}

.chip[data-kind='all'] {
  border-color: var(--ant-accent);
}

.value {
  white-space: nowrap;
}

.share {
  color: var(--ant-text-2);
  font-variant-numeric: tabular-nums;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.note {
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
}
</style>
