/**
 * Присутствие на посту → текст и тон (FR-6). Тоны — палитра словаря статусов
 * (AD-30): на месте — зелёный; расхождение СКУД и ключа — жёлтый или красный;
 * данных нет — серый «неизвестно», а не «на месте» (NFR-UI-4).
 */
import { PRESENCE_TEXT, PRESENCE_TONE, type PostPresence, type PostRow } from '@/entities/workplace'
import type { StatusTone } from '@/shared/api/generated/statuses'
import type { WidgetDataState } from '@/shared/config/widget'

export const PRESENCE = Object.fromEntries(
  (Object.keys(PRESENCE_TEXT) as PostPresence[]).map((p) => [p, { key: PRESENCE_TEXT[p], tone: PRESENCE_TONE[p] }]),
) as Record<PostPresence, { key: string; tone: StatusTone }>

/** Состояние панели: есть посты без данных присутствия — «оценка невозможна». */
export const postsState = (rows: readonly PostRow[] | null | undefined): WidgetDataState =>
  rows?.some((r) => r.presence === 'unknown') ? 'unable_to_assess' : 'normal'
