/**
 * Выбор на столе контролёра — состояние интерфейса (Pinia, AD-21: серверных
 * данных здесь нет). Связывает виджеты без знания раскладки стола: строка
 * очереди «Ждут моего решения» → карточка несоответствия, панель решений и
 * компактный паспорт того же изделия.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DecisionQueueRow } from './types'

/** Ключ строки очереди: вид и объект. */
export const rowKey = (row: Pick<DecisionQueueRow, 'kind' | 'object_id'>): string => `${row.kind}:${row.object_id}`

export const useDecisionFocusStore = defineStore('decision-focus', () => {
  /** Выбранная строка очереди. */
  const rowId = ref<string | null>(null)
  /** Несоответствие в карточке. */
  const ncId = ref<string | null>(null)
  /** Изделие в паспорте. */
  const itemId = ref<string | null>(null)

  /** Выбрать строку очереди. */
  function select(row: Pick<DecisionQueueRow, 'kind' | 'object_id' | 'item_id' | 'nc_id'>): void {
    rowId.value = rowKey(row)
    ncId.value = row.nc_id ?? null
    itemId.value = row.item_id
  }

  /** Сбросить выбор. */
  function clear(): void {
    rowId.value = null
    ncId.value = null
    itemId.value = null
  }

  return { rowId, ncId, itemId, select, clear }
})
