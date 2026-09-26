/**
 * Решение на точке предъявления и пересмотр решения, принятого до новых данных
 * (FR-19, FR-56, FR-32, Д-81; UI-28). Что показывать и какие действия доступны —
 * только из ответа сервера (`nonconformity.presentation.read`): действия с
 * доступностью для вошедшего, основанием и последствиями, рекомендация отдельно
 * от политики. Здесь — сборка команды из выбранного действия и сводка для окна
 * подписи; ничего не вычисляется за сервер.
 */
import type { SummaryField } from '@/entities/document'
import type { DsseEnvelope, NCPresentationAction, NCPresentationView, ResolvePresentation, ReviewPresentation } from '@/shared/api/generated/model'

export type { NCPresentationAction, NCPresentationView } from '@/shared/api/generated/model'

/** Сведения команды от сеанса и клиента (AD-7, AD-39, AD-15). */
export interface PresentationMeta {
  command_id: string
  policy_seq: number
  workplace_id?: string
  signature?: DsseEnvelope
}

/** Команда по выбранному действию: решение на точке или пересмотр. */
export type PresentationCommand =
  | { kind: 'resolve'; item_id: string; body: ResolvePresentation }
  | { kind: 'review'; item_id: string; body: ReviewPresentation }

/** Тип записи журнала, которую пишет команда (для подписи уровня 2). */
export const presentationEventType = (a: Pick<NCPresentationAction, 'operation'>): string =>
  a.operation === 'nonconformity.presentation.review' ? 'decision.presentation.reviewed' : 'decision.presentation.resolved'

/** Нужна ли причина: у пересмотра основание обязательно (контракт), у решения на точке — по желанию. */
export const reasonRequired = (a: Pick<NCPresentationAction, 'operation'>): boolean => a.operation === 'nonconformity.presentation.review'

/**
 * Тело команды из выбранного действия и ответа чтения. `basis_seq` — из ответа
 * (AD-39: сервер отклонит, если объект изменился после него).
 * @param action — действие из `view.actions` (allowed)
 * @param view — ответ `nonconformity.presentation.read`
 * @param meta — сведения команды
 * @param reason — основание словами
 */
export function buildPresentationCommand(action: NCPresentationAction, view: NCPresentationView, meta: PresentationMeta, reason: string): PresentationCommand {
  const common = {
    basis_seq: view.basis_seq,
    command_id: meta.command_id,
    policy_seq: meta.policy_seq,
    ...(meta.workplace_id ? { workplace_id: meta.workplace_id } : {}),
    ...(meta.signature ? { signature: meta.signature } : {}),
  }
  const text = reason.trim()
  if (action.operation === 'nonconformity.presentation.review') {
    const review = view.review
    if (!review || !action.outcome) throw new Error('review: нет пересматриваемого решения или исхода')
    return {
      kind: 'review',
      item_id: view.item_id,
      body: { ...common, outcome: action.outcome, reviewed_event_id: review.decision.event_id, new_fact_ids: review.new_facts.map((f) => f.event_id), reason: { text } },
    }
  }
  if (!action.resolution) throw new Error('resolve: нет исхода решения')
  const p = view.presentation
  return {
    kind: 'resolve',
    item_id: view.item_id,
    body: {
      ...common,
      step_key: p.step_key,
      closing_point: p.closing_point,
      presentation_no: p.presentation_no,
      method_event_ids: p.method_event_ids,
      resolution: action.resolution,
      ...(text ? { reason: { text } } : {}),
    },
  }
}

/**
 * Сводка для окна подписи уровня 2 (FR-66): действие, изделие, точка, основание,
 * последствия (первые), уровень.
 */
export function presentationSummary(action: NCPresentationAction, view: NCPresentationView, reason: string): SummaryField[] {
  const fields: SummaryField[] = [
    { labelKey: 'widgets.signing.fields.action', value: action.label },
    { labelKey: 'common.words.item', value: view.item_label },
    { labelKey: 'widgets.presentation.point', value: view.presentation.closing_point_label ?? view.presentation.closing_point },
  ]
  if (reason.trim()) fields.push({ labelKey: 'common.words.basis', value: reason.trim() })
  if (action.consequences.length) fields.push({ labelKey: 'widgets.presentation.consequences', value: action.consequences.join('; ') })
  fields.push({ labelKey: 'widgets.signing.fields.level', valueKey: 'decisions.signature.level2' })
  return fields
}
