/**
 * CSS-переменные `--ant-*` из токенов (NFR-UI-2): одна таблица стилей на всё
 * приложение — `:root` (плотность «обычно») и классы `.ant-density-‹плотность›`,
 * которые переопределяют размеры шрифта, высоты и отступы внутри виджета (AD-21).
 *
 * Список имён переменных — в README дизайн-системы (`shared/ui/README.md`).
 */
import type { Density } from '@/shared/api/generated/model'
import { accent, color, density as densityTokens, font, neutral, radius, shadow, space, status, width, type DensityTokens } from './tokens'

/** camelCase → kebab-case с отделением цифр: text2 → text-2. */
const kebab = (s: string) => s.replace(/([a-z])([A-Z0-9])/g, '$1-$2').toLowerCase()
const px = (n: number) => `${n}px`

/** Переменные плотности. */
export function densityVariables(d: DensityTokens): Record<string, string> {
  return {
    '--ant-fs-body': px(d.fontBody),
    '--ant-fs-meta': px(d.fontMeta),
    '--ant-fs-title': px(d.fontTitle),
    '--ant-control-h-sm': px(d.controlSmall),
    '--ant-control-h-md': px(d.controlMedium),
    '--ant-control-h-lg': px(d.controlLarge),
    '--ant-pad-section': px(d.padSection),
    '--ant-pad-cell-x': px(d.padCellX),
    '--ant-pad-cell-y': px(d.padCellY),
    '--ant-gap': px(d.gap),
  }
}

/** Все переменные `:root`. */
export function rootVariables(): Record<string, string> {
  const v: Record<string, string> = {}
  for (const [k, c] of Object.entries(neutral)) v[`--ant-n-${k}`] = c
  for (const [k, c] of Object.entries(accent)) v[k === 'base' ? '--ant-accent' : `--ant-accent-${kebab(k)}`] = c
  for (const [tone, c] of Object.entries(status)) {
    v[`--ant-status-${tone}`] = c.base
    v[`--ant-status-${tone}-soft`] = c.soft
  }
  for (const [k, c] of Object.entries(color)) v[`--ant-${kebab(k)}`] = c
  v['--ant-font'] = font.family
  v['--ant-font-mono'] = font.familyMono
  for (const [k, n] of Object.entries(font.size)) v[`--ant-fs-${k}`] = px(n)
  for (const [k, n] of Object.entries(font.weight)) v[`--ant-fw-${k}`] = String(n)
  for (const [k, n] of Object.entries(font.lineHeight)) v[`--ant-lh-${k}`] = String(n)
  for (const [k, n] of Object.entries(space)) v[`--ant-space-${k}`] = px(n)
  for (const [k, n] of Object.entries(radius)) v[`--ant-radius-${k}`] = px(n)
  for (const [k, s] of Object.entries(shadow)) v[`--ant-shadow-${k}`] = s
  for (const [k, n] of Object.entries(width)) v[`--ant-w-${kebab(k)}`] = px(n)
  return { ...v, ...densityVariables(densityTokens.comfortable) }
}

const block = (selector: string, vars: Record<string, string>) =>
  `${selector}{${Object.entries(vars)
    .map(([k, val]) => `${k}:${val}`)
    .join(';')}}`

/** Текст таблицы стилей с переменными. */
export function tokensStylesheet(): string {
  const densities = (Object.keys(densityTokens) as Density[]).map((d) => block(`.ant-density-${d}`, densityVariables(densityTokens[d])))
  return [block(':root', rootVariables()), ...densities].join('\n')
}

/** Класс плотности для корня виджета или стола. */
export const densityClass = (d: Density) => `ant-density-${d}`

/**
 * Поставить переменные в документ (один раз, до монтирования приложения).
 * @param doc — документ (в тестах — happy-dom)
 */
export function installTokens(doc: Document = document): void {
  const id = 'ant-tokens'
  let el = doc.getElementById(id) as HTMLStyleElement | null
  if (!el) {
    el = doc.createElement('style')
    el.id = id
    doc.head.prepend(el)
  }
  el.textContent = tokensStylesheet()
}
