<script setup lang="ts">
/**
 * Карточка раздела «Аналитика»: заголовок и показатели под ним.
 * - Показатель один — заголовок только у карточки (без второго заголовка с
 *   названием показателя); название показателя от сервера, если оно другое, и
 *   оговорки числа — в подсказке значка у заголовка; пояснение — мелко под числом.
 * - Показателей несколько — заголовок карточки и подписи показателей.
 * - Показателей нет — «не передано», не ноль (NFR-UI-4).
 * Без `title` заголовок карточки — название единственного показателя.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { MetricNotes, originOf, valueNotes, type MetricPick, type MetricRow } from '@/entities/metric'
import { SectionPanel } from '@/shared/ui'
import MetricRowView from './MetricRowView.vue'

const props = withDefaults(defineProps<{ rows: readonly MetricRow[]; picked: MetricPick | null; title?: string }>(), { title: undefined })
const emit = defineEmits<{ pick: [p: MetricPick] }>()
const { t } = useI18n()

const single = computed(() => (props.rows.length === 1 ? props.rows[0]! : null))
const heading = computed(() => props.title ?? single.value?.title ?? '')
/** Подсказка у заголовка карточки с одним показателем: его название (если скрыто) и оговорки числа. */
const notes = computed(() => {
  const row = single.value
  if (!row) return []
  return [row.title !== heading.value ? row.title : null, ...valueNotes(t, row.total)].filter((x): x is string => Boolean(x))
})
const warn = computed(() => Boolean(single.value && originOf(single.value.total)?.warn))
</script>

<template>
  <SectionPanel variant="card" class="metric-card" :data-card="single?.metric_id">
    <div class="card-head">
      <h4 class="card-title ant-clamp-2" data-testid="card-title" :title="heading">{{ heading }}</h4>
      <MetricNotes :notes="notes" :warn="warn" />
    </div>
    <MetricRowView v-for="r in rows" :key="r.metric_id" class="card-row" :row="r" :picked="picked" :show-title="!single" @pick="emit('pick', $event)" />
    <p v-if="!rows.length" class="absent ant-wrap" data-testid="absent">{{ t('widgets.analytics.notProvided') }}</p>
  </SectionPanel>
</template>

<style scoped>
.card-head {
  display: flex;
  gap: var(--ant-space-1);
  align-items: flex-start;
  min-width: 0;
}

.card-title {
  flex: 1 1 auto;
  margin: 0;
  color: var(--ant-text);
  font-size: var(--ant-fs-body);
  font-weight: var(--ant-fw-bold);
  line-height: var(--ant-lh-tight);
}

/* Несколько показателей в карточке — тонкий разделитель между ними. */
.card-row + .card-row {
  padding-top: var(--ant-space-3);
  border-top: 1px solid var(--ant-border);
}

.absent {
  margin: 0;
  color: var(--ant-n-400);
  font-size: var(--ant-fs-meta);
}
</style>
