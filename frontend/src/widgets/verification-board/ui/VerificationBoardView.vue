<script setup lang="ts">
/**
 * Табло «ожидалось → получилось» — представление (AD-26, кейс §5.1, FR-129):
 * строки утверждений сценария (ожидаемое хранится отдельно от входных событий)
 * со значением, полученным теми же операциями API, и состоянием: совпало,
 * не совпало, ждёт проверки, сценарий не дошёл до шага. Сверху — сводка;
 * фильтр оставляет несовпадения.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BOARD_STATUS_TONE, boardRows, boardSummary, boardValue, displayStep, type Board, type BoardFilter } from '@/entities/run'
import { statusPalette } from '@/shared/api/generated/statuses'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'

const props = withDefaults(defineProps<{ board: Board; filter: BoardFilter; density?: Density }>(), { density: 'comfortable' })
const emit = defineEmits<{ 'update:filter': [filter: BoardFilter] }>()
const { t } = useI18n()

const summary = computed(() => boardSummary(props.board))
const rows = computed(() =>
  boardRows(props.board.rows, props.filter).map((row) => ({
    row,
    expected: boardValue(row.expected),
    actual: boardValue(row.actual),
    color: statusPalette[BOARD_STATUS_TONE[row.status]],
  })),
)
</script>

<template>
  <div class="board" :class="`density-${density}`" data-testid="verification-board">
    <p class="head" data-testid="board-summary">
      <strong v-if="summary.allMatched" class="all" data-testid="all-matched">{{ t('testStand.board.allMatched') }}</strong>
      <template v-else>
        {{ t('widgets.board.passedOf', { passed: summary.passed, total: summary.total }) }}
        <span v-if="summary.failed" class="failed" data-testid="mismatches"> · {{ t('testStand.board.mismatches', { n: summary.failed }) }}</span>
      </template>
    </p>
    <p class="quiet">{{ t('testStand.board.subtitle') }}</p>
    <p class="tabs">
      <button v-for="f in (['all', 'failed', 'open'] as const)" :key="f" type="button" class="tab" :data-on="filter === f || undefined" :data-testid="`filter-${f}`" @click="emit('update:filter', f)">
        {{ t(`widgets.board.filter.${f}`) }}
      </button>
    </p>
    <ol class="rows">
      <li v-for="{ row, expected, actual, color } in rows" :key="row.assertion_id" class="row" :data-id="row.assertion_id" :data-status="row.status" :title="t('widgets.board.checkedBy', { operation: row.operation_id, path: row.path })">
        <span class="dot" :style="{ background: color }" :aria-label="t(`widgets.board.status.${codeToKey(row.status)}`)" />
        <span class="title">{{ row.title }}</span>
        <span v-if="row.status === 'failed'" class="values">
          {{ t('testStand.board.expected') }} <code data-testid="expected">{{ expected }}</code> → {{ t('testStand.board.actual') }}
          <code v-if="actual !== null" data-testid="actual">{{ actual }}</code><template v-else>{{ t('widgets.board.notChecked') }}</template>
        </span>
      </li>
    </ol>
  </div>
</template>

<style scoped>
.board {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
  font-size: var(--ant-fs-body);
}

p {
  margin: 0;
}

.head {
  font-size: var(--ant-fs-title);
}

.all {
  color: var(--ant-status-success-text);
}

.failed {
  color: var(--ant-status-danger-text);
}

.quiet {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.tabs {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-3);
}

.tab {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-text-2);
  font: inherit;
  cursor: pointer;
}

.tab[data-on] {
  color: var(--ant-text);
  font-weight: var(--ant-fw-bold);
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
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: baseline;
  padding: var(--ant-space-2) 0;
  border-bottom: 1px solid var(--ant-border);
}

.row:last-child {
  border-bottom: 0;
}

.row[data-status='not_reached'],
.row[data-status='pending'] {
  color: var(--ant-text-3);
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  transform: translateY(-1px);
}

.title {
  flex: 1 1 0;
  min-width: 0;
}

.values {
  flex-basis: 100%;
  padding-left: calc(8px + var(--ant-space-2));
  color: var(--ant-status-danger-text);
  font-size: var(--ant-fs-meta);
}

code {
  font-family: var(--ant-font-mono);
}
</style>
