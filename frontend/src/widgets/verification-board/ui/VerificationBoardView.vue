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
import { NRadioButton, NRadioGroup } from 'naive-ui'
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
    <p class="subtitle">{{ t('testStand.board.subtitle') }}</p>
    <div class="summary" data-testid="board-summary">
      <strong v-if="summary.allMatched" class="all" data-testid="all-matched">{{ t('testStand.board.allMatched') }}</strong>
      <template v-else>
        <strong>{{ t('widgets.board.passedOf', { passed: summary.passed, total: summary.total }) }}</strong>
        <span v-if="summary.failed" class="failed" data-testid="mismatches">{{ t('testStand.board.mismatches', { n: summary.failed }) }}</span>
        <span v-if="summary.pending" class="muted">{{ t('widgets.board.pending', { n: summary.pending }) }}</span>
        <span v-if="summary.notReached" class="muted">{{ t('widgets.board.notReached', { n: summary.notReached }) }}</span>
      </template>
    </div>
    <NRadioGroup :value="filter" size="small" @update:value="(v: BoardFilter) => emit('update:filter', v)">
      <NRadioButton value="all" data-testid="filter-all">{{ t('widgets.board.filter.all') }}</NRadioButton>
      <NRadioButton value="failed" data-testid="filter-failed">{{ t('widgets.board.filter.failed') }}</NRadioButton>
      <NRadioButton value="open" data-testid="filter-open">{{ t('widgets.board.filter.open') }}</NRadioButton>
    </NRadioGroup>
    <ol class="rows">
      <li v-for="{ row, expected, actual, color } in rows" :key="row.assertion_id" class="row" :data-id="row.assertion_id" :data-status="row.status">
        <div class="line">
          <span class="status">
            <span class="dot" :style="{ background: color }" aria-hidden="true" />
            {{ t(`widgets.board.status.${codeToKey(row.status)}`) }}
          </span>
          <span class="muted">{{ row.assertion_id }} · {{ t('widgets.board.step', { step: displayStep(row.step) }) }}</span>
        </div>
        <div class="title">{{ row.title }}</div>
        <div class="values">
          <span class="muted">{{ t('testStand.board.expected') }}</span>
          <code data-testid="expected">{{ expected }}</code>
          <span aria-hidden="true">→</span>
          <span class="muted">{{ t('testStand.board.actual') }}</span>
          <code v-if="actual !== null" data-testid="actual">{{ actual }}</code>
          <span v-else class="muted" data-testid="actual">{{ t('widgets.board.notChecked') }}</span>
        </div>
        <div class="muted check">{{ t('widgets.board.checkedBy', { operation: row.operation_id, path: row.path }) }}</div>
      </li>
    </ol>
  </div>
</template>

<style scoped>
.board {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}

.density-large {
  font-size: 16px;
}

.subtitle {
  margin: 0;
  color: #6b7280;
}

.summary {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 12px;
  align-items: baseline;
}

.all {
  color: #2e9e5b;
}

.failed {
  color: #d64545;
  font-weight: 700;
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
  padding: 6px 8px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
}

.row[data-status='failed'] {
  border-left: 3px solid #d64545;
}

.row[data-status='not_reached'] {
  opacity: 0.7;
}

.line,
.values {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 8px;
  align-items: baseline;
}

.status {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  font-weight: 700;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

code {
  font-family: 'PT Mono', monospace;
  font-size: 12px;
}

.muted {
  color: #6b7280;
  font-size: 12px;
}

.check {
  word-break: break-all;
}
</style>
