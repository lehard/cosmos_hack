<script setup lang="ts">
/**
 * Очередь «Ждут моего решения» (PRD §3a, FR-55, FR-57, UI-25) — задачи, а не
 * записи: строки сгруппированы по тому, что от контролёра нужно («подтвердить
 * или отклонить сигнал», «решить, что делать с изделием», «принять на точке
 * предъявления»), у группы счётчик и число просроченных. Порядок — по риску или
 * по сроку — считает сервер: группы идут в порядке первой своей строки, строки
 * внутри — как прислал сервер; «пересмотреть решение — пришли новые данные» —
 * всегда первой: изделие может уйти дальше по маршруту на прежнем решении.
 * В строке — суть (`reason`: вид дефекта и место), иначе заголовок сервера.
 * Сначала исключения: сверху сводка, пересмотры — крупными блоками, плановая
 * приёмка на точках предъявления свёрнута (раскрывается по кнопке или когда
 * открыта её строка). Щелчок или Enter по строке — открыть запись в
 * правом окне (Д-70). Работа с клавиатуры: ↑/↓ — соседняя строка сквозь группы.
 */
import { computed, ref } from 'vue'
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
const { t, d } = useI18n()

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
  const list = [...map.values()].map((g) => ({ ...g, overdue: g.items.filter((x) => x.overdue).length }))
  return [...list.filter((g) => g.kind === 'review'), ...list.filter((g) => g.kind !== 'review')]
})
/** Плановая приёмка — рутина: свёрнута, пока её не раскрыли или не открыли её строку. */
const routineOpen = ref(false)
const isOpen = (g: { kind: DecisionQueueRow['kind']; items: Item[] }) => g.kind !== 'presentation' || routineOpen.value || g.items.some((x) => x.key === props.selected)
/** Сводка по четырём видам работы: пересмотр, новые сигналы, решения по изделиям, плановая приёмка. */
const SUMMARY_ORDER: DecisionQueueRow['kind'][] = ['review', 'signal', 'isolated', 'presentation']
const summary = computed(() =>
  SUMMARY_ORDER.map((kind) => ({ kind, n: props.rows.filter((r) => r.kind === kind).length })).filter((x) => x.n > 0),
)
/** Строка задачи: у сигнала — «Возможный дефект: …» (это ещё не несоответствие), у остальных — суть или заголовок сервера. */
const rowTitle = (row: DecisionQueueRow) =>
  row.kind === 'signal' && row.reason ? t('widgets.decisionQueue.signalTitle', { what: row.reason.toLowerCase() }) : (row.reason ?? row.title)
/** Строки в порядке показа (только раскрытые группы) — для ↑/↓ сквозь группы. */
const flat = computed(() => groups.value.filter(isOpen).flatMap((g) => g.items))

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
    <p class="summary ant-wrap" data-testid="queue-summary">
      <span v-for="x in summary" :key="x.kind" :data-kind="x.kind">{{ t(`widgets.decisionQueue.summary.${x.kind}`, { n: x.n }, x.n) }}</span>
    </p>
    <div class="groups" tabindex="0" :aria-label="t('desks.decisionQueue')" data-testid="queue-groups" @keydown.down.prevent="move(1)" @keydown.up.prevent="move(-1)">
      <!-- Четыре вида работы — четыре блока со своим акцентом и одной строкой смысла (UI-50). -->
      <section v-for="g in groups" :key="g.kind" class="group" :data-group="g.kind">
        <h4 class="group-title">
          <span class="ant-wrap">{{ t(QUEUE_GROUP_TEXT[g.kind]) }}</span>
          <span class="count" data-testid="group-count">{{ g.items.length }}</span>
          <span v-if="g.overdue" class="overdue-count" data-testid="group-overdue">{{ t('widgets.decisionQueue.overdueCount', { n: g.overdue }) }}</span>
          <button
            v-if="g.kind === 'presentation'"
            type="button"
            class="toggle"
            data-testid="toggle-routine"
            :aria-expanded="isOpen(g)"
            @click="routineOpen = !isOpen(g)"
          >
            {{ isOpen(g) ? t('widgets.decisionQueue.hideRoutine') : t('widgets.decisionQueue.showRoutine') }}
          </button>
        </h4>
        <p class="group-sense ant-wrap" data-testid="group-sense">{{ t(`widgets.decisionQueue.sense.${g.kind}`) }}</p>
        <ol v-if="isOpen(g)" class="rows">
          <li
            v-for="{ row, key, overdue, due } in g.items"
            :key="key"
            class="row"
            :class="{ hero: row.kind === 'review' }"
            :data-key="key"
            :data-kind="row.kind"
            :data-severity="row.severity"
            :data-overdue="overdue || undefined"
            :aria-selected="key === selected"
            tabindex="-1"
            @click="emit('select', row)"
            @keydown.enter.prevent="emit('select', row)"
          >
            <p v-if="row.kind === 'review'" class="hero-kicker ant-wrap">{{ t('widgets.decisionQueue.heroKicker') }}</p>
            <div class="line head">
              <strong class="item">{{ row.item_label }}</strong>
              <span class="title ant-ellipsis" :title="rowTitle(row)" data-testid="row-title">{{ rowTitle(row) }}</span>
              <span v-if="due" class="due" data-testid="due">{{ due }}</span>
            </div>
            <div class="line meta">
              <span>{{ t('common.words.severity') }}: {{ codeText(SEVERITY_TEXT, row.severity, t) }}</span>
              <span v-if="row.presentation_no">{{ t('decisions.gate.presentationNumber', { n: row.presentation_no }) }}</span>
              <span v-if="row.review_since" data-testid="review-since">{{ t('widgets.decisionQueue.reviewSince', { time: d(new Date(row.review_since), 'dateTime') }) }}</span>
            </div>
            <div v-if="row.reason && row.kind === 'review'" class="line meta">
              <span class="ant-clamp-2" :title="row.title">{{ row.title }}</span>
            </div>
            <p v-if="row.kind === 'review'" class="hero-action">{{ t('widgets.decisionQueue.heroAction') }} →</p>
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
  padding: var(--ant-space-3) var(--ant-space-4);
  border: 1px solid var(--ant-border);
  border-left: 4px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-lg);
  background: var(--ant-surface);
}

/* Пересмотр — самое критичное исключение; сигнал — установить факт; решение — судьба изделия; приёмка — рутина. */
.group[data-group='review'] {
  border-left-color: var(--ant-status-attention);
  background: var(--ant-status-attention-soft);
}

.group[data-group='signal'] {
  border-left-color: var(--ant-accent);
}

.group[data-group='isolated'] {
  border-left-color: var(--ant-status-danger);
}

.group[data-group='presentation'] {
  border-left-color: var(--ant-border);
  background: var(--ant-surface-subtle);
}

.group-sense {
  margin: 0;
  color: var(--ant-text-2);
  font-size: var(--ant-fs-meta);
}

.summary {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-4);
  margin: 0;
  color: var(--ant-text-2);
}

.summary [data-kind='review'] {
  color: var(--ant-status-attention-text);
  font-weight: var(--ant-fw-bold);
}

.toggle {
  margin-left: auto;
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  text-transform: none;
  letter-spacing: 0;
  cursor: pointer;
}

/* Пересмотр — крупный блок: изменились данные после принятого решения. */
.row.hero {
  padding: var(--ant-space-3) var(--ant-space-4);
  border-color: var(--ant-status-attention);
  background: var(--ant-surface);
}

.row.hero .item {
  font-size: var(--ant-fs-title);
}

.hero-kicker {
  margin: 0 0 var(--ant-space-1);
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.hero-action {
  margin: var(--ant-space-1) 0 0;
  color: var(--ant-accent);
  font-weight: var(--ant-fw-bold);
}

.group-title {
  display: flex;
  gap: var(--ant-space-2);
  align-items: baseline;
  min-width: 0;
  margin: 0;
  color: var(--ant-text);
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
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
