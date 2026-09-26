<script setup lang="ts" generic="Row extends object">
/**
 * Таблица данных в едином стиле: приглушённая шапка, тонкие разделители,
 * подсветка строки, числа — вправо, коды — моноширинно; длинный текст в
 * ячейке переносится (или многоточие с подсказкой — `ellipsis` у столбца);
 * широкая таблица прокручивается внутри своей рамки, а не ломает раскладку.
 *
 * Два способа:
 * - `columns` + `rows` (+ слоты `cell-‹key›`) — простая таблица;
 * - своя разметка `<thead>/<tbody>` в слоте по умолчанию — когда у строк свои
 *   атрибуты и события; стиль тот же.
 *
 * Когда брать: любой табличный список. Список карточек (очередь, лента) —
 * не таблица.
 */
import type { DataColumn } from './types'

const props = withDefaults(
  defineProps<{
    columns?: DataColumn[]
    rows?: readonly Row[]
    /** Ключ строки; без него — индекс. */
    rowKey?: (row: Row, index: number) => string | number
    /** Текст «нет строк» (только для columns + rows). */
    emptyText?: string
    /** Подпись таблицы для программ чтения экрана. */
    caption?: string
  }>(),
  { columns: undefined, rows: () => [], rowKey: undefined, emptyText: undefined, caption: undefined },
)

const cellText = (row: Row, key: string): string => {
  const v = (row as Record<string, unknown>)[key]
  return v === null || v === undefined || v === '' ? '—' : String(v)
}
const keyOf = (row: Row, i: number) => (props.rowKey ? props.rowKey(row, i) : i)
</script>

<template>
  <div class="data-table">
    <table class="ant-box">
      <caption v-if="caption" class="ant-sr-only ant-wrap">{{ caption }}</caption>
      <template v-if="columns">
        <thead>
          <tr>
            <th v-for="c in columns" :key="c.key" :style="{ width: c.width, textAlign: c.align }" scope="col">
              <span class="ant-ellipsis" :title="c.title">{{ c.title }}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in rows" :key="keyOf(row, i)">
            <td v-for="c in columns" :key="c.key" class="ant-box" :class="{ 'ant-mono': c.mono, 'is-num': c.align === 'right' }" :style="{ textAlign: c.align }">
              <slot :name="`cell-${c.key}`" :row="row" :value="(row as Record<string, unknown>)[c.key]">
                <span v-if="c.ellipsis" class="ant-ellipsis" :title="cellText(row, c.key)">{{ cellText(row, c.key) }}</span>
                <span v-else class="ant-wrap">{{ cellText(row, c.key) }}</span>
              </slot>
            </td>
          </tr>
        </tbody>
      </template>
      <slot v-else />
    </table>
    <p v-if="columns && !rows.length && emptyText" class="empty ant-wrap">{{ emptyText }}</p>
  </div>
</template>

<style scoped>
.data-table {
  width: 100%;
  min-width: 0;
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  font-size: var(--ant-fs-body);
  line-height: var(--ant-lh-tight);
}

.data-table :deep(th) {
  padding: var(--ant-pad-cell-y) var(--ant-pad-cell-x);
  border-bottom: 1px solid var(--ant-border-strong);
  background: var(--ant-surface-subtle);
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-bold);
  text-align: left;
  vertical-align: bottom;
  white-space: nowrap;
}

.data-table :deep(td) {
  padding: var(--ant-pad-cell-y) var(--ant-pad-cell-x);
  border-bottom: 1px solid var(--ant-border);
  vertical-align: top;
  overflow-wrap: break-word;
}

.data-table :deep(tbody tr:last-child > td) {
  border-bottom: 0;
}

.data-table :deep(tbody tr:hover > td) {
  background: var(--ant-surface-subtle);
}

.data-table :deep(tbody tr[aria-selected='true'] > td) {
  background: var(--ant-accent-soft);
}

.is-num {
  font-variant-numeric: tabular-nums;
}

.empty {
  padding: var(--ant-space-4);
  color: var(--ant-text-3);
  text-align: center;
}
</style>
