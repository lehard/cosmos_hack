<script setup lang="ts">
/**
 * Очередь «Ждут моего решения» (PRD §3a, FR-55, FR-57, UI-25) — задачи, а не
 * записи: строки сгруппированы по тому, что от контролёра нужно («подтвердить
 * или отклонить сигнал», «решить, что делать с изделием», «принять на точке
 * предъявления»), у группы счётчик и число просроченных. Порядок — по риску или
 * по сроку — считает сервер: группы идут в порядке первой своей строки, строки
 * внутри — как прислал сервер. Щелчок или Enter по строке — открыть запись в
 * правом окне (Д-70). Работа с клавиатуры: ↑/↓ — соседняя строка сквозь группы.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { QUEUE_GROUP_TEXT, SEVERITY_TEXT, codeText, deadlineOf, rowKey, type DecisionQueueRow, type QueueSort } from '@/entities/nonconformity'
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

type Item = (typeof items.value)[number]
/** Группы по виду задачи в порядке первой строки группы (порядок сервера сохраняется). */
const groups = computed(() => {
  const map = new Map<string, { kind: DecisionQueueRow['kind']; items: Item[] }>()
  for (const it of items.value) {
    const g = map.get(it.row.kind) ?? { kind: it.row.kind, items: [] }
    g.items.push(it)
    map.set(it.row.kind, g)
  }
  return [...map.values()].map((g) => ({ ...g, overdue: g.items.filter((x) => x.overdue).length }))
})
/** Строки в порядке показа — для ↑/↓ сквозь группы. */
const flat = computed(() => groups.value.flatMap((g) => g.items))

function move(delta: number): void {
  if (!flat.value.length) return
  const i = flat.value.findIndex((x) => x.key === props.selected)
  const next = flat.value[Math.min(flat.value.length - 1, Math.max(0, i < 0 ? 0 : i + delta))]
  if (next) emit('select', next.row)
}
</script>

<template>
  <div class="queue" :class="`density-${density}`" data-testid="decision-queue">
    <div class="bar">
      <NRadioGroup :value="sort" size="small" @update:value="(v: QueueSort) => emit('update:sort', v)">
        <NRadioButton value="risk" data-testid="sort-risk">{{ t('widgets.decisionQueue.sort.risk') }}</NRadioButton>
        <NRadioButton value="deadline" data-testid="sort-deadline">{{ t('widgets.decisionQueue.sort.deadline') }}</NRadioButton>
      </NRadioGroup>
      <span class="hint" data-testid="open-hint">{{ t('widgets.decisionQueue.openHint') }}</span>
    </div>
    <div class="groups" tabindex="0" :aria-label="t('desks.decisionQueue')" data-testid="queue-groups" @keydown.down.prevent="move(1)" @keydown.up.prevent="move(-1)">
      <section v-for="g in groups" :key="g.kind" class="group" :data-group="g.kind">
        <h4 class="group-title">
          <span class="ant-ellipsis">{{ t(QUEUE_GROUP_TEXT[g.kind]) }}</span>
          <span class="count" data-testid="group-count">{{ g.items.length }}</span>
          <span v-if="g.overdue" class="overdue-count" data-testid="group-overdue">{{ t('widgets.decisionQueue.overdueCount', { n: g.overdue }) }}</span>
        </h4>
        <ol class="rows">
          <li
            v-for="{ row, key, overdue, due } in g.items"
            :key="key"
            class="row"
            :data-key="key"
            :data-kind="row.kind"
            :data-severity="row.severity"
            :data-overdue="overdue || undefined"
            :aria-selected="key === selected"
            tabindex="-1"
            @click="emit('select', row)"
            @keydown.enter.prevent="emit('select', row)"
          >
            <div class="line head">
              <strong class="item">{{ row.item_label }}</strong>
              <span class="title ant-ellipsis" :title="row.title">{{ row.title }}</span>
              <span v-if="due" class="due" data-testid="due">{{ due }}</span>
            </div>
            <div class="line meta">
              <span>{{ t('common.words.severity') }}: {{ codeText(SEVERITY_TEXT, row.severity, t) }}</span>
              <span v-if="row.presentation_no">{{ t('decisions.gate.presentationNumber', { n: row.presentation_no }) }}</span>
            </div>
          </li>
        </ol>
      </section>
    </div>
  </div>
</template>

<style scoped>
.queue {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: var(--ant-fs-body);
}

.bar {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 16px;
  align-items: center;
}

.hint {
  min-width: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.density-large {
  font-size: var(--ant-fs-lg);
}

.groups {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-4);
  outline: none;
}

.group {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
  min-width: 0;
}

.group-title {
  display: flex;
  gap: var(--ant-space-2);
  align-items: baseline;
  min-width: 0;
  margin: 0;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.count {
  padding: 0 var(--ant-space-2);
  border-radius: var(--ant-radius-pill);
  background: var(--ant-surface-subtle);
  color: var(--ant-text);
}

.overdue-count {
  color: var(--ant-status-danger);
  text-transform: none;
  letter-spacing: 0;
}

.rows {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.row {
  padding: 8px 12px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
  cursor: pointer;
}

.row:hover {
  border-color: var(--ant-border-strong);
  background: var(--ant-surface-hover);
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
  min-width: 0;
}

/* Изделие и срок — по краям, причина между ними сжимается с многоточием. */
.head {
  flex-wrap: nowrap;
}

.item {
  flex: none;
}

.title {
  flex: 1 1 auto;
  min-width: 0;
}

.due {
  flex: none;
  margin-left: auto;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.meta {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.row[data-overdue] .due {
  color: var(--ant-status-danger);
  font-weight: var(--ant-fw-bold);
}
</style>
