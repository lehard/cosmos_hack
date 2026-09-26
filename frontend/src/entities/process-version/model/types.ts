/**
 * Входные данные раздела «Процесс» (FR-24): версии описания процесса в
 * читаемом виде. Поля — как у нормативного слоя (contracts/normative,
 * расширения BPMN `ant:*`); когда операция чтения версий появится в
 * contracts/openapi.yaml, сгенерированные типы подставляются сюда.
 * Просмотр схемы BPMN — эпик 10 (widgets/live-map), здесь — только текст.
 */
import type ru from '@/shared/i18n/ru.json'

/** Статус версии процесса — словарь `processVersion`. */
export type ProcessVersionStatus = 'draft' | 'on_approval' | 'active' | 'retired'

/** Вид элемента — ключи `process.elements.*`. */
export type ProcessElementKind = keyof (typeof ru)['process']['elements']

/** Свойство элемента — ключи `process.properties.*` (наша панель свойств, FR-12). */
export type ProcessPropertyKey = keyof (typeof ru)['process']['properties']

/** Значение свойства в читаемом виде. */
export type ProcessPropertyValue = string | number | boolean | null

/** Элемент описания процесса. */
export interface ProcessElement {
  /** id элемента BPMN. */
  id: string
  /** Смысловой ключ шага (`ant:properties/@stepKey`) — сопоставляет элементы версий. */
  step_key?: string | null
  /** Вид. */
  kind: ProcessElementKind
  /** Название. */
  name: string
  /** Цех (дорожка). */
  lane?: string | null
  /** Наши свойства. */
  properties: Partial<Record<ProcessPropertyKey, ProcessPropertyValue>>
  /** Пороги уверенности карты реакций по видам дефектов, б. п. */
  thresholds?: Record<string, number>
  /** id следующих элементов (исходящие переходы). */
  next?: string[]
}

/** Версия процесса в читаемом виде. */
export interface ProcessVersion {
  /** Идентификатор версии. */
  version_id: string
  /** Номер для людей, например `0.1`. */
  label: string
  /** Статус. */
  status: ProcessVersionStatus
  /** Автор черновика. */
  author?: string | null
  /** Создана. */
  created_at: string
  /** Вступила в силу. */
  effective_from?: string | null
  /** Кворум утверждения: есть подписей / нужно (FR-23). */
  quorum?: { have: number; need: number } | null
  /** Элементы в порядке маршрута. */
  elements: ProcessElement[]
}

/** Отличие версии от действующей — строки `process.diff.*` (FR-24). */
export type ProcessDiffEntry =
  | { kind: 'elementAdded'; element: string }
  | { kind: 'elementRemoved'; element: string }
  | { kind: 'presentationPointAdded'; step: string }
  | { kind: 'thresholdChanged'; element: string; defectType: string; from: number | null; to: number | null }
  | { kind: 'propertyChanged'; element: string; property: ProcessPropertyKey; from: ProcessPropertyValue; to: ProcessPropertyValue }
