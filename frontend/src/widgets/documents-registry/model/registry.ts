/**
 * Реестр документов (FR-65, FR-66, FR-139; AD-12, AD-13, AD-43; Д-72) —
 * чистые функции виджета «Документы» и окна документа: состояние для людей и
 * его тон, объект, вид по шаблону, класс ключа подписи, отбор строк.
 * Состояние считает сервер (`state`), здесь — только запасной расчёт из
 * статуса маршрута и бумаги (ответ без поля — старый сервер).
 */
import type { StatusTone } from '@/shared/api/generated/statuses'
import type { DocumentSummary } from '@/shared/api/generated/model'

/** Состояния реестра. */
export const DOC_STATES = ['draft', 'signing', 'paper', 'returned', 'signed', 'annulled'] as const
export type DocState = (typeof DOC_STATES)[number]

/** Тон плашки состояния (тона словаря статусов, NFR-UI-2). */
export const STATE_TONE: Record<DocState, StatusTone> = {
  draft: 'neutral',
  signing: 'attention',
  paper: 'info',
  returned: 'danger',
  signed: 'success',
  annulled: 'muted',
}

/** Состояние документа для людей (как RegistryState на сервере). */
export function docState(d: Pick<DocumentSummary, 'state' | 'status' | 'paper_status'>): DocState {
  if (d.state && (DOC_STATES as readonly string[]).includes(d.state)) return d.state as DocState
  switch (d.status) {
    case 'annulled':
      return 'annulled'
    case 'returned':
      return 'returned'
    case 'route_closed':
      return 'signed'
  }
  if (d.paper_status === 'printed') return 'paper'
  return d.status === 'signing' ? 'signing' : 'draft'
}

/** id шаблона из template_ref `‹id›@‹версия›`. */
export const templateId = (ref: string): string => ref.split('@')[0] ?? ref

/** Ключ текста вида документа: `docRegistry.kinds.‹id шаблона в camelCase›`. */
export const kindKey = (ref: string): string =>
  `docRegistry.kinds.${templateId(ref).replace(/-([a-z0-9])/g, (_, c: string) => c.toUpperCase())}`

/** Ключ текста вида объекта. */
export const subjectKey = (entity: string): string => `docRegistry.subjects.${entity}`

/** Отбор реестра — то же, что параметры documents.document.list. */
export interface RegistryFilter {
  process_id?: string
  item_id?: string
  template?: string
  state?: DocState
  q?: string
}

/** Параметры запроса без пустых полей. */
export function filterParams(f: RegistryFilter): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(f)) if (typeof v === 'string' && v.trim()) out[k] = v.trim()
  return out
}

/** Сколько документов в каждом состоянии — для счётчиков над списком. */
export function countByState(rows: readonly DocumentSummary[]): Record<DocState, number> {
  const out = Object.fromEntries(DOC_STATES.map((s) => [s, 0])) as Record<DocState, number>
  for (const r of rows) out[docState(r)]++
  return out
}

/** Виды документов, встречающиеся в списке (для фильтра «вид»), по порядку появления. */
export function templatesOf(rows: readonly DocumentSummary[]): string[] {
  const out: string[] = []
  for (const r of rows) {
    const id = templateId(r.template)
    if (!out.includes(id)) out.push(id)
  }
  return out
}

/** Изделия в списке (для фильтра «изделие»): id → подпись. */
export function itemsOf(rows: readonly DocumentSummary[]): { id: string; label: string }[] {
  const seen = new Map<string, string>()
  for (const r of rows) {
    for (const id of r.item_ids ?? []) {
      if (!seen.has(id)) seen.set(id, itemLabel(id))
    }
  }
  return [...seen].map(([id, label]) => ({ id, label })).sort((a, b) => a.label.localeCompare(b.label, 'ru'))
}

/** Подпись изделия по машинному id: `ENT01:F-017` → `Ф-017`. */
export function itemLabel(id: string): string {
  const local = id.includes(':') ? id.slice(id.lastIndexOf(':') + 1) : id
  return local.replace(/^F-/, 'Ф-').replace(/^R-/, 'К-').replace(/^C-/, 'КР-')
}

/** Класс хранения ключа подписи (Д-72) → ключ текста. */
export const keyStorageKey = (storage: string | undefined): string | null =>
  storage === 'hardware_token' ? 'docRegistry.key.hardwareToken' : storage === 'software_browser' ? 'docRegistry.key.softwareBrowser' : null
