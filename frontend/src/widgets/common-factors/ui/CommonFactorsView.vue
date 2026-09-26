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
const share = (r: CommonFactorRow) => (n.value > 0 ? Math.min(100, (r.matches / n.value) * 100) : 0)

type Kind = 'all' | 'some' | 'varies' | 'unknown'
const kindOf = (r: CommonFactorRow): Kind => {
  if (r.value === null) return 'unknown'
  if (isCommonToAll(r, n.value)) return 'all'
  // Разное и ни одно значение не повторилось — «не совпадает»; иначе — самое частое «у k из n».
  if (r.distinct_values > 1 && r.matches <= 1) return 'varies'
  return 'some'
}
/** Сколько совпало — словами. */
function howMany(r: CommonFactorRow): string {
  switch (kindOf(r)) {
    case 'all':
      return t('widgets.analysis.factors.inAll', { n: n.value })
    case 'varies':
      return t('widgets.analysis.factors.variesIn', { k: r.distinct_values, n: n.value })
    case 'unknown':
      return t('widgets.analysis.factors.noData')
    default:
      return t('widgets.analysis.factors.inSome', { k: r.matches, n: n.value })
  }
}
</script>

<template>
  <div class="factors" :class="`density-${density}`" data-testid="common-factors">
    <p class="lead ant-wrap" :title="t('hints.commonFactors')">{{ t('widgets.analysis.factors.lead', { n }) }}</p>
    <ul class="rows">
      <li v-for="r in rows" :key="r.factor" class="row" :data-factor="r.factor" :data-kind="kindOf(r)">
        <p class="what ant-wrap">
          <span class="name">{{ t(FACTOR_TEXT[r.factor]) }}</span><template v-if="r.value !== null && kindOf(r) !== 'varies'">: <span class="value">{{ r.value_label || r.value }}</span></template>
        </p>
        <p class="how">
          <span class="bar" aria-hidden="true"><span class="fill" :style="{ width: `${share(r)}%` }" /></span>
          <span class="how-text ant-wrap" data-testid="how-many">{{ howMany(r) }}</span>
        </p>
      </li>
    </ul>
    <p class="muted ant-wrap">{{ t('widgets.analysis.factors.notACause') }}</p>
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
  color: var(--ant-text-2);
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.rows {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.row {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding: var(--ant-space-2) 0;
  border-top: 1px solid var(--ant-border);
}

.row:first-child {
  border-top: 0;
}

.row[data-kind='all'] {
  margin: 0 calc(-1 * var(--ant-space-2));
  padding: var(--ant-space-2);
  border-top: 0;
  border-left: 4px solid var(--ant-accent);
  border-radius: var(--ant-radius-md);
  background: var(--ant-accent-soft);
}

.name {
  font-weight: var(--ant-fw-bold);
}

/* Обозначение (ИС-2, П-88) не рвётся посередине. */
.value {
  white-space: nowrap;
}

.row[data-kind='unknown'] .what,
.row[data-kind='unknown'] .how-text {
  color: var(--ant-text-3);
}

.how {
  display: flex;
  gap: var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.bar {
  flex: none;
  width: 64px;
  height: 6px;
  overflow: hidden;
  border-radius: var(--ant-radius-pill);
  background: var(--ant-border);
}

.fill {
  display: block;
  height: 100%;
  background: var(--ant-accent);
}

.row[data-kind='all'] .how-text {
  font-weight: var(--ant-fw-bold);
}
</style>
