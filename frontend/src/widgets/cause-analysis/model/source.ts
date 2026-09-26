/**
 * Источник данных виджета «cause-analysis» (эпик 12). Будущая операция — группы несоответствий «вид дефекта × операция × оборудование».
 *
 * Пока операции чтения нет в contracts/openapi.yaml (эпик 02), источник пуст —
 * виджет показывает пустое состояние, а не выдуманные данные (PRD §11.10,
 * FR-150). Подключение: запрос Vue Query через обёртку entities/* с ключами
 * по соглашению shared/api/keys.ts и параметрами момента (AD-21).
 */
import { emptySource, type WidgetSource } from '@/entities/incident'
import type { NcGroup } from '@/entities/incident'

/** Данные виджета «cause-analysis». */
export const useNcGroupsSource = (): WidgetSource<NcGroup[]> => emptySource<NcGroup[]>()
