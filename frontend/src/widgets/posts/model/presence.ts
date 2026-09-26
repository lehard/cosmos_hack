/**
 * Присутствие на посту → текст и тон (FR-6). Тоны — палитра словаря статусов
 * (AD-30): на месте — зелёный; расхождение СКУД и ключа — жёлтый или красный;
 * данных нет — серый «неизвестно», а не «на месте» (NFR-UI-4).
 */
import type { PostPresence, PostRow } from '@/entities/workplace'
import type { StatusTone } from '@/shared/api/generated/statuses'
import type { WidgetDataState } from '@/shared/config/widget'

export const PRESENCE: Record<PostPresence, { key: string; tone: StatusTone }> = {
  present: { key: 'liveMap.posts.present', tone: 'success' },
  key_missing: { key: 'liveMap.posts.keyMissing', tone: 'attention' },
  owner_absent: { key: 'liveMap.posts.ownerAbsent', tone: 'danger' },
  absent: { key: 'mapWidgets.posts.absent', tone: 'danger' },
  not_assigned: { key: 'liveMap.posts.notAssigned', tone: 'neutral' },
  unknown: { key: 'empty.noDataUnknown', tone: 'neutral' },
}

/** Состояние панели: есть посты без данных присутствия — «оценка невозможна». */
export const postsState = (rows: readonly PostRow[] | null | undefined): WidgetDataState =>
  rows?.some((r) => r.presence === 'unknown') ? 'unable_to_assess' : 'normal'
