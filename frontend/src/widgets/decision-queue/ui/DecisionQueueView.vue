<script setup lang="ts">
/**
 * Очередь «Ждут моего решения» (PRD §3a, FR-55, FR-57): точки предъявления,
 * сигналы на рассмотрение, изолированные изделия со сроком решения (обратный
 * отсчёт). Порядок — по риску или по сроку — считает сервер; здесь
 * переключатель и показ. Работа с клавиатуры: ↑/↓ — соседняя строка.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { QUEUE_KIND_TEXT, SEVERITY_TEXT, codeText, deadlineOf, rowKey, type DecisionQueueRow, type QueueSort } from '@/entities/nonconformity'
import type { Density } from '@/shared/config/widget'
import { formatMinutes } from '@/shared/lib/duration'

const props = withDefaults(
  defineProps<{
    rows: DecisionQueueRow[]
    /** Ключ выбранной строки (`вид:объект`). */
    selected?: string | null
    sort: QueueSort
    /** «Сейчас» для обратного отсчёта, мс. */
    now: number
    density?: Density
  }>(),
  { selected: null, density: 'comfortable' },
)
const emit = defineEmits<{
  select: [row: DecisionQueueRow]
  'update:sort': [sort: QueueSort]
}>()
const { t } = useI18n()

const items = computed(() =>
  props.rows.map((row) => {
    const dl = deadlineOf(row.due_at, props.now)
    const overdue = row.overdue || !!dl?.overdue
    const due = dl ? (dl.overdue ? t('ncCard.deadline.overdueBy', { time: formatMinutes(t, dl.minutes) }) : t('ncCard.deadline.countdown', { time: formatMinutes(t, dl.minutes) })) : null
    return { row, key: rowKey(row), overdue, due }
  }),
)

function move(delta: number): void {
  if (!items.value.length) return
  const i = items.value.findIndex((x) => x.key === props.selected)
  const next = items.value[Math.min(items.value.length - 1, Math.max(0, i < 0 ? 0 : i + delta))]
  if (next) emit('select', next.row)
}
</script>

<template>
  <div class="queue" :class="`density-${density}`" data-testid="decision-queue">
    <NRadioGroup :value="sort" size="small" @update:value="(v: QueueSort) => emit('update:sort', v)">
      <NRadioButton value="risk" data-testid="sort-risk">{{ t('widgets.decisionQueue.sort.risk') }}</NRadioButton>
      <NRadioButton value="deadline" data-testid="sort-deadline">{{ t('widgets.decisionQueue.sort.deadline') }}</NRadioButton>
    </NRadioGroup>
    <ol class="rows" tabindex="0" :aria-label="t('desks.decisionQueue')" @keydown.down.prevent="move(1)" @keydown.up.prevent="move(-1)">
      <li
        v-for="{ row, key, overdue, due } in items"
        :key="key"
        class="row"
        :data-key="key"
        :data-kind="row.kind"
        :data-severity="row.severity"
        :data-overdue="overdue || undefined"
        :aria-selected="key === selected"
        @click="emit('select', row)"
      >
        <div class="line">
          <span class="kind">{{ codeText(QUEUE_KIND_TEXT, row.kind, t) }}</span>
          <strong>{{ row.item_label }}</strong>
          <span v-if="row.presentation_no" class="muted">{{ t('decisions.gate.presentationNumber', { n: row.presentation_no }) }}</span>
        </div>
        <div class="line">
          <span>{{ row.title }}</span>
        </div>
        <div class="line meta">
          <span>{{ t('common.words.severity') }}: {{ codeText(SEVERITY_TEXT, row.severity, t) }}</span>
          <span v-if="due" class="due" data-testid="due">{{ due }}</span>
        </div>
      </li>
    </ol>
  </div>
</template>

<style scoped>
.queue {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: var(--ant-fs-body);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.rows {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
  outline: none;
}

.row {
  padding: 6px 8px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  cursor: pointer;
}

.row[aria-selected='true'] {
  border-color: var(--ant-accent);
  background: var(--ant-accent-soft);
}

.row[data-severity='critical'] {
  border-left: 3px solid var(--ant-status-critical);
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 8px;
  align-items: baseline;
}

.kind,
.muted,
.meta {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.row[data-overdue] .due {
  color: var(--ant-status-danger);
  font-weight: var(--ant-fw-bold);
}
</style>
