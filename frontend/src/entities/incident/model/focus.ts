/**
 * Фокус разбора — состояние интерфейса стола технолога (Pinia, AD-21: серверных
 * данных здесь нет). Связывает виджеты без знания раскладки стола: группа
 * несоответствий в «Разборе причин» → общие факторы → гипотеза и область риска;
 * выбранная запись на дорожках подсвечивает связанные записи.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { CommonFactorRow } from './types'

/** Куда направлен общий фактор: в гипотезу или в сужение области. */
export type FactorIntent = 'hypothesis' | 'narrow_scope'

export const useAnalysisFocusStore = defineStore('analysis-focus', () => {
  /** Выбранная группа несоответствий. */
  const groupKey = ref<string | null>(null)
  /** Выбранное несоответствие. */
  const ncId = ref<string | null>(null)
  /** Выбранный инцидент (область риска). */
  const incidentId = ref<string | null>(null)
  /** Выбранная запись на дорожках. */
  const eventId = ref<string | null>(null)
  /** Общий фактор, из которого технолог вошёл в гипотезу или в сужение. */
  const factor = ref<{ row: CommonFactorRow; intent: FactorIntent } | null>(null)

  /** Выбрать группу — сбрасывает выбор внутри прежней группы. */
  function selectGroup(key: string | null): void {
    groupKey.value = key
    ncId.value = null
    eventId.value = null
    factor.value = null
  }

  /** Вход из строки общих факторов (FR-135). */
  function enterFromFactor(row: CommonFactorRow, intent: FactorIntent): void {
    factor.value = { row, intent }
  }

  return { groupKey, ncId, incidentId, eventId, factor, selectGroup, enterFromFactor }
})
