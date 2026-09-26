/**
 * Выбор на пульте тестовых сценариев — состояние интерфейса (Pinia, AD-21:
 * серверных данных здесь нет). Связывает виджеты стола без знания раскладки:
 * пульт выбирает сценарий и прогон, табло «ожидалось → получилось» и столы
 * показывают тот же прогон.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useRunFocusStore = defineStore('run-focus', () => {
  /** Выбранный прогон; null — текущий по правилу pickCurrentRun. */
  const runId = ref<string | null>(null)
  /** Сценарий, выбранный для запуска. */
  const scenarioId = ref<string | null>(null)

  /** Выбрать прогон. */
  function selectRun(id: string | null): void {
    runId.value = id
  }

  /** Выбрать сценарий для запуска. */
  function selectScenario(id: string | null): void {
    scenarioId.value = id
  }

  return { runId, scenarioId, selectRun, selectScenario }
})
