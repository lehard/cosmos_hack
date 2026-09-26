/**
 * Источник данных виджета «risk-scope» (эпик 12). Будущая операция — версии области риска инцидента: `incident.scope.*`, `incident.membership.changed` (FR-61, FR-62).
 *
 * Пока операции чтения нет в contracts/openapi.yaml (эпик 02), источник пуст —
 * виджет показывает пустое состояние, а не выдуманные данные (PRD §11.10,
 * FR-150). Подключение: запрос Vue Query через обёртку entities/* с ключами
 * по соглашению shared/api/keys.ts и параметрами момента (AD-21).
 */
import { emptySource, type WidgetSource } from '@/entities/incident'
import type { RiskScopeModel } from '@/entities/incident'

/** Данные виджета «risk-scope». */
export const useRiskScopeSource = (): WidgetSource<RiskScopeModel> => emptySource<RiskScopeModel>()
