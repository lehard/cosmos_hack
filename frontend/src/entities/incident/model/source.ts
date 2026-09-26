/**
 * Источник данных виджета разбора: то, что контейнер виджета передаёт рамке
 * (загрузка, ошибка, режим fixtures | live) и представлению (данные).
 *
 * Пока операций чтения analysis нет в contracts/openapi.yaml, источник пуст:
 * виджет честно показывает «записей нет», а не выдуманные данные — заготовок в
 * коде интерфейса нет (PRD §11.10, FR-150). Когда операции появятся, источник
 * становится запросом Vue Query через сгенерированный клиент с ключами
 * `incidentKeys` / `processVersionKeys` и параметрами момента.
 */
import { ref, type Ref } from 'vue'
import type { BackendMode } from '@/shared/api/generated/model'

/** Источник данных виджета. */
export interface WidgetSource<T> {
  data: Ref<T | null>
  isPending: Ref<boolean>
  error: Ref<unknown>
  mode: Ref<BackendMode | null>
}

/** Источник без операции API: данных нет, ошибки нет. */
export const emptySource = <T>(): WidgetSource<T> => ({
  data: ref(null) as Ref<T | null>,
  isPending: ref(false),
  error: ref(undefined),
  mode: ref(null),
})
