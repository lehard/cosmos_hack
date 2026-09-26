/**
 * Фокус аналитики — состояние интерфейса (Pinia, AD-21: серверных данных здесь
 * нет). Связывает виджеты без знания раскладки стола: период общий для плиток,
 * раздела «Аналитика», раскрытия и контрольной карты; число, выбранное в
 * плитке или в разделе, раскрывает виджет «Раскрытие показателя» (FR-7).
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { PeriodKind } from '@/shared/api/generated/model'

/** Периоды, которые можно выбрать на экране (произвольный — во втором слое). */
export const PERIODS = ['shift', 'day', 'week', 'month'] as const satisfies readonly PeriodKind[]
export type SelectablePeriod = (typeof PERIODS)[number]

/** Выбранное число: показатель и, если выбран срез, его ключ. */
export interface MetricPick {
  metricId: string
  /** Название показателя (от сервера) — для заголовка раскрытия. */
  title: string
  /** Ключ среза; нет — итог показателя. */
  sliceKey?: string
  /** Подпись среза. */
  sliceLabel?: string
}

export const useMetricFocusStore = defineStore('metric-focus', () => {
  /** Период показателей (по умолчанию — смена, как у операций). */
  const period = ref<SelectablePeriod>('shift')
  /** Выбранное число. */
  const pick = ref<MetricPick | null>(null)

  /** Раскрыть число. */
  function select(p: MetricPick): void {
    pick.value = { ...p }
  }

  /** Снять выбор. */
  function clear(): void {
    pick.value = null
  }

  return { period, pick, select, clear }
})
