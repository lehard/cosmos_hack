/**
 * Источник данных виджета «circumstances» (эпик 12). Будущая операция — чтение проекции `analysis.circumstances` по выбранному несоответствию (AD-29, FR-153).
 *
 * Пока операции чтения нет в contracts/openapi.yaml (эпик 02), источник пуст —
 * виджет показывает пустое состояние, а не выдуманные данные (PRD §11.10,
 * FR-150). Подключение: запрос Vue Query через обёртку entities/* с ключами
 * по соглашению shared/api/keys.ts и параметрами момента (AD-21).
 */
import { emptySource, type WidgetSource } from '@/entities/incident'
import type { CircumstancesModel } from '@/entities/incident'

/** Данные виджета «circumstances». */
export const useCircumstancesSource = (): WidgetSource<CircumstancesModel> => emptySource<CircumstancesModel>()
