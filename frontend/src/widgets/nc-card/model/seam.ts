/**
 * Участок шва для схемы (разбор «Камеры + ИИ»): зона типа изделия вида «участок
 * шва» и состояние её проверки по паспорту изделия.
 */
export interface SeamZone {
  zone_id: string
  name: string
  /** Проверка зоны по паспорту: inspected / stale / not_inspected; нет — неизвестно. */
  status?: string
}
