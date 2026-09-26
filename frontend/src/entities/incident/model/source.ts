/**
 * Источник данных виджета разбора: то, что контейнер виджета передаёт рамке
 * (загрузка, ошибка, режим fixtures | live) и представлению (данные).
 *
 * Источник — запрос Vue Query через сгенерированный клиент (api.ts сущностей incident и process-version);
 * пустой источник — когда нечего читать (ничего не выбрано). Заготовок данных в
 * коде интерфейса нет (PRD §11.10, FR-150).
 */
import { ref, type Ref } from 'vue'
import type { BackendMode } from '@/shared/api/generated/model'

/** Источник данных виджета. */
export interface WidgetSource<T> {
  data: Readonly<Ref<T | null>>
  isPending: Readonly<Ref<boolean>>
  error: Readonly<Ref<unknown>>
  mode: Readonly<Ref<BackendMode | null>>
  /** `seq`, на котором построен ответ, — `basis_seq` команд (AD-39); null — нет. */
  basisSeq: Readonly<Ref<number | null>>
}

/** Источник без операции API: данных нет, ошибки нет. */
export const emptySource = <T>(): WidgetSource<T> => ({
  data: ref(null) as Ref<T | null>,
  isPending: ref(false),
  error: ref(undefined),
  mode: ref(null),
  basisSeq: ref(null),
})
