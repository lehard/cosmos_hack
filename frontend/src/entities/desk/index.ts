/**
 * Стол активной роли (AD-21, NFR-EXT-1): данные normative/desks/‹роль›.yaml,
 * отданные сервером (access.Queries.Desks). Новый стол — правка yaml, не кода.
 */
import { useAccessDeskRead } from '@/shared/api/generated/client'

export type { Density, Desk, DeskArea, DeskLayout, DeskSlot, DeskTab } from '@/shared/api/generated/model'

/** Ключ кэша стола. */
export const deskKey = ['desk', 'me'] as const

/** Стол роли текущего сеанса. */
export const useDesk = () => useAccessDeskRead({ query: { queryKey: deskKey, staleTime: Infinity } })

/** Области раскладок: в какие области можно ставить слоты (та же таблица — в normative/desks/desk.schema.json). */
export const LAYOUT_AREAS = {
  single: ['main'],
  'main-side': ['main', 'right'],
  'queue-main-side': ['left', 'main', 'right'],
  overview: ['top', 'main', 'right', 'bottom'],
} as const
