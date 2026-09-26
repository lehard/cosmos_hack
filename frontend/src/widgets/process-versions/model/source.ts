/**
 * Источник данных виджета «process-versions» (эпик 12). Будущая операция — версии описания процесса в читаемом виде (нормативный слой, FR-24).
 *
 * Пока операции чтения нет в contracts/openapi.yaml (эпик 02), источник пуст —
 * виджет показывает пустое состояние, а не выдуманные данные (PRD §11.10,
 * FR-150). Подключение: запрос Vue Query через обёртку entities/* с ключами
 * по соглашению shared/api/keys.ts и параметрами момента (AD-21).
 */
import { emptySource, type WidgetSource } from '@/entities/incident'
import type { ProcessVersion } from '@/entities/process-version'

/** Данные виджета «process-versions». */
export const useProcessVersionsSource = (): WidgetSource<ProcessVersion[]> => emptySource<ProcessVersion[]>()
