/**
 * Контракт между столом и виджетом (AD-21, NFR-UI-1, NFR-EXT-1).
 *
 * Стол (normative/desks/‹роль›.yaml) ставит виджет из реестра
 * (widgets/registry.ts) в слот и передаёт ему эти свойства. Виджет не знает,
 * на чьём столе стоит: «одна правда — разные взгляды» задаются срезом и плотностью.
 */
import type { Density } from '@/shared/api/generated/model'

export type { Density }

/** Свойства, которые оболочка передаёт каждому виджету. */
export interface WidgetProps {
  /** id виджета в реестре. */
  widgetId: string
  /** Ключ текста заголовка (из реестра). */
  titleKey: string
  /** id слота на столе. */
  slotId: string
  /** Параметры среза из yaml стола. */
  slice: Record<string, unknown>
  /** Плотность: мастеру и исполнителю — крупно, технологу — плотно. */
  density: Density
}

/**
 * Состояние данных виджета — минимум четыре состояния каждого экрана
 * (AD-21, NFR-UI-4). «Оценка невозможна» ≠ «норма», «признак дефекта» ≠ «брак».
 */
export type WidgetDataState = 'normal' | 'defect_indication' | 'unable_to_assess' | 'input_error'

/** Размер компонентов Naive UI по плотности. */
export const naiveSizeOf = (density: Density): 'large' | 'medium' | 'small' =>
  density === 'large' ? 'large' : density === 'compact' ? 'small' : 'medium'
