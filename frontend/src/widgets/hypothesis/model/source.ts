/**
 * Источник данных виджета «hypothesis» (эпик 12). Будущая операция — гипотезы по несоответствию: `incident.hypothesis.computed` и записи людей, похожие случаи (FR-59, FR-60).
 *
 * Пока операции чтения нет в contracts/openapi.yaml (эпик 02), источник пуст —
 * виджет показывает пустое состояние, а не выдуманные данные (PRD §11.10,
 * FR-150). Подключение: запрос Vue Query через обёртку entities/* с ключами
 * по соглашению shared/api/keys.ts и параметрами момента (AD-21).
 */
import { emptySource, type WidgetSource } from '@/entities/incident'
import type { HypothesesModel } from '@/entities/incident'

/** Данные виджета «hypothesis». */
export const useHypothesesSource = (): WidgetSource<HypothesesModel> => emptySource<HypothesesModel>()
