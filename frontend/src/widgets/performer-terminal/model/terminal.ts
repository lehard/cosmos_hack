/**
 * Терминал исполнителя (FR-137, PRD §3a «Исполнитель — терминал»): только своё
 * рабочее место и текущие изделия на нём. Чистые функции над ответами сервера.
 *
 * - Рабочее место — из сеанса (барьер 2, AD-15): нет места — действий нет.
 * - Оборудование поста — `station_id` = рабочее место; предупреждения
 *   («ресурс инструмента 73/75») и поверку считает сервер.
 * - Операции — операции цеха рабочего места по схеме процесса (AD-17).
 * - Изделия у поста — текущее изделие поста, изделие текущего выполнения и
 *   изделия на выбранном шаге (очередь и в работе).
 */
import type { EquipmentState, EquipmentWarning, RefEquipment, RunProfile } from '@/entities/equipment'
import { toolLifeOf } from '@/entities/equipment'
import type { ItemRow } from '@/entities/operation'
import type { PostRow } from '@/entities/workplace'

/** Предупреждение терминала. */
export type TerminalWarning =
  | { kind: 'tool_life'; equipment: string; used: number; limit: number; text: string }
  | { kind: 'equipment'; equipment: string; warning: EquipmentWarning }
  | { kind: 'unusable'; equipment: string; reason: string }
  | { kind: 'no_data'; equipment: string }

/**
 * Предупреждения оборудования поста: ресурс инструмента — с числами, если
 * источник их передал; прочие — текстом сервера; непригодность по поверке — из
 * справочника; оборудование без данных — «нет данных», а не «норма».
 */
export function terminalWarnings(equipment: readonly EquipmentState[], registry: readonly RefEquipment[] | null | undefined): TerminalWarning[] {
  const out: TerminalWarning[] = []
  for (const e of equipment) {
    const life = toolLifeOf(e)
    for (const w of e.warnings) {
      if (w.kind === 'tool_life_warning' && life) out.push({ kind: 'tool_life', equipment: e.title, ...life, text: w.text })
      else out.push({ kind: 'equipment', equipment: e.title, warning: w })
    }
    const reg = registry?.find((r) => r.equipment_id === e.equipment_id)
    if (reg && !reg.usable) out.push({ kind: 'unusable', equipment: e.title, reason: reg.unusable_reason ?? 'unknown' })
    if (e.condition === 'unknown') out.push({ kind: 'no_data', equipment: e.title })
  }
  return out
}

/** Текущее выполнение на посту: id из оборудования, иначе начатое с этого терминала. */
export function currentRunId(equipment: readonly EquipmentState[], startedHere: string | null): string | null {
  return equipment.find((e) => e.current_run_id)?.current_run_id ?? startedHere
}

/** Изделие у поста для экрана. */
export interface PostItem {
  item_id: string
  label: string
  /** Строка списка изделий шага (статус, положение), если есть. */
  row: ItemRow | null
}

/**
 * Изделия у поста: текущее изделие поста, изделие выполнения, изделия шага в
 * работе и в изоляции — без повторов, не больше `limit`.
 */
export function postItems(post: PostRow | null, run: RunProfile | null, atStep: readonly ItemRow[] | null | undefined, limit = 6): PostItem[] {
  const out = new Map<string, PostItem>()
  const rowOf = (id: string) => atStep?.find((r) => r.item_id === id) ?? null
  const add = (id: string, label: string) => {
    if (!out.has(id)) out.set(id, { item_id: id, label, row: rowOf(id) })
  }
  if (post?.current_item) add(post.current_item.item_id, post.current_item.label)
  if (run?.item_id && !run.finished_at) add(run.item_id, rowOf(run.item_id)?.label ?? run.item_id)
  for (const r of atStep ?? []) {
    if (r.status.position === 'in_progress' || r.status.position === 'isolated') add(r.item_id, r.label)
  }
  return [...out.values()].slice(0, limit)
}

/** Изделие ждёт контроля — операцию начинать нельзя («сначала контроль», FR-137). */
export const inspectionFirst = (row: ItemRow | null | undefined): boolean =>
  !!row && (row.status.position === 'at_inspection' || row.status.position === 'at_presentation_point')

/**
 * Изделия шага для «начать операцию»: в очереди — можно; ждущие контроля —
 * видны, но выключены с пометкой «сначала контроль».
 */
export const startCandidates = (atStep: readonly ItemRow[] | null | undefined): { row: ItemRow; blocked: boolean }[] =>
  (atStep ?? []).filter((r) => r.status.position === 'in_queue' || inspectionFirst(r)).map((row) => ({ row, blocked: inspectionFirst(row) }))

/** Состояние терминала: оборудование поста без данных — «оценка невозможна». */
export const terminalState = (equipment: readonly EquipmentState[]): 'normal' | 'unable_to_assess' =>
  equipment.some((e) => e.condition === 'unknown') ? 'unable_to_assess' : 'normal'
