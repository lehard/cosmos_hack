/**
 * Источник данных виджета «common-factors» (эпик 12). Будущая операция — общие факторы по выбранной группе несоответствий (FR-135).
 *
 * Пока операции чтения нет в contracts/openapi.yaml (эпик 02), источник пуст —
 * виджет показывает пустое состояние, а не выдуманные данные (PRD §11.10,
 * FR-150). Подключение: запрос Vue Query через обёртку entities/* с ключами
 * по соглашению shared/api/keys.ts и параметрами момента (AD-21).
 */
import { emptySource, type WidgetSource } from '@/entities/incident'
import type { CommonFactorsModel } from '@/entities/incident'

/** Данные виджета «common-factors». */
export const useCommonFactorsSource = (): WidgetSource<CommonFactorsModel> => emptySource<CommonFactorsModel>()
