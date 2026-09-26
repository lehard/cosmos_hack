/**
 * Выбор на столе контролёра — состояние интерфейса (Pinia, AD-21: серверных
 * данных здесь нет). Связывает виджеты без знания раскладки стола: строка
 * очереди «Ждут моего решения» → карточка несоответствия, панель решений и
 * компактный паспорт того же изделия.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { QueueEntry } from './types'

export const useDecisionFocusStore = defineStore('decision-focus', () => {
  /** Выбранная строка очереди. */
  const entryId = ref<string | null>(null)
  /** Несоответствие в карточке. */
  const ncId = ref<string | null>(null)
  /** Изделие в паспорте. */
  const itemId = ref<string | null>(null)

  /** Выбрать строку очереди. */
  function select(entry: Pick<QueueEntry, 'entry_id' | 'item_id' | 'nc_id'>): void {
    entryId.value = entry.entry_id
    ncId.value = entry.nc_id ?? null
    itemId.value = entry.item_id
  }

  /** Сбросить выбор. */
  function clear(): void {
    entryId.value = null
    ncId.value = null
    itemId.value = null
  }

  return { entryId, ncId, itemId, select, clear }
})
