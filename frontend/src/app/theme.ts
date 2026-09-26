/**
 * Тема Naive UI (NFR-UI-2): минималистично и одноцветно. Главное действие —
 * тёмно-серое; цвета успеха, внимания, ошибки и информации — тона словаря
 * статусов из контракта (AD-30), других цветов в интерфейсе нет.
 */
import type { GlobalThemeOverrides } from 'naive-ui'
import { statusPalette } from '@/shared/api/generated/statuses'

export const themeOverrides: GlobalThemeOverrides = {
  common: {
    fontFamily: "'PT Sans', system-ui, sans-serif",
    fontFamilyMono: "'PT Mono', ui-monospace, monospace",
    primaryColor: '#1f2937',
    primaryColorHover: '#374151',
    primaryColorPressed: '#111827',
    primaryColorSuppl: '#374151',
    infoColor: statusPalette.info,
    successColor: statusPalette.success,
    warningColor: statusPalette.attention,
    errorColor: statusPalette.danger,
    borderRadius: '4px',
  },
}
