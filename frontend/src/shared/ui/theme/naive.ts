/**
 * Тема Naive UI из токенов (NFR-UI-2): `themeOverrides` для корневого
 * `NConfigProvider` и поправки плотности для виджета (AD-21: мастеру и
 * исполнителю — крупно, технологу — плотно). Главное действие — акцент;
 * успех, внимание, ошибка, информация — тона словаря статусов (AD-30).
 */
import type { GlobalThemeOverrides } from 'naive-ui'
import type { Density } from '@/shared/api/generated/model'
import { accent, color, density as densityTokens, font, neutral, radius, shadow, status } from './tokens'

const px = (n: number) => `${n}px`

/** Тема всего приложения (плотность «обычно»). */
export const themeOverrides: GlobalThemeOverrides = {
  common: {
    fontFamily: font.family,
    fontFamilyMono: font.familyMono,
    fontWeightStrong: String(font.weight.bold),
    lineHeight: String(font.lineHeight.normal),
    fontSize: px(font.size.md),
    fontSizeMini: px(font.size.xs),
    fontSizeTiny: px(font.size.xs),
    fontSizeSmall: px(font.size.sm),
    fontSizeMedium: px(font.size.md),
    fontSizeLarge: px(font.size.lg),
    fontSizeHuge: px(font.size.xl),
    borderRadius: px(radius.md),
    borderRadiusSmall: px(radius.sm),

    primaryColor: accent.base,
    primaryColorHover: accent.hover,
    primaryColorPressed: accent.pressed,
    primaryColorSuppl: accent.hover,
    infoColor: status.info.base,
    infoColorHover: status.info.base,
    infoColorPressed: status.info.base,
    infoColorSuppl: status.info.base,
    successColor: status.success.base,
    successColorHover: status.success.base,
    successColorPressed: status.success.base,
    successColorSuppl: status.success.base,
    warningColor: status.attention.base,
    warningColorHover: status.attention.base,
    warningColorPressed: status.attention.base,
    warningColorSuppl: status.attention.base,
    errorColor: status.danger.base,
    errorColorHover: status.danger.base,
    errorColorPressed: status.danger.base,
    errorColorSuppl: status.danger.base,

    textColorBase: color.text,
    textColor1: color.text,
    textColor2: color.text,
    textColor3: color.text3,
    textColorDisabled: color.textDisabled,
    placeholderColor: neutral[400],
    iconColor: neutral[500],
    dividerColor: color.border,
    borderColor: color.borderStrong,
    hoverColor: color.surfaceHover,
    pressedColor: neutral[150],
    tableHeaderColor: color.surfaceSubtle,
    tableColorHover: color.bgApp,
    tableColorStriped: color.surfaceSubtle,
    actionColor: color.surfaceSubtle,
    tabColor: color.bgApp,
    codeColor: neutral[100],
    tagColor: neutral[100],
    railColor: neutral[200],
    progressRailColor: neutral[150],
    inputColorDisabled: neutral[50],
    buttonColor2: neutral[100],
    buttonColor2Hover: neutral[150],
    buttonColor2Pressed: neutral[200],
    bodyColor: color.bgApp,
    cardColor: color.surface,
    modalColor: color.surface,
    popoverColor: color.surface,
    boxShadow1: shadow.sm,
    boxShadow2: shadow.lg,
    boxShadow3: shadow.lg,
  },
  Card: {
    borderRadius: px(radius.lg),
    borderColor: color.border,
    titleFontWeight: String(font.weight.bold),
    titleFontSizeSmall: px(font.size.md),
    titleFontSizeMedium: px(font.size.lg - 1),
    titleFontSizeLarge: px(font.size.xl),
    paddingSmall: '12px 16px',
    paddingMedium: '16px 20px',
    paddingLarge: '20px 24px',
  },
  Button: {
    fontWeight: String(font.weight.regular),
    fontWeightStrong: String(font.weight.bold),
  },
  DataTable: {
    thFontWeight: String(font.weight.bold),
    thTextColor: color.text3,
    borderColor: color.border,
  },
  Tabs: {
    tabTextColorLine: color.text2,
    tabTextColorActiveLine: color.text,
    tabTextColorHoverLine: color.text,
    barColor: accent.base,
    tabFontWeightActive: String(font.weight.bold),
  },
  Form: {
    labelTextColor: color.text2,
    labelFontSizeTopMedium: px(font.size.sm),
    feedbackFontSizeMedium: px(font.size.sm),
  },
  Tag: {
    borderRadius: px(radius.sm),
  },
  Alert: {
    borderRadius: px(radius.md),
  },
  Layout: {
    color: color.bgApp,
    headerColor: color.surface,
    headerBorderColor: color.border,
  },
  Tooltip: {
    color: neutral[800],
    textColor: neutral[0],
    borderRadius: px(radius.sm),
  },
}

/**
 * Поправки темы под плотность — для вложенного `NConfigProvider` внутри виджета:
 * размеры шрифта и высоты элементов управления.
 * @param d — плотность из среза стола
 */
export function densityOverrides(d: Density): GlobalThemeOverrides {
  const t = densityTokens[d]
  return {
    common: {
      fontSize: px(t.fontBody),
      fontSizeSmall: px(t.fontBody),
      fontSizeMedium: px(t.fontBody),
      fontSizeTiny: px(t.fontMeta),
      heightTiny: px(t.controlSmall - 6),
      heightSmall: px(t.controlSmall),
      heightMedium: px(t.controlMedium),
      heightLarge: px(t.controlLarge),
    },
  }
}
