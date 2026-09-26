/**
 * Правила показа общего журнала (PRD §3a; AD-2, AD-44; FR-68, FR-140; Д-78):
 * чистые функции над записями API — вид записи, статус подписи, источник,
 * название события из каталога, семейства типов для фильтра, содержимое data.
 *
 * Интерфейс ничего не пересчитывает (AD-21): тексты — ключами vue-i18n, тон —
 * палитра статусов. Код без словаря показывается как UNKNOWN(код) и не
 * подменяется похожим (NFR-UI-4).
 */
import type { JournalEntryKind, JournalEntryView, JournalSignatureStatus } from '@/entities/integrity'
import { SIGNATURE_CLASS_TEXT, SOURCE_KIND_TEXT } from '@/entities/item'
import type { StatusTone } from '@/shared/api/generated/statuses'
import { eventCatalog } from '@/shared/contracts/catalog'
import { codeToKey } from '@/shared/i18n'

type Translate = (key: string) => string

const catalog = eventCatalog as Record<string, { title: string; kind: string } | undefined>

/** Вид записи (AD-2, кейс §7.2): подпись, подпись фильтра (множественное), тон. */
export const ENTRY_KIND: Record<JournalEntryKind, { key: string; filterKey: string; tone: StatusTone }> = {
  fact: { key: 'widgets.journal.kind.fact', filterKey: 'widgets.journal.filter.fact', tone: 'neutral' },
  reaction: { key: 'widgets.journal.kind.reaction', filterKey: 'widgets.journal.filter.reaction', tone: 'info' },
  decision: { key: 'widgets.journal.kind.decision', filterKey: 'widgets.journal.filter.decision', tone: 'qualified' },
  service: { key: 'widgets.journal.kind.service', filterKey: 'widgets.journal.filter.service', tone: 'muted' },
}

/**
 * Статус проверки подписи (FR-68): «проверить нельзя» и «не проверялась» ≠
 * «действительна» — зелёный только у `valid`.
 */
export const SIGNATURE: Record<JournalSignatureStatus, { key: string; hintKey: string; tone: StatusTone }> = {
  valid: { key: 'widgets.journal.signature.valid', hintKey: 'widgets.journal.signature.validHint', tone: 'success' },
  invalid: { key: 'widgets.journal.signature.invalid', hintKey: 'widgets.journal.signature.invalidHint', tone: 'danger' },
  unverifiable: { key: 'widgets.journal.signature.unverifiable', hintKey: 'widgets.journal.signature.unverifiableHint', tone: 'attention' },
  not_checked: { key: 'widgets.journal.signature.notChecked', hintKey: 'widgets.journal.signature.notCheckedHint', tone: 'neutral' },
}

/** Текст по таблице или UNKNOWN(код). */
const known = <T extends string>(table: Partial<Record<T, string>>, code: string, t: Translate): string => {
  const key = table[code as T]
  return key ? t(key) : `UNKNOWN(${code})`
}

/** Вид записи словами. */
export const kindText = (kind: string, t: Translate): string => {
  const k = ENTRY_KIND[kind as JournalEntryKind]
  return k ? t(k.key) : `UNKNOWN(${kind})`
}
/** Тон вида записи. */
export const kindTone = (kind: string): StatusTone => ENTRY_KIND[kind as JournalEntryKind]?.tone ?? 'neutral'

/** Статус подписи словами. */
export const signatureText = (s: string, t: Translate): string => {
  const x = SIGNATURE[s as JournalSignatureStatus]
  return x ? t(x.key) : `UNKNOWN(${s})`
}
/** Тон статуса подписи. */
export const signatureTone = (s: string): StatusTone => SIGNATURE[s as JournalSignatureStatus]?.tone ?? 'neutral'

/** Человеческое название типа записи из каталога; нет в каталоге — null. */
export const eventTitle = (eventType: string): string | null => catalog[eventType]?.title ?? null

/**
 * Источник записи словами (FR-140): у факта — вид источника; у вывода системы
 * и служебной записи — «вывод системы»; у решения — «решение человека»; факт
 * без вида источника — «источник неизвестен» (не додумывать, FR-123).
 */
export function sourceText(e: Pick<JournalEntryView, 'source_kind' | 'entry_kind'>, t: Translate): string {
  if (e.source_kind) return known(SOURCE_KIND_TEXT, e.source_kind, t)
  if (e.entry_kind === 'reaction' || e.entry_kind === 'service') return t('timeline.sourceKind.system')
  if (e.entry_kind === 'decision') return t('timeline.layer.decision')
  return t('widgets.passport.source.unknown')
}

/** Класс происхождения подписи словами (AD-2). */
export const provenanceText = (cls: string, t: Translate): string => known(SIGNATURE_CLASS_TEXT, cls, t)

/** Цепочка словами: основная или критических действий (AD-8). */
export const chainText = (chain: string, t: Translate): string =>
  chain === 'main' ? t('widgets.journal.chain.main') : chain === 'ca' ? t('widgets.journal.chain.ca') : `UNKNOWN(${chain})`

/** Отпечаток сокращённо: алгоритм и первые знаки (полный — в подсказке). */
export function shortDigest(ref: string, keep = 12): string {
  const i = ref.indexOf(':')
  const algo = i > 0 ? ref.slice(0, i + 1) : ''
  const hex = i > 0 ? ref.slice(i + 1) : ref
  return hex.length > keep ? `${algo}${hex.slice(0, keep)}…` : ref
}

/** Семейство типа записи — первое слово кода (`item.carrier.applied` → `item`). */
export const familyOf = (eventType: string): string => eventType.split('.')[0] ?? eventType

/** Ключ названия семейства. */
export const familyKey = (family: string): string => `widgets.journal.family.${codeToKey(family)}`

/** Вариант выбора типа события. */
export type EventTypeOption = {
  label: string
  value: string
}
/** Группа вариантов: семейство и его типы. */
export type EventTypeGroup = {
  type: 'group'
  key: string
  label: string
  children: EventTypeOption[]
}

/**
 * Типы событий для фильтра — по семействам каталога (AD-40): первым в группе —
 * «все записи семейства» (префикс `семейство.`), дальше типы по названию.
 * @param t — функция текстов
 * @param te — есть ли ключ
 */
export function eventTypeGroups(t: (key: string, named?: Record<string, unknown>) => string, te: (key: string) => boolean): EventTypeGroup[] {
  const byFamily = new Map<string, EventTypeOption[]>()
  for (const [code, entry] of Object.entries(catalog)) {
    if (!entry) continue
    const f = familyOf(code)
    const list = byFamily.get(f) ?? []
    list.push({ label: entry.title, value: code })
    byFamily.set(f, list)
  }
  const familyLabel = (f: string) => (te(familyKey(f)) ? t(familyKey(f)) : `UNKNOWN(${f})`)
  return [...byFamily.entries()]
    .map(([f, list]) => ({
      type: 'group' as const,
      key: f,
      label: familyLabel(f),
      children: [
        { label: t('widgets.journal.filter.wholeFamily', { family: familyLabel(f) }), value: `${f}.` },
        ...list.sort((a, b) => a.label.localeCompare(b.label, 'ru')),
      ],
    }))
    .sort((a, b) => a.label.localeCompare(b.label, 'ru'))
}

/** Строка содержимого data: ключ, простое значение или свёрнутый JSON. */
export interface DataRow {
  key: string
  /** Текст простого значения; для сложного — null. */
  text: string | null
  /** Сложное значение (объект, массив) — JSON с отступами. */
  json: string | null
}

/** Содержимое data записи — «ключ — значение», сложное — свёрнутым JSON. */
export function dataRows(data: Record<string, unknown> | undefined): DataRow[] {
  if (!data) return []
  return Object.entries(data).map(([key, v]) => {
    if (v === null || v === undefined) return { key, text: '—', json: null }
    if (typeof v === 'object') return { key, text: null, json: JSON.stringify(v, null, 2) }
    return { key, text: String(v), json: null }
  })
}
