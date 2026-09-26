/**
 * Токены дизайн-системы «Главный» — единственное место, где заданы цвета,
 * типографика, отступы, скругления, тени и ширины интерфейса (NFR-UI-2, UI-3).
 *
 * Стиль — минималистичный одноцветный enterprise: нейтральная холодно-серая шкала,
 * один акцент — только для главного действия, ссылок и выбора; остальные цвета —
 * тона словаря статусов из контракта (AD-30, contracts/statuses.yaml).
 *
 * Отсюда строятся и тема Naive UI (`naive.ts`), и CSS-переменные `--ant-*`
 * (`css.ts`) — виджеты берут значения только через них, не пишут цвета руками.
 */
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { Density } from '@/shared/api/generated/model'

/** Нейтральная шкала (холодный серый): 0 — белый, 900 — почти чёрный. */
export const neutral = {
  0: '#ffffff',
  25: '#fbfbfc',
  50: '#f5f6f8',
  100: '#eef0f3',
  150: '#e4e7eb',
  200: '#d8dce2',
  300: '#c1c6ce',
  400: '#99a0ab',
  500: '#6f7682',
  600: '#555c67',
  700: '#3d434c',
  800: '#282c33',
  900: '#181b20',
} as const

/** Единственный акцент — главное действие, ссылка, выбор, фокус. */
export const accent = {
  base: '#2553c7',
  hover: '#3563d6',
  pressed: '#1c43a6',
  soft: '#edf2fc',
  softBorder: '#c9d6f4',
} as const

/**
 * Смешать цвет с белым — мягкий фон статуса (доля цвета 0…1).
 * @param hex — цвет #rrggbb
 * @param amount — доля исходного цвета
 */
export function tint(hex: string, amount: number): string {
  const n = Number.parseInt(hex.slice(1), 16)
  const ch = (shift: number) => {
    const c = (n >> shift) & 0xff
    return Math.round(255 - (255 - c) * amount)
      .toString(16)
      .padStart(2, '0')
  }
  return `#${ch(16)}${ch(8)}${ch(0)}`
}

/** Статусные цвета — из словаря статусов контракта; мягкий фон — производный. */
export const status: Record<StatusTone, { base: string; soft: string }> = Object.fromEntries(
  (Object.keys(statusPalette) as StatusTone[]).map((tone) => [tone, { base: statusPalette[tone], soft: tint(statusPalette[tone], 0.1) }]),
) as Record<StatusTone, { base: string; soft: string }>

/** Смысловые цвета интерфейса. */
export const color = {
  /** Фон приложения (за карточками). */
  bgApp: neutral[50],
  /** Поверхность карточки, панели, поля. */
  surface: neutral[0],
  /** Приглушённая поверхность: шапка таблицы, выделенный блок. */
  surfaceSubtle: neutral[25],
  /** Наведение на строку или пункт. */
  surfaceHover: neutral[100],
  /** Рамки и разделители. */
  border: neutral[150],
  borderStrong: neutral[200],
  /** Текст: основной, вторичный, приглушённый, выключенный. */
  text: neutral[900],
  text2: neutral[600],
  text3: neutral[500],
  textDisabled: neutral[400],
  textOnAccent: neutral[0],
} as const

/** Шрифты — локальные пакеты PT Sans и PT Mono (NFR-UI-2). */
export const font = {
  family: "'PT Sans', system-ui, sans-serif",
  familyMono: "'PT Mono', ui-monospace, monospace",
  /** Шкала размеров, px. */
  size: { xs: 12, sm: 13, md: 14, lg: 16, xl: 18, xxl: 22, display: 28 },
  /** У PT Sans два начертания. */
  weight: { regular: 400, bold: 700 },
  lineHeight: { tight: 1.25, normal: 1.45 },
} as const

/** Отступы — шаг 4 px. */
export const space = { 0: 0, 1: 4, 2: 8, 3: 12, 4: 16, 5: 20, 6: 24, 8: 32, 10: 40, 12: 48 } as const

/** Скругления, px. */
export const radius = { sm: 4, md: 6, lg: 8, pill: 999 } as const

/** Тени: карточка почти плоская, всплывающее — заметнее. */
export const shadow = {
  sm: '0 1px 2px rgba(16, 24, 40, 0.05)',
  md: '0 1px 2px rgba(16, 24, 40, 0.04), 0 4px 12px -2px rgba(16, 24, 40, 0.08)',
  lg: '0 12px 32px -8px rgba(16, 24, 40, 0.18), 0 2px 6px rgba(16, 24, 40, 0.06)',
  focus: `0 0 0 3px ${tint(accent.base, 0.22)}`,
} as const

/** Ширины, px. */
export const width = {
  /** Карточка экрана входа. */
  auth: 420,
  /** Правая панель контекста стола. */
  side: 400,
  sideMin: 320,
  /** Очередь слева на столе «очередь — основное — контекст». */
  queue: 360,
  queueMin: 280,
  /** Поиск в шапке. */
  search: 320,
  /** Высота шапки. */
  header: 56,
} as const

/** Параметры плотности стола (AD-21): мастеру и исполнителю — крупно, технологу — плотно. */
export interface DensityTokens {
  /** Основной текст и подписи, px. */
  fontBody: number
  fontMeta: number
  fontTitle: number
  /** Высота элементов управления: маленькие, средние, крупные, px. */
  controlSmall: number
  controlMedium: number
  controlLarge: number
  /** Внутренние отступы секции и ячейки таблицы, px. */
  padSection: number
  padCellX: number
  padCellY: number
  /** Промежуток между строками и блоками, px. */
  gap: number
}

export const density: Record<Density, DensityTokens> = {
  large: { fontBody: 16, fontMeta: 14, fontTitle: 18, controlSmall: 32, controlMedium: 40, controlLarge: 48, padSection: 20, padCellX: 12, padCellY: 10, gap: 12 },
  comfortable: { fontBody: 14, fontMeta: 12, fontTitle: 15, controlSmall: 28, controlMedium: 34, controlLarge: 40, padSection: 16, padCellX: 10, padCellY: 7, gap: 8 },
  compact: { fontBody: 13, fontMeta: 12, fontTitle: 14, controlSmall: 24, controlMedium: 28, controlLarge: 34, padSection: 12, padCellX: 8, padCellY: 4, gap: 6 },
}

/** Все токены одним объектом. */
export const tokens = { neutral, accent, status, color, font, space, radius, shadow, width, density } as const
